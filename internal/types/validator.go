package types

// Validator represents a blockchain validator
type Validator struct {
	NodeID      string `json:"nodeId"`
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
	StakeAmount string `json:"stakeAmount"`
	Uptime      int64  `json:"uptime,omitempty"`
}

// ValidatorEvent is used for publishing validator events
type ValidatorEvent struct {
	Type      string    `json:"type"`
	Validator Validator `json:"validator"`
} 