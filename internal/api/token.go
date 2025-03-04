package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// GetTokenDetails search for all token details
func (c *Client) GetTokenDetails(tokenAddress string) (*types.TokenInfo, error) {
	url := fmt.Sprintf("/v1/assets/%s", tokenAddress)

	body, err := c.makeRequest(url)
	if err != nil {
		log.Printf("Erro ao buscar detalhes do token %s: %v", tokenAddress, err)
		return nil, err
	}

	var tokenDetails types.TokenInfo
	err = json.Unmarshal(body, &tokenDetails)
	if err != nil {
		log.Printf("Erro ao decodificar resposta do token %s: %v", tokenAddress, err)
		return nil, err
	}

	return &tokenDetails, nil
}
