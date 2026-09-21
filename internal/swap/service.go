package swap

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"crosschain/internal/bitnob"
	"crosschain/internal/config"

	"github.com/google/uuid"
)

type Store interface {
	CreateSwap(sw *Swap) error
	GetSwapByID(id string) (*Swap, error)
	GetSwapByDepositAddress(addr string) (*Swap, error)
	GetSwapByReference(ref string) (*Swap, error)
	UpdateSwapStatus(id string, status Status) error
	UpdateSwapAfterTrades(id, btcAmount, trade1ID, trade2ID string) error
	SetWithdrawalTxID(id, txID string) error
}

// Exchange is the venue the service trades on. *bitnob.Client is the real
// implementation; the demo mode substitutes a stub so the interface runs with
// no credentials.
type Exchange interface {
	GetPrices() ([]bitnob.PriceItem, error)
	ValidateAddress(address, chain string) (bool, error)
	GenerateAddress(chain, label, reference string) (*bitnob.Address, error)
	CreateQuote(req bitnob.QuoteRequest) (*bitnob.Quote, error)
	CreateOrder(req bitnob.OrderRequest) (*bitnob.Order, error)
	GetOrder(id string) (*bitnob.Order, error)
	Withdraw(req bitnob.WithdrawalRequest) (*bitnob.Withdrawal, error)
}

type Service struct {
	bitnob Exchange
	store  Store
	cfg    *config.Config
}

func NewService(exchange Exchange, store Store, cfg *config.Config) *Service {
	return &Service{bitnob: exchange, store: store, cfg: cfg}
}

// Quote estimates a swap without touching Bitnob's address or order APIs, so
// the interface can price a pair while the user is still typing.
func (s *Service) Quote(fromChain, toChain, fromAsset, toAsset string, amountIn float64) (*Estimate, error) {
	req, err := normalize(fromChain, toChain, fromAsset, toAsset, amountIn, s.cfg.MinSwapAmount)
	if err != nil {
		return nil, err
	}

	amountAfterFee := req.AmountIn - s.cfg.SwapFeeUSDT
	if amountAfterFee <= 0 {
		return nil, errors.New("amount must be greater than the swap fee")
	}

	prices, err := s.bitnob.GetPrices()
	if err != nil {
		return nil, fmt.Errorf("fetch prices: %w", err)
	}

	amountOut := s.estimateAmountOut(prices, req.FromAsset, req.ToAsset, amountAfterFee)
	rate := 0.0
	if req.AmountIn > 0 {
		rate = amountOut / req.AmountIn
	}

	return &Estimate{
		FromChain: req.FromChain,
		ToChain:   req.ToChain,
		FromAsset: req.FromAsset,
		ToAsset:   req.ToAsset,
		AmountIn:  req.AmountIn,
		AmountOut: amountOut,
		Fee:       s.cfg.SwapFeeUSDT,
		Rate:      rate,
		MinAmount: s.cfg.MinSwapAmount,
	}, nil
}

func (s *Service) CreateSwap(fromChain, toChain, fromAsset, toAsset, destAddress string, amountIn float64) (*Swap, error) {
	req, err := normalize(fromChain, toChain, fromAsset, toAsset, amountIn, s.cfg.MinSwapAmount)
	if err != nil {
		return nil, err
	}

	fromChain, toChain = req.FromChain, req.ToChain
	fromAsset, toAsset = req.FromAsset, req.ToAsset
	amountIn = req.AmountIn
	destAddress = strings.TrimSpace(destAddress)
	if destAddress == "" {
		return nil, errors.New("destination address is required")
	}

	valid, err := s.bitnob.ValidateAddress(destAddress, toChain)
	if err != nil {
		return nil, fmt.Errorf("validate destination address: %w", err)
	}
	if !valid {
		return nil, errors.New("destination address is invalid for the target chain")
	}

	prices, err := s.bitnob.GetPrices()
	if err != nil {
		return nil, fmt.Errorf("fetch prices: %w", err)
	}

	amountAfterFee := amountIn - s.cfg.SwapFeeUSDT
	if amountAfterFee <= 0 {
		return nil, errors.New("amount must be greater than the swap fee")
	}

	amountOut := s.estimateAmountOut(prices, fromAsset, toAsset, amountAfterFee)
	ref := uuid.NewString()
	addr, err := s.bitnob.GenerateAddress(fromChain, "swap-"+ref, ref)
	if err != nil {
		return nil, fmt.Errorf("generate deposit address: %w", err)
	}

	sw := &Swap{
		ID:             ref,
		FromChain:      fromChain,
		ToChain:        toChain,
		FromAsset:      fromAsset,
		ToAsset:        toAsset,
		AmountIn:       amountIn,
		AmountOut:      amountOut,
		Fee:            s.cfg.SwapFeeUSDT,
		DepositAddress: addr.Address,
		DestAddress:    destAddress,
		Status:         StatusAwaitingDeposit,
		Reference:      ref,
	}
	if err := s.store.CreateSwap(sw); err != nil {
		return nil, fmt.Errorf("save swap: %w", err)
	}

	return sw, nil
}

func (s *Service) ExecuteTrades(sw *Swap) error {
	if sw == nil {
		return errors.New("swap is nil")
	}

	if err := s.executeTrades(sw); err != nil {
		_ = s.store.UpdateSwapStatus(sw.ID, StatusFailed)
		return err
	}

	return nil
}

func (s *Service) executeTrades(sw *Swap) error {
	if err := s.store.UpdateSwapStatus(sw.ID, StatusDepositConfirmed); err != nil {
		return fmt.Errorf("mark deposit confirmed: %w", err)
	}

	amountAfterFee := sw.AmountIn - sw.Fee
	if amountAfterFee <= 0 {
		return errors.New("amount after fee is not positive")
	}

	if sw.FromAsset == sw.ToAsset {
		amount := amountToUnits(amountAfterFee, sw.ToAsset)
		return s.withdraw(sw, amount)
	}

	prices, err := s.bitnob.GetPrices()
	if err != nil {
		return fmt.Errorf("fetch prices for routing: %w", err)
	}

	if route, ok := directRoute(prices, sw.FromAsset, sw.ToAsset); ok {
		receiveAmount, orderID, err := s.executeDirectTrade(sw, route, amountAfterFee)
		if err != nil {
			return err
		}
		if err := s.store.UpdateSwapAfterTrades(sw.ID, "", orderID, ""); err != nil {
			return fmt.Errorf("save direct trade details: %w", err)
		}
		if err := s.store.UpdateSwapStatus(sw.ID, StatusTrade2Done); err != nil {
			return fmt.Errorf("mark direct trade done: %w", err)
		}

		return s.withdraw(sw, receiveAmount)
	}

	receiveAmount, btcAmount, trade1ID, trade2ID, err := s.executeBTCPath(sw, prices, amountAfterFee)
	if err != nil {
		return err
	}
	if err := s.store.UpdateSwapAfterTrades(sw.ID, btcAmount, trade1ID, trade2ID); err != nil {
		return fmt.Errorf("save routed trade details: %w", err)
	}

	return s.withdraw(sw, receiveAmount)
}

func (s *Service) executeDirectTrade(sw *Swap, route tradeRoute, amountAfterFee float64) (string, string, error) {
	req := quoteRequestForRoute(route, amountAfterFee, sw.ToAsset)
	quote, order, err := s.quoteAndOrder(req)
	if err != nil {
		return "", "", fmt.Errorf("execute direct trade: %w", err)
	}
	if err := s.pollFilled(order.ID); err != nil {
		return "", "", fmt.Errorf("poll direct trade: %w", err)
	}

	return quote.Exchange.ReceiveQuantity, order.ID, nil
}

func (s *Service) executeBTCPath(sw *Swap, prices []bitnob.PriceItem, amountAfterFee float64) (string, string, string, string, error) {
	btcPrice, ok := priceFor(prices, "BTC", sw.FromAsset)
	if !ok || btcPrice <= 0 {
		return "", "", "", "", fmt.Errorf("missing BTC/%s price", sw.FromAsset)
	}

	btcQty := amountAfterFee / btcPrice
	trade1Req := bitnob.QuoteRequest{
		BaseCurrency:  "BTC",
		QuoteCurrency: sw.FromAsset,
		Side:          "buy",
		Quantity:      amountToUnits(btcQty, "BTC"),
	}
	trade1Quote, trade1Order, err := s.quoteAndOrder(trade1Req)
	if err != nil {
		return "", "", "", "", fmt.Errorf("execute BTC buy: %w", err)
	}
	if err := s.pollFilled(trade1Order.ID); err != nil {
		return "", "", "", "", fmt.Errorf("poll BTC buy: %w", err)
	}
	if err := s.store.UpdateSwapStatus(sw.ID, StatusTrade1Done); err != nil {
		return "", "", "", "", fmt.Errorf("mark trade 1 done: %w", err)
	}

	btcAmount := trade1Quote.Exchange.ReceiveQuantity
	trade2Req := bitnob.QuoteRequest{
		BaseCurrency:  "BTC",
		QuoteCurrency: sw.ToAsset,
		Side:          "sell",
		Quantity:      btcAmount,
	}
	trade2Quote, trade2Order, err := s.quoteAndOrder(trade2Req)
	if err != nil {
		return "", "", "", "", fmt.Errorf("execute BTC sell: %w", err)
	}
	if err := s.pollFilled(trade2Order.ID); err != nil {
		return "", "", "", "", fmt.Errorf("poll BTC sell: %w", err)
	}
	if err := s.store.UpdateSwapStatus(sw.ID, StatusTrade2Done); err != nil {
		return "", "", "", "", fmt.Errorf("mark trade 2 done: %w", err)
	}

	return trade2Quote.Exchange.ReceiveQuantity, btcAmount, trade1Order.ID, trade2Order.ID, nil
}

func (s *Service) quoteAndOrder(req bitnob.QuoteRequest) (*bitnob.Quote, *bitnob.Order, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		quote, err := s.bitnob.CreateQuote(req)
		if err != nil {
			return nil, nil, err
		}

		order, err := s.bitnob.CreateOrder(bitnob.OrderRequest{
			BaseCurrency:  req.BaseCurrency,
			QuoteCurrency: req.QuoteCurrency,
			Side:          req.Side,
			Quantity:      req.Quantity,
			Price:         quote.Price,
			QuoteID:       quote.ID,
		})
		if err == nil {
			return quote, order, nil
		}
		lastErr = err
		if !looksExpired(err) {
			break
		}
	}

	return nil, nil, lastErr
}

func (s *Service) pollFilled(orderID string) error {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		order, err := s.bitnob.GetOrder(orderID)
		if err != nil {
			return err
		}

		switch strings.ToLower(order.Status) {
		case "filled", "completed", "success", "successful":
			return nil
		case "failed", "cancelled", "canceled", "rejected":
			return fmt.Errorf("order %s ended with status %s", orderID, order.Status)
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("order %s was not filled before timeout", orderID)
}

func (s *Service) withdraw(sw *Swap, amount string) error {
	if err := s.store.UpdateSwapStatus(sw.ID, StatusSending); err != nil {
		return fmt.Errorf("mark sending: %w", err)
	}

	withdrawal, err := s.bitnob.Withdraw(bitnob.WithdrawalRequest{
		ToAddress:   sw.DestAddress,
		Amount:      amount,
		Currency:    sw.ToAsset,
		Chain:       sw.ToChain,
		Reference:   sw.ID + "-withdrawal",
		Description: "Cross-chain stablecoin swap",
	})
	if err != nil {
		return fmt.Errorf("withdraw output asset: %w", err)
	}

	if withdrawal.TransactionID != "" {
		if err := s.store.SetWithdrawalTxID(sw.ID, withdrawal.TransactionID); err != nil {
			return fmt.Errorf("save withdrawal tx id: %w", err)
		}
	}

	return nil
}

func (s *Service) estimateAmountOut(prices []bitnob.PriceItem, fromAsset, toAsset string, amountAfterFee float64) float64 {
	if fromAsset == toAsset {
		return amountAfterFee
	}

	if route, ok := directRoute(prices, fromAsset, toAsset); ok {
		if route.reversed {
			return (amountAfterFee / route.price) * 0.997
		}
		return (amountAfterFee * route.price) * 0.997
	}

	fromBTCPrice, okFrom := priceFor(prices, "BTC", fromAsset)
	toBTCPrice, okTo := priceFor(prices, "BTC", toAsset)
	if okFrom && okTo && toBTCPrice > 0 {
		return ((amountAfterFee / fromBTCPrice) * toBTCPrice) * 0.994
	}

	return amountAfterFee * 0.994
}

type tradeRoute struct {
	base     string
	quote    string
	side     string
	price    float64
	reversed bool
}

func directRoute(prices []bitnob.PriceItem, fromAsset, toAsset string) (tradeRoute, bool) {
	for _, item := range prices {
		base := strings.ToUpper(item.BaseCurrency)
		quote := strings.ToUpper(item.QuoteCurrency)
		price, err := strconv.ParseFloat(item.Price, 64)
		if err != nil || price <= 0 {
			continue
		}

		if base == fromAsset && quote == toAsset {
			return tradeRoute{base: base, quote: quote, side: "sell", price: price}, true
		}
		if base == toAsset && quote == fromAsset {
			return tradeRoute{base: base, quote: quote, side: "buy", price: price, reversed: true}, true
		}
	}

	return tradeRoute{}, false
}

func quoteRequestForRoute(route tradeRoute, amountAfterFee float64, toAsset string) bitnob.QuoteRequest {
	quantityAsset := route.base
	quantity := amountAfterFee
	if route.reversed {
		quantity = amountAfterFee / route.price
		quantityAsset = toAsset
	}

	return bitnob.QuoteRequest{
		BaseCurrency:  route.base,
		QuoteCurrency: route.quote,
		Side:          route.side,
		Quantity:      amountToUnits(quantity, quantityAsset),
	}
}

func priceFor(prices []bitnob.PriceItem, baseCurrency, quoteCurrency string) (float64, bool) {
	for _, item := range prices {
		if strings.EqualFold(item.BaseCurrency, baseCurrency) && strings.EqualFold(item.QuoteCurrency, quoteCurrency) {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				return price, true
			}
		}
	}

	return 0, false
}

func amountToUnits(amount float64, asset string) string {
	decimals := 6
	if strings.EqualFold(asset, "BTC") {
		decimals = 8
	}

	multiplier := math.Pow10(decimals)
	units := math.Round(amount * multiplier)
	if units < 0 {
		units = 0
	}

	return strconv.FormatInt(int64(units), 10)
}

func looksExpired(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "expired") || strings.Contains(message, "quote")
}

// normalize trims and validates a swap request against the asset catalog. Both
// the quote and the create path go through it so an estimate can never be
// priced on a pair that create would reject.
func normalize(fromChain, toChain, fromAsset, toAsset string, amountIn, minAmount float64) (*Estimate, error) {
	fromChain = strings.ToLower(strings.TrimSpace(fromChain))
	toChain = strings.ToLower(strings.TrimSpace(toChain))
	fromAsset = strings.ToUpper(strings.TrimSpace(fromAsset))
	toAsset = strings.ToUpper(strings.TrimSpace(toAsset))

	if amountIn < minAmount {
		return nil, fmt.Errorf("minimum swap amount is %.2f", minAmount)
	}
	if !supports(fromAsset, fromChain) {
		return nil, fmt.Errorf("%s is not supported on %s", fromAsset, fromChain)
	}
	if !supports(toAsset, toChain) {
		return nil, fmt.Errorf("%s is not supported on %s", toAsset, toChain)
	}
	if fromChain == toChain && fromAsset == toAsset {
		return nil, errors.New("source and destination are the same")
	}

	return &Estimate{
		FromChain: fromChain,
		ToChain:   toChain,
		FromAsset: fromAsset,
		ToAsset:   toAsset,
		AmountIn:  amountIn,
	}, nil
}
