package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// DelegatorsAPI manages delegator-related API operations
type DelegatorsAPI struct {
	client *Client
}

// NewDelegatorsAPI creates a new delegators API client
func NewDelegatorsAPI(client *Client) *DelegatorsAPI {
	return &DelegatorsAPI{client: client}
}

// GetDelegators retrieves all delegators for a network
func (c *DelegatorsAPI) GetDelegators(network string) ([]types.Delegator, error) {
	endpoint := fmt.Sprintf("/networks/%s/delegators", network)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching delegators for network %s: %v", network, err)
		return nil, err
	}
	var result struct {
		Delegations []types.Delegator `json:"delegations"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding delegators: %v", err)
		return nil, err
	}
	return result.Delegations, nil
} 