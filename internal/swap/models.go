package swap

import "time"

type Status string

const (
	StatusAwaitingDeposit  Status = "awaiting_deposit"
	StatusDepositConfirmed Status = "deposit_confirmed"
	StatusTrade1Done       Status = "trade1_done"
	StatusTrade2Done       Status = "trade2_done"
	StatusSending          Status = "sending"
	StatusCompleted        Status = "completed"
	StatusFailed           Status = "failed"
)

type Swap struct {
	ID             string
	FromChain      string
	ToChain        string
	FromAsset      string
	ToAsset        string
	AmountIn       float64
	AmountOut      float64
	Fee            float64
	DepositAddress string
	DestAddress    string
	Status         Status
	BTCAmount      string
	Trade1OrderID  string
	Trade2OrderID  string
	WithdrawalTxID string
	Reference      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Estimate is a priced swap the user has not committed to yet.
type Estimate struct {
	FromChain string  `json:"from_chain"`
	ToChain   string  `json:"to_chain"`
	FromAsset string  `json:"from_asset"`
	ToAsset   string  `json:"to_asset"`
	AmountIn  float64 `json:"amount_in"`
	AmountOut float64 `json:"amount_out"`
	Fee       float64 `json:"fee"`
	Rate      float64 `json:"rate"`
	MinAmount float64 `json:"min_amount"`
}
