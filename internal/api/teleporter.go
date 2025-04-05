package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// TeleporterAPI manages teleporter-related API operations
type TeleporterAPI struct {
	client *Client
}

// NewTeleporterAPI creates a new teleporter API client
func NewTeleporterAPI(client *Client) *TeleporterAPI {
	return &TeleporterAPI{client: client}
}

// GetTeleporterTxs retrieves teleporter transactions between chains
func (c *TeleporterAPI) GetTeleporterTxs(sourceChain, destChain string) ([]types.TeleporterTx, error) {
	endpoint := fmt.Sprintf("/teleporter/transactions?sourceChain=%s&destChain=%s", sourceChain, destChain)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching teleporter transactions: %v", err)
		return nil, err
	}
	var result struct {
		Transactions []types.TeleporterTx `json:"transactions"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding teleporter transactions: %v", err)
		return nil, err
	}
	return result.Transactions, nil
} 