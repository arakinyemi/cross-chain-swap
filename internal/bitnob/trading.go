package bitnob

import (
	"fmt"
	"net/http"
)

type PriceItem struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Price         string `json:"price"`
}

type QuoteRequest struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
}

type Quote struct {
	ID        string        `json:"id"`
	Price     string        `json:"price"`
	ExpiresAt string        `json:"expires_at"`
	Exchange  QuoteExchange `json:"exchange"`
}

type QuoteExchange struct {
	SendQuantity    string `json:"send_quantity"`
	SendCurrency    string `json:"send_currency"`
	ReceiveQuantity string `json:"receive_quantity"`
	ReceiveCurrency string `json:"receive_currency"`
}

type OrderRequest struct {
	BaseCurrency  string `json:"base_currency"`
	QuoteCurrency string `json:"quote_currency"`
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
	Price         string `json:"price"`
	QuoteID       string `json:"quote_id"`
}

type Order struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Quantity string `json:"quantity"`
	Price    string `json:"price"`
}

func (c *Client) GetPrices() ([]PriceItem, error) {
	var prices []PriceItem
	if err := c.Do(http.MethodGet, "/api/trading/prices", nil, &prices); err != nil {
		return nil, err
	}

	return prices, nil
}

func (c *Client) CreateQuote(req QuoteRequest) (*Quote, error) {
	var quote Quote
	if err := c.Do(http.MethodPost, "/api/trading/quotes", req, &quote); err != nil {
		return nil, err
	}

	return &quote, nil
}

func (c *Client) CreateOrder(req OrderRequest) (*Order, error) {
	var order Order
	if err := c.Do(http.MethodPost, "/api/trading/orders", req, &order); err != nil {
		return nil, err
	}

	return &order, nil
}

func (c *Client) GetOrder(id string) (*Order, error) {
	var order Order
	if err := c.Do(http.MethodGet, fmt.Sprintf("/api/trading/orders/%s", id), nil, &order); err != nil {
		return nil, err
	}

	return &order, nil
}
