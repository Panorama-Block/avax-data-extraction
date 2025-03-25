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

// MetricsAPI para gerenciar chamadas de API relacionadas a métricas
type MetricsAPI struct {
    client *Client
}

// NewMetricsAPI cria um novo cliente MetricsAPI
func NewMetricsAPI(client *Client) *MetricsAPI {
    return &MetricsAPI{
        client: client,
    }
}

// MetricDataPoint representa um único ponto de dados em uma série temporal
type MetricDataPoint struct {
    Timestamp int64       `json:"timestamp"`
    Value     json.Number `json:"value"`
}

// MetricsResponse é a resposta geral para endpoints de métricas
type MetricsResponse struct {
    Results []MetricDataPoint `json:"results"`
    NextPageToken string `json:"nextPageToken,omitempty"`
}

// TimeInterval representa a granularidade dos dados de métricas
type TimeInterval string

const (
    TimeIntervalHour TimeInterval = "hour"
    TimeIntervalDay  TimeInterval = "day"
    TimeIntervalWeek TimeInterval = "week"
    TimeIntervalMonth TimeInterval = "month"
)

// MetricsParams representa parâmetros para requisições de API de métricas
type MetricsParams struct {
    StartTimestamp int64
    EndTimestamp   int64
    TimeInterval   TimeInterval
    PageSize       int
    PageToken      string
}

// ToQueryParams converte MetricsParams para parâmetros de consulta URL
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

// EVMChainMetric representa as métricas disponíveis para chains EVM
type EVMChainMetric string

const (
    // Métricas de atividade da chain
    ActiveAddresses    EVMChainMetric = "activeAddresses"
    ActiveSenders      EVMChainMetric = "activeSenders"
    TxCount            EVMChainMetric = "txCount"
    
    // Métricas cumulativas
    CumulativeAddresses  EVMChainMetric = "cumulativeAddresses"
    CumulativeTxCount    EVMChainMetric = "cumulativeTxCount"
    CumulativeContracts  EVMChainMetric = "cumulativeContracts"
    CumulativeDeployers  EVMChainMetric = "cumulativeDeployers"
    
    // Métricas de gás e desempenho
    GasUsed            EVMChainMetric = "gasUsed"
    AvgGps             EVMChainMetric = "avgGps"
    MaxGps             EVMChainMetric = "maxGps"
    AvgTps             EVMChainMetric = "avgTps"
    MaxTps             EVMChainMetric = "maxTps"
    AvgGasPrice        EVMChainMetric = "avgGasPrice"
    MaxGasPrice        EVMChainMetric = "maxGasPrice"
    FeesPaid           EVMChainMetric = "feesPaid"
)

// GetEVMChainMetrics recupera métricas para uma chain compatível com EVM
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
