package types

import "time"

// StakingMetrics represents metrics related to staking
type StakingMetrics struct {
	ValidatorCount int     `json:"validatorCount"`
	DelegatorCount int     `json:"delegatorCount"`
	TotalStaked    float64 `json:"totalStaked"`
}

// ActivityMetrics represents user activity metrics
type ActivityMetrics struct {
	ChainID         string    `json:"chainId"`
	Timestamp       time.Time `json:"timestamp"`
	ActiveAddresses int64     `json:"activeAddresses"`
	ActiveSenders   int64     `json:"activeSenders"`
	TxCount         int64     `json:"txCount"`
	NewAddresses    int64     `json:"newAddresses"`
	UniqueContracts int64     `json:"uniqueContracts"`
}

// PerformanceMetrics represents chain performance metrics
type PerformanceMetrics struct {
	ChainID    string    `json:"chainId"`
	Timestamp  time.Time `json:"timestamp"`
	AvgTPS     float64   `json:"avgTps"`
	MaxTPS     float64   `json:"maxTps"`
	AvgGPS     float64   `json:"avgGps"`
	MaxGPS     float64   `json:"maxGps"`
	BlockTime  float64   `json:"blockTime"`
	AvgLatency float64   `json:"avgLatency"`
}

// GasMetrics represents gas usage metrics
type GasMetrics struct {
	ChainID      string    `json:"chainId"`
	Timestamp    time.Time `json:"timestamp"`
	GasUsed      int64     `json:"gasUsed"`
	AvgGasPrice  float64   `json:"avgGasPrice"`
	MaxGasPrice  float64   `json:"maxGasPrice"`
	FeesPaid     float64   `json:"feesPaid"`
	AvgGasLimit  float64   `json:"avgGasLimit"`
	GasEfficiency float64  `json:"gasEfficiency"` // gasUsed/gasLimit ratio
}

// CumulativeMetrics represents cumulative chain metrics over time
type CumulativeMetrics struct {
	ChainID          string    `json:"chainId"`
	Timestamp        time.Time `json:"timestamp"`
	TotalTxs         int64     `json:"totalTxs"`
	TotalAddresses   int64     `json:"totalAddresses"`
	TotalContracts   int64     `json:"totalContracts"`
	TotalTokens      int64     `json:"totalTokens"`
	TotalGasUsed     int64     `json:"totalGasUsed"`
	TotalFeesCollected float64 `json:"totalFeesCollected"`
}

// MetricsEvent is used for publishing metrics events
type MetricsEvent struct {
	Type    string      `json:"type"`
	Metrics interface{} `json:"metrics"`
}

// ActivityMetricsEvent is a specialized metrics event for activity metrics
type ActivityMetricsEvent struct {
	Type    string         `json:"type"`
	Metrics ActivityMetrics `json:"metrics"`
}

// PerformanceMetricsEvent is a specialized metrics event for performance metrics
type PerformanceMetricsEvent struct {
	Type    string            `json:"type"`
	Metrics PerformanceMetrics `json:"metrics"`
}

// GasMetricsEvent is a specialized metrics event for gas metrics
type GasMetricsEvent struct {
	Type    string     `json:"type"`
	Metrics GasMetrics `json:"metrics"`
}

// CumulativeMetricsEvent is a specialized metrics event for cumulative metrics
type CumulativeMetricsEvent struct {
	Type    string            `json:"type"`
	Metrics CumulativeMetrics `json:"metrics"`
} 