package bitnob

import "net/http"

type Address struct {
	ID      string `json:"id"`
	Address string `json:"address"`
	Chain   string `json:"chain"`
	Status  string `json:"status"`
}

type ChainConfig struct {
	Chain    string `json:"chain"`
	Decimals int    `json:"decimals"`
}

func (c *Client) GenerateAddress(chain, label, reference string) (*Address, error) {
	req := map[string]string{
		"chain":     chain,
		"label":     label,
		"reference": reference,
	}

	var addr Address
	if err := c.Do(http.MethodPost, "/api/addresses", req, &addr); err != nil {
		return nil, err
	}

	return &addr, nil
}

func (c *Client) ValidateAddress(address, chain string) (bool, error) {
	req := map[string]string{
		"address": address,
		"chain":   chain,
	}

	var resp struct {
		Valid bool `json:"valid"`
	}
	if err := c.Do(http.MethodPost, "/api/addresses/validate", req, &resp); err != nil {
		return false, err
	}

	return resp.Valid, nil
}

func (c *Client) GetSupportedChains() ([]ChainConfig, error) {
	var chains []ChainConfig
	if err := c.Do(http.MethodGet, "/api/addresses/stablecoins/chains", nil, &chains); err != nil {
		return nil, err
	}

	return chains, nil
}
