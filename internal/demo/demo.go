// Package demo runs the swap interface with no Bitnob credentials, no
// database, and no webhook tunnel. It stubs the exchange, keeps orders in
// memory, and walks each order through the real status machine on a timer so
// the interface can be developed and reviewed end to end.
package demo

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"crosschain/internal/bitnob"
	"crosschain/internal/swap"
)

// prices are plausible stand-ins for Bitnob's live board.
var prices = []bitnob.PriceItem{
	{BaseCurrency: "BTC", QuoteCurrency: "USDT", Price: "64250.00"},
	{BaseCurrency: "BTC", QuoteCurrency: "USDC", Price: "64190.00"},
	{BaseCurrency: "USDT", QuoteCurrency: "USDC", Price: "0.9994"},
}

// Exchange is a stub venue: quotes fill instantly at the stubbed price.
type Exchange struct{ counter int64 }

func (e *Exchange) GetPrices() ([]bitnob.PriceItem, error) { return prices, nil }

func (e *Exchange) ValidateAddress(address, chain string) (bool, error) {
	return len(strings.TrimSpace(address)) >= 8, nil
}

func (e *Exchange) GenerateAddress(chain, label, reference string) (*bitnob.Address, error) {
	return &bitnob.Address{
		ID:      e.next("addr"),
		Address: demoAddress(chain, reference),
		Chain:   chain,
		Status:  "active",
	}, nil
}

func (e *Exchange) CreateQuote(req bitnob.QuoteRequest) (*bitnob.Quote, error) {
	price := lookup(req.BaseCurrency, req.QuoteCurrency)
	quantity, _ := strconv.ParseFloat(req.Quantity, 64)
	receive := quantity * price
	if strings.EqualFold(req.Side, "buy") {
		receive = quantity
	}

	return &bitnob.Quote{
		ID:        e.next("quote"),
		Price:     strconv.FormatFloat(price, 'f', -1, 64),
		ExpiresAt: time.Now().Add(30 * time.Second).Format(time.RFC3339),
		Exchange: bitnob.QuoteExchange{
			SendQuantity:    req.Quantity,
			SendCurrency:    req.QuoteCurrency,
			ReceiveQuantity: strconv.FormatInt(int64(receive), 10),
			ReceiveCurrency: req.BaseCurrency,
		},
	}, nil
}

func (e *Exchange) CreateOrder(req bitnob.OrderRequest) (*bitnob.Order, error) {
	return &bitnob.Order{ID: e.next("order"), Status: "filled", Quantity: req.Quantity, Price: req.Price}, nil
}

func (e *Exchange) GetOrder(id string) (*bitnob.Order, error) {
	return &bitnob.Order{ID: id, Status: "filled"}, nil
}

func (e *Exchange) Withdraw(req bitnob.WithdrawalRequest) (*bitnob.Withdrawal, error) {
	return &bitnob.Withdrawal{
		TransactionID: e.next("tx"),
		Status:        "success",
		Amount:        req.Amount,
		Chain:         req.Chain,
	}, nil
}

func (e *Exchange) next(prefix string) string {
	e.counter++
	return fmt.Sprintf("demo-%s-%d", prefix, e.counter)
}

func lookup(base, quote string) float64 {
	for _, item := range prices {
		if strings.EqualFold(item.BaseCurrency, base) && strings.EqualFold(item.QuoteCurrency, quote) {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				return price
			}
		}
	}

	return 1
}

func demoAddress(chain, reference string) string {
	seed := strings.ReplaceAll(reference, "-", "")
	if len(seed) > 26 {
		seed = seed[:26]
	}

	switch chain {
	case "bitcoin":
		return "bc1q" + seed
	case "tron":
		return "T" + strings.ToUpper(seed)
	case "solana", "stellar":
		return strings.ToUpper(seed)
	default:
		return "0x" + seed
	}
}

// Store is an in-memory swap.Store. OnCreate fires after each new swap so the
// demo can pretend a deposit arrived.
type Store struct {
	mu       sync.RWMutex
	swaps    map[string]*swap.Swap
	OnCreate func(sw *swap.Swap)
}

func NewStore() *Store { return &Store{swaps: map[string]*swap.Swap{}} }

func (s *Store) CreateSwap(sw *swap.Swap) error {
	s.mu.Lock()
	copied := *sw
	copied.CreatedAt = time.Now()
	copied.UpdatedAt = copied.CreatedAt
	s.swaps[sw.ID] = &copied
	s.mu.Unlock()

	if s.OnCreate != nil {
		s.OnCreate(sw)
	}

	return nil
}

func (s *Store) GetSwapByID(id string) (*swap.Swap, error) {
	return s.find(func(sw *swap.Swap) bool { return sw.ID == id })
}

func (s *Store) GetSwapByDepositAddress(addr string) (*swap.Swap, error) {
	return s.find(func(sw *swap.Swap) bool { return sw.DepositAddress == addr })
}

func (s *Store) GetSwapByReference(ref string) (*swap.Swap, error) {
	return s.find(func(sw *swap.Swap) bool { return sw.Reference == ref })
}

func (s *Store) UpdateSwapStatus(id string, status swap.Status) error {
	return s.mutate(id, func(sw *swap.Swap) { sw.Status = status })
}

func (s *Store) UpdateSwapAfterTrades(id, btcAmount, trade1ID, trade2ID string) error {
	return s.mutate(id, func(sw *swap.Swap) {
		sw.BTCAmount = btcAmount
		sw.Trade1OrderID = trade1ID
		sw.Trade2OrderID = trade2ID
	})
}

func (s *Store) SetWithdrawalTxID(id, txID string) error {
	return s.mutate(id, func(sw *swap.Swap) { sw.WithdrawalTxID = txID })
}

func (s *Store) Close() error { return nil }

func (s *Store) find(match func(*swap.Swap) bool) (*swap.Swap, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sw := range s.swaps {
		if match(sw) {
			copied := *sw
			return &copied, nil
		}
	}

	return nil, fmt.Errorf("swap not found")
}

func (s *Store) mutate(id string, apply func(*swap.Swap)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sw, ok := s.swaps[id]
	if !ok {
		return fmt.Errorf("swap %s not found", id)
	}
	apply(sw)
	sw.UpdatedAt = time.Now()

	return nil
}

// Drive makes the demo feel live: a deposit "arrives" a few seconds after the
// order is created, then the real service walks the swap to completion.
func Drive(store *Store, service *swap.Service, depositDelay time.Duration) {
	store.OnCreate = func(sw *swap.Swap) {
		go func() {
			time.Sleep(depositDelay)
			log.Printf("demo: pretending deposit arrived for swap %s", sw.ID)
			if err := service.ExecuteTrades(sw); err != nil {
				log.Printf("demo: execute trades for %s: %v", sw.ID, err)
				return
			}
			time.Sleep(2 * time.Second)
			if err := store.UpdateSwapStatus(sw.ID, swap.StatusCompleted); err != nil {
				log.Printf("demo: complete swap %s: %v", sw.ID, err)
			}
		}()
	}
}
