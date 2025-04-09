package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// ValidatorsAPI manages validator-related API operations
type ValidatorsAPI struct {
	client *Client
}

// NewValidatorsAPI creates a new validators API client
func NewValidatorsAPI(client *Client) *ValidatorsAPI {
	return &ValidatorsAPI{client: client}
}

// GetValidators retrieves all validators for a network
func (c *ValidatorsAPI) GetValidators(network string) ([]types.Validator, error) {
	endpoint := fmt.Sprintf("/networks/%s/validators", network)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching validators for network %s: %v", network, err)
		return nil, err
	}
	var result struct {
		Validators []types.Validator `json:"validators"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding validators: %v", err)
		return nil, err
	}
	return result.Validators, nil
}

// GetValidatorByNodeID retrieves a specific validator by node ID
func (c *ValidatorsAPI) GetValidatorByNodeID(network, nodeID string) (*types.Validator, error) {
	endpoint := fmt.Sprintf("/networks/%s/validators/%s", network, nodeID)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching validator %s: %v", nodeID, err)
		return nil, err
	}
	var wrapper struct {
		Validator types.Validator `json:"validator"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		log.Printf("Error decoding validator %s: %v", nodeID, err)
		return nil, err
	}
	return &wrapper.Validator, nil
} 