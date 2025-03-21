package api

import (
    "encoding/json"
    "fmt"
    "log"

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