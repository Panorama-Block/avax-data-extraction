package api

import (
	"encoding/json"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// GetTransaction obtém detalhes completos de uma transação
func (c *Client) GetTransaction(txHash string) (*types.Transaction, error) {
	body, err := c.makeRequest("/transactions/" + txHash)
	if err != nil {
		log.Printf("Erro ao buscar transação %s: %v", txHash, err)
		return nil, err
	}

	var tx types.Transaction
	err = json.Unmarshal(body, &tx)
	if err != nil {
		log.Printf("Erro ao decodificar transação %s: %v", txHash, err)
		return nil, err
	}

	return &tx, nil
}
