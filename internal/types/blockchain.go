package types

// Blockchain represents a blockchain in the Avalanche network
type Blockchain struct {
	BlockchainID string `json:"blockchainId"`
	VMID         string `json:"vmId"`
	SubnetID     string `json:"subnetId"`
}

// BlockchainEvent is used for publishing blockchain events
type BlockchainEvent struct {
	Type       string     `json:"type"`
	Blockchain Blockchain `json:"blockchain"`
} 