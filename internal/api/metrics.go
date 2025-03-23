package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// MetricsAPI handles metrics-related API calls to AvaCloud
type MetricsAPI struct {
	client *Client
}

// NewMetricsAPI creates a new Metrics API client
func NewMetricsAPI(client *Client) *MetricsAPI {
	return &MetricsAPI{client: client}
}

// MetricDataPoint represents a single data point in a time series
type MetricDataPoint struct {
	Timestamp int64       `json:"timestamp"`
	Value     json.Number `json:"value"`
}

// MetricsResponse is the general response for metrics endpoints
type MetricsResponse struct {
	Results []MetricDataPoint `json:"results"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// TimeInterval represents the granularity of metrics data
type TimeInterval string

const (
	TimeIntervalHour TimeInterval = "hour"
	TimeIntervalDay  TimeInterval = "day"
	TimeIntervalWeek TimeInterval = "week"
	TimeIntervalMonth TimeInterval = "month"
)

// MetricsParams represents parameters for metrics API requests
type MetricsParams struct {
	StartTimestamp int64
	EndTimestamp   int64
	TimeInterval   TimeInterval
	PageSize       int
	PageToken      string
}

// ToQueryParams converts MetricsParams to URL query parameters
func (p *MetricsParams) ToQueryParams() url.Values {
	params := url.Values{}
	if p.StartTimestamp > 0 {
		params.Add("startTimestamp", strconv.FormatInt(p.StartTimestamp, 10))
	}
	if p.EndTimestamp > 0 {
		params.Add("endTimestamp", strconv.FormatInt(p.EndTimestamp, 10))
	}
	if p.TimeInterval != "" {
		params.Add("timeInterval", string(p.TimeInterval))
	}
	if p.PageSize > 0 {
		params.Add("pageSize", strconv.Itoa(p.PageSize))
	}
	if p.PageToken != "" {
		params.Add("pageToken", p.PageToken)
	}
	return params
}

// EVMChainMetric represents the available metrics for EVM chains
type EVMChainMetric string

const (
	// Chain Activity Metrics
	ActiveAddresses    EVMChainMetric = "activeAddresses"
	ActiveSenders      EVMChainMetric = "activeSenders"
	TxCount            EVMChainMetric = "txCount"
	
	// Cumulative Metrics
	CumulativeAddresses  EVMChainMetric = "cumulativeAddresses"
	CumulativeTxCount    EVMChainMetric = "cumulativeTxCount"
	CumulativeContracts  EVMChainMetric = "cumulativeContracts"
	CumulativeDeployers  EVMChainMetric = "cumulativeDeployers"
	
	// Gas and Performance Metrics
	GasUsed            EVMChainMetric = "gasUsed"
	AvgGps             EVMChainMetric = "avgGps"
	MaxGps             EVMChainMetric = "maxGps"
	AvgTps             EVMChainMetric = "avgTps"
	MaxTps             EVMChainMetric = "maxTps"
	AvgGasPrice        EVMChainMetric = "avgGasPrice"
	MaxGasPrice        EVMChainMetric = "maxGasPrice"
	FeesPaid           EVMChainMetric = "feesPaid"
)

// GetEVMChainMetrics retrieves metrics for an EVM-compatible chain
func (m *MetricsAPI) GetEVMChainMetrics(chainID string, metric EVMChainMetric, params *MetricsParams) (*MetricsResponse, error) {
	queryParams := url.Values{}
	if params != nil {
		queryParams = params.ToQueryParams()
	}
	
	respBody, err := m.client.SendRequest("GET", fmt.Sprintf("/v2/chains/%s/metrics/%s", chainID, metric), queryParams, nil)
	if err != nil {
		return nil, err
	}
	
	var response MetricsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// StakingMetric represents the available staking metrics
type StakingMetric string

const (
	ValidatorCount  StakingMetric = "validatorCount"
	DelegatorCount  StakingMetric = "delegatorCount"
	ValidatorWeight StakingMetric = "validatorWeight"
	DelegatorWeight StakingMetric = "delegatorWeight"
)

// GetStakingMetrics retrieves staking metrics for a network/subnet
func (m *MetricsAPI) GetStakingMetrics(network string, metric StakingMetric, subnetID string, params *MetricsParams) (*MetricsResponse, error) {
	queryParams := url.Values{}
	if subnetID != "" {
		queryParams.Add("subnetId", subnetID)
	}
	
	if params != nil {
		for k, v := range params.ToQueryParams() {
			for _, value := range v {
				queryParams.Add(k, value)
			}
		}
	}
	
	respBody, err := m.client.SendRequest("GET", fmt.Sprintf("/v2/networks/%s/metrics/%s", network, metric), queryParams, nil)
	if err != nil {
		return nil, err
	}
	
	var response MetricsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetTeleporterMetrics retrieves metrics related to the Teleporter (BTC bridge)
func (m *MetricsAPI) GetTeleporterMetrics(metric string, params *MetricsParams) (*MetricsResponse, error) {
	queryParams := url.Values{}
	if params != nil {
		queryParams = params.ToQueryParams()
	}
	
	respBody, err := m.client.SendRequest("GET", fmt.Sprintf("/v2/chains/teleporter/metrics/%s", metric), queryParams, nil)
	if err != nil {
		return nil, err
	}
	
	var response MetricsResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// AddressesByBalanceResponse is the response for addresses by balance endpoint
type AddressesByBalanceResponse struct {
	Results []struct {
		Timestamp int64 `json:"timestamp"`
		Count     int   `json:"count"`
	} `json:"results"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// GetAddressesByBalance retrieves the count of addresses with balance above a threshold
func (m *MetricsAPI) GetAddressesByBalance(chainID string, balanceThreshold string, params *MetricsParams) (*AddressesByBalanceResponse, error) {
	queryParams := url.Values{}
	queryParams.Add("balance", balanceThreshold)
	
	if params != nil {
		for k, v := range params.ToQueryParams() {
			for _, value := range v {
				queryParams.Add(k, value)
			}
		}
	}
	
	respBody, err := m.client.SendRequest("GET", fmt.Sprintf("/v2/chains/%s/metrics/addressesByBalance", chainID), queryParams, nil)
	if err != nil {
		return nil, err
	}
	
	var response AddressesByBalanceResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// BtcBalanceResponse is the response for addresses by BTC bridge balance
type BtcBalanceResponse struct {
	Results []struct {
		Address string `json:"address"`
		Balance string `json:"balance"`
	} `json:"results"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// GetAddressesByBtcBalance retrieves addresses with BTC.b balance above a threshold
func (m *MetricsAPI) GetAddressesByBtcBalance(balanceThreshold string, params *MetricsParams) (*BtcBalanceResponse, error) {
	queryParams := url.Values{}
	queryParams.Add("balance", balanceThreshold)
	
	if params != nil {
		for k, v := range params.ToQueryParams() {
			for _, value := range v {
				queryParams.Add(k, value)
			}
		}
	}
	
	respBody, err := m.client.SendRequest("GET", fmt.Sprintf("/v2/chains/metrics/addressesByBtcbBridgedBalance"), queryParams, nil)
	if err != nil {
		return nil, err
	}
	
	var response BtcBalanceResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// ValidatorAddress represents a validator node and its associated address
type ValidatorAddress struct {
	NodeID  string `json:"nodeId"`
	Address string `json:"address"`
}

// ValidatorAddressesResponse is the response for the validator addresses endpoint
type ValidatorAddressesResponse struct {
	Validators []ValidatorAddress `json:"validators"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// GetValidatorAddresses retrieves addresses running validators during a timeframe
func (m *MetricsAPI) GetValidatorAddresses(network string, startTime, endTime int64, pagination *PaginationParams) (*ValidatorAddressesResponse, error) {
	params := url.Values{}
	
	if startTime > 0 {
		params.Add("startTimestamp", strconv.FormatInt(startTime, 10))
	}
	
	if endTime > 0 {
		params.Add("endTimestamp", strconv.FormatInt(endTime, 10))
	}
	
	if pagination != nil {
		for k, v := range pagination.ToQueryParams() {
			for _, value := range v {
				params.Add(k, value)
			}
		}
	}
	
	respBody, err := m.client.SendRequest("GET", fmt.Sprintf("/v2/networks/%s/metrics/validatorAddresses", network), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response ValidatorAddressesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}