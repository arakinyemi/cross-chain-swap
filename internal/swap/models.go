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
