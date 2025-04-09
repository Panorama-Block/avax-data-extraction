package api

import (
    "encoding/json"
    "fmt"
    "log"
    "net/url"
    "strconv"

    "github.com/Panorama-Block/avax/internal/types"
)

func (c *Client) GetChainMetric(chainID, metric string) (int64, error) {
    endpoint := fmt.Sprintf("/v2/chains/%s/metrics/%s", chainID, metric)
    body, err := c.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar métrica %s da chain %s: %v", metric, chainID, err)
        return 0, err
    }
    var result struct {
        Result struct {
            Value int64 `json:"value"`
        } `json:"result"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando métrica chain: %v", err)
        return 0, err
    }
    return result.Result.Value, nil
}

func (c *Client) GetTeleporterMetric(chainID, metric string) (int64, error) {
    endpoint := fmt.Sprintf("/v2/chains/%s/teleporterMetrics/%s", chainID, metric)
    body, err := c.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar teleporter metric %s: %v", metric, err)
        return 0, err
    }
    var result struct {
        Result struct {
            Value int64 `json:"value"`
        } `json:"result"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando teleporter metric: %v", err)
        return 0, err
    }
    return result.Result.Value, nil
}

func (c *Client) GetStakingMetrics(networkOrChainID string) (*types.StakingMetrics, error) {
    endpoint := fmt.Sprintf("/v2/chains/%s/metrics/staking", networkOrChainID)
    body, err := c.makeRequest(endpoint)
    if err != nil {
        log.Printf("Erro ao buscar staking metrics da chain %s: %v", networkOrChainID, err)
        return nil, err
    }
    var result struct {
        Result types.StakingMetrics `json:"result"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        log.Printf("Erro decodificando staking metrics: %v", err)
        return nil, err
    }
    return &result.Result, nil
}

type MetricsAPI struct {
    client *Client
}

func NewMetricsAPI(client *Client) *MetricsAPI {
    return &MetricsAPI{
        client: client,
    }
}

type MetricDataPoint struct {
    Timestamp int64       `json:"timestamp"`
    Value     json.Number `json:"value"`
}

type MetricsResponse struct {
    Results []MetricDataPoint `json:"results"`
    NextPageToken string `json:"nextPageToken,omitempty"`
}

type TimeInterval string

const (
    TimeIntervalHour TimeInterval = "hour"
    TimeIntervalDay  TimeInterval = "day"
    TimeIntervalWeek TimeInterval = "week"
    TimeIntervalMonth TimeInterval = "month"
)

type MetricsParams struct {
    StartTimestamp int64
    EndTimestamp   int64
    TimeInterval   TimeInterval
    PageSize       int
    PageToken      string
}

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

type EVMChainMetric string

const (
    ActiveAddresses    EVMChainMetric = "activeAddresses"
    ActiveSenders      EVMChainMetric = "activeSenders"
    TxCount            EVMChainMetric = "txCount"
    
    CumulativeAddresses  EVMChainMetric = "cumulativeAddresses"
    CumulativeTxCount    EVMChainMetric = "cumulativeTxCount"
    CumulativeContracts  EVMChainMetric = "cumulativeContracts"
    CumulativeDeployers  EVMChainMetric = "cumulativeDeployers"
    
    GasUsed            EVMChainMetric = "gasUsed"
    AvgGps             EVMChainMetric = "avgGps"
    MaxGps             EVMChainMetric = "maxGps"
    AvgTps             EVMChainMetric = "avgTps"
    MaxTps             EVMChainMetric = "maxTps"
    AvgGasPrice        EVMChainMetric = "avgGasPrice"
    MaxGasPrice        EVMChainMetric = "maxGasPrice"
    FeesPaid           EVMChainMetric = "feesPaid"
)

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
        return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
    }
    
    return &response, nil
}
