package api

import (
	"encoding/json"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// ChainsAPI manages chain-related API operations
type ChainsAPI struct {
	client *Client
}

// NewChainsAPI creates a new chain API client
func NewChainsAPI(client *Client) *ChainsAPI {
	return &ChainsAPI{client: client}
}

// GetChains retrieves all chains
func (c *ChainsAPI) GetChains() ([]types.Chain, error) {
	body, err := c.client.makeRequest("/chains")
	if err != nil {
		log.Println("Error fetching chains:", err)
		return nil, err
	}
	var response struct {
		Chains []types.Chain `json:"chains"`
	}
	if err = json.Unmarshal(body, &response); err != nil {
		log.Println("Error decoding /chains:", err)
		return nil, err
	}
	return response.Chains, nil
}

// GetChainByID retrieves a specific chain by ID
func (c *ChainsAPI) GetChainByID(chainID string) (*types.Chain, error) {
	body, err := c.client.makeRequest("/chains/" + chainID)
	if err != nil {
		log.Printf("Error fetching chain %s: %v", chainID, err)
		return nil, err
	}
	var chain types.Chain
	if err = json.Unmarshal(body, &chain); err != nil {
		log.Printf("Error decoding chain %s: %v", chainID, err)
		return nil, err
	}
	return &chain, nil
} 