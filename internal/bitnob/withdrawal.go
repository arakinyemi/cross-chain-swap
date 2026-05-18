package bitnob

import "net/http"

type WithdrawalRequest struct {
	ToAddress   string `json:"to_address"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Chain       string `json:"chain"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
}

type Withdrawal struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Amount        string `json:"amount"`
	Chain         string `json:"chain"`
}

func (c *Client) Withdraw(req WithdrawalRequest) (*Withdrawal, error) {
	var withdrawal Withdrawal
	if err := c.Do(http.MethodPost, "/api/withdrawal", req, &withdrawal); err != nil {
		return nil, err
	}

	return &withdrawal, nil
}
