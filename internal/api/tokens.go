package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// TokensAPI manages token-related API operations
type TokensAPI struct {
	client *Client
}

// NewTokensAPI creates a new tokens API client
func NewTokensAPI(client *Client) *TokensAPI {
	return &TokensAPI{client: client}
}

// GetTokenDetails retrieves details for a specific token address
func (c *TokensAPI) GetTokenDetails(tokenAddress string) (*types.TokenInfo, error) {
	endpoint := fmt.Sprintf("/assets/%s", tokenAddress)
	body, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching token details for %s: %v", tokenAddress, err)
		return nil, err
	}
	var tokenDetails types.TokenInfo
	if err := json.Unmarshal(body, &tokenDetails); err != nil {
		log.Printf("Error decoding token %s: %v", tokenAddress, err)
		return nil, err
	}
	return &tokenDetails, nil
} 