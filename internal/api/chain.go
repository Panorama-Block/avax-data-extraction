package api

import (
	"encoding/json"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// GetChains busca todas as chains disponíveis na API da AvaCloud
func (c *Client) GetChains() ([]types.Chain, error) {
	body, err := c.makeRequest("/chains")
	if err != nil {
		log.Println("Erro ao buscar chains:", err)
		return nil, err
	}

	var response struct {
		Chains []types.Chain `json:"chains"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Erro ao decodificar resposta:", err)
		return nil, err
	}

	return response.Chains, nil
}

// GetChainByID busca detalhes de uma chain específica
func (c *Client) GetChainByID(chainID string) (*types.Chain, error) {
	body, err := c.makeRequest("/chains/" + chainID)
	if err != nil {
		log.Printf("Erro ao buscar chain %s: %v", chainID, err)
		return nil, err
	}

	var chain types.Chain
	err = json.Unmarshal(body, &chain)
	if err != nil {
		log.Printf("Erro ao decodificar resposta da chain %s: %v", chainID, err)
		return nil, err
	}

	return &chain, nil
}
