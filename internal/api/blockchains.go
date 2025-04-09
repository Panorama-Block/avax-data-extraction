package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// BlockchainsAPI manages blockchain-related API operations
type BlockchainsAPI struct {
	client *Client
}

// NewBlockchainsAPI creates a new blockchains API client
func NewBlockchainsAPI(client *Client) *BlockchainsAPI {
	return &BlockchainsAPI{client: client}
}

// GetBlockchains retrieves all blockchains for a network
func (c *BlockchainsAPI) GetBlockchains(network string) ([]types.Blockchain, error) {
	endpoint := fmt.Sprintf("/networks/%s/blockchains", network)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching blockchains for network %s: %v", network, err)
		return nil, err
	}
	var result struct {
		Blockchains []types.Blockchain `json:"blockchains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding blockchains: %v", err)
		return nil, err
	}
	return result.Blockchains, nil
} 