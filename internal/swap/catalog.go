package swap

import "strings"

// Network is a chain a given asset can be deposited from or sent to.
type Network struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Short string `json:"short"`
}

// Asset is a coin the interface offers, with the networks it lives on.
type Asset struct {
	Symbol   string    `json:"symbol"`
	Name     string    `json:"name"`
	Networks []Network `json:"networks"`
}

var networks = map[string]Network{
	"bitcoin":  {ID: "bitcoin", Name: "Bitcoin", Short: "BTC"},
	"ethereum": {ID: "ethereum", Name: "Ethereum", Short: "ERC20"},
	"bsc":      {ID: "bsc", Name: "BNB Smart Chain", Short: "BEP20"},
	"polygon":  {ID: "polygon", Name: "Polygon", Short: "POLY"},
	"tron":     {ID: "tron", Name: "Tron", Short: "TRC20"},
	"solana":   {ID: "solana", Name: "Solana", Short: "SOL"},
	"stellar":  {ID: "stellar", Name: "Stellar", Short: "XLM"},
	"arbitrum": {ID: "arbitrum", Name: "Arbitrum One", Short: "ARB"},
}

// catalog is the single source of truth for what the interface offers and what
// the API accepts. An asset is only swappable on the networks listed here.
var catalog = []Asset{
	{
		Symbol:   "USDT",
		Name:     "Tether",
		Networks: pick("ethereum", "tron", "bsc", "polygon", "solana", "arbitrum"),
	},
	{
		Symbol:   "USDC",
		Name:     "USD Coin",
		Networks: pick("ethereum", "solana", "polygon", "bsc", "stellar", "arbitrum"),
	},
	{
		Symbol:   "BTC",
		Name:     "Bitcoin",
		Networks: pick("bitcoin"),
	},
}

func pick(ids ...string) []Network {
	out := make([]Network, 0, len(ids))
	for _, id := range ids {
		out = append(out, networks[id])
	}

	return out
}

// Catalog returns the assets and networks the interface offers.
func Catalog() []Asset {
	return catalog
}

// supports reports whether an asset can be swapped on a chain.
func supports(asset, chain string) bool {
	for _, a := range catalog {
		if !strings.EqualFold(a.Symbol, asset) {
			continue
		}
		for _, n := range a.Networks {
			if strings.EqualFold(n.ID, chain) {
				return true
			}
		}
	}

	return false
}
