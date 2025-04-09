package types

// Subnet represents a subnet in Avalanche
type Subnet struct {
	SubnetID    string       `json:"subnetId"`
	Blockchains []Blockchain `json:"blockchains"`
}

// SubnetEvent is used for publishing subnet events
type SubnetEvent struct {
	Type   string `json:"type"`
	Subnet Subnet `json:"subnet"`
} 