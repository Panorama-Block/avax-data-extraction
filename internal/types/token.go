package types

// TokenInfo represents basic token information
type TokenInfo struct {
	Address           string `json:"address"`
	Name              string `json:"name"`
	Symbol            string `json:"symbol"`
	Decimals          int    `json:"decimals"`
	ValueWithDecimals string `json:"valueWithDecimals"`
}

// TokenEvent is used for publishing token events
type TokenEvent struct {
	Type  string    `json:"type"`
	Token TokenInfo `json:"token"`
}

// Token represents a token with additional information
type Token struct {
	Address  string      `json:"address,omitempty"`
	Name     string      `json:"name"`
	Symbol   string      `json:"symbol"`
	Decimals int         `json:"decimals"`
	LogoURI  string      `json:"logoUri,omitempty"`
	Price    *TokenPrice `json:"price,omitempty"`
}

// TokenPrice represents pricing information for a token
type TokenPrice struct {
	Value        float64 `json:"value"`
	CurrencyCode string  `json:"currencyCode"`
}

// AddressInfo contains detailed information about an address
type AddressInfo struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	LogoUri  string `json:"logoUri"`
	Address  string `json:"address"`
} 