package extractor

import (
    "encoding/json"
    "log"
    "time"

    "github.com/Panorama-Block/avax/internal/api"
    "github.com/Panorama-Block/avax/internal/kafka"
)

func StartMetricsPipeline(client *api.Client, producer *kafka.Producer, interval time.Duration, tokenAddresses []string) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for range ticker.C {
        sm, err := client.GetStakingMetrics("mainnet")
        if err != nil {
            log.Printf("Erro ao obter métricas de staking: %v", err)
        } else {
            msg := map[string]interface{}{
                "type":           "staking",
                "validatorCount": sm.ValidatorCount,
                "delegatorCount": sm.DelegatorCount,
                "totalStaked":    sm.TotalStaked,
                "timestamp":      time.Now().Unix(),
            }
            data, _ := json.Marshal(msg)
            producer.PublishMetrics(data)
        }

        for _, addr := range tokenAddresses {
            tokenDetails, err := client.GetTokenDetails(addr)
            if err != nil {
                log.Printf("Erro ao obter detalhes do token %s: %v", addr, err)
                continue
            }
            msg := map[string]interface{}{
                "type":         "token",
                "address":      addr,
                "name":         tokenDetails.Name,
                "symbol":       tokenDetails.Symbol,
                "decimals":     tokenDetails.Decimals,
                "valueWithDec": tokenDetails.ValueWithDecimals,
                "time":         time.Now().Unix(),
            }
            data, _ := json.Marshal(msg)
            producer.PublishMetrics(data)
        }
    }
}
