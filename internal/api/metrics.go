package api

import (
	"encoding/json"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// GetValidatorDetails search for all validators metrics
func (c *Client) GetValidatorDetails() (*types.ValidatorMetrics, error) {
	body, err := c.makeRequest("/v2/chains/P/metrics/validatorCount")
	if err != nil {
		log.Printf("Erro ao buscar métricas de validadores: %v", err)
		return nil, err
	}

	var validatorMetrics types.ValidatorMetrics
	err = json.Unmarshal(body, &validatorMetrics)
	if err != nil {
		log.Printf("Erro ao decodificar métricas de validadores: %v", err)
		return nil, err
	}

	return &validatorMetrics, nil
}
