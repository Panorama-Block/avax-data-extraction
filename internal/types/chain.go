package types

// Chain represents a blockchain
type Chain struct {
	ChainID         string   `json:"chainId"`
	Status          string   `json:"status"`
	ChainName       string   `json:"chainName"`
	Description     string   `json:"description"`
	PlatformChainId string   `json:"platformChainId"`
	SubnetId        string   `json:"subnetId"`
	VmId            string   `json:"vmId"`
	VmName          string   `json:"vmName"`
	ExplorerURL     string   `json:"explorerUrl"`
	RpcURL          string   `json:"rpcUrl"`
	WsURL           string   `json:"wsUrl"`
	IsTestnet       bool     `json:"isTestnet"`
	Private         bool     `json:"private"`
	ChainLogoUri    string   `json:"chainLogoUri"`
	EnabledFeatures []string `json:"enabledFeatures"`
	NetworkToken    struct {
		Name        string `json:"name"`
		Symbol      string `json:"symbol"`
		Decimals    int    `json:"decimals"`
		LogoUri     string `json:"logoUri"`
		Description string `json:"description"`
	} `json:"networkToken"`
}

// ChainEvent is used for publishing chain events
type ChainEvent struct {
	Type  string `json:"type"`
	Chain Chain  `json:"chain"`
} 