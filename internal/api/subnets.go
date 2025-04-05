package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// SubnetsAPI manages subnet-related API operations
type SubnetsAPI struct {
	client *Client
}

// NewSubnetsAPI creates a new subnets API client
func NewSubnetsAPI(client *Client) *SubnetsAPI {
	return &SubnetsAPI{client: client}
}

// GetSubnets retrieves all subnets for a network
func (c *SubnetsAPI) GetSubnets(network string) ([]types.Subnet, error) {
	endpoint := fmt.Sprintf("/networks/%s/subnets", network)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching subnets for network %s: %v", network, err)
		return nil, err
	}
	var result struct {
		Subnets []types.Subnet `json:"subnets"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding subnets: %v", err)
		return nil, err
	}
	return result.Subnets, nil
}

// GetSubnetByID retrieves a specific subnet by ID
func (c *SubnetsAPI) GetSubnetByID(network, subnetID string) (*types.Subnet, error) {
	endpoint := fmt.Sprintf("/networks/%s/subnets/%s", network, subnetID)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching subnet %s: %v", subnetID, err)
		return nil, err
	}
	var wrapper struct {
		Subnet types.Subnet `json:"subnet"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		log.Printf("Error decoding subnet %s: %v", subnetID, err)
		return nil, err
	}
	return &wrapper.Subnet, nil
} 