package types

// Delegator represents a delegator staking with a validator
type Delegator struct {
	Delegator     string `json:"delegator"`
	NodeID        string `json:"nodeId"`
	StakeAmount   string `json:"stakeAmount"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	RewardAddress string `json:"rewardAddress"`
}

// DelegatorEvent is used for publishing delegator events
type DelegatorEvent struct {
	Type      string    `json:"type"`
	Delegator Delegator `json:"delegator"`
} 