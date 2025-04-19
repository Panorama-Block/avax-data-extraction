package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

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
	log.Printf("Fetching delegators with extended timeout for network %s", network)
	// Use 30 second timeout for this request as it can return a large dataset
	body, err := c.client.makeRequestWithTimeout(endpoint, 30*time.Second)
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