package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/Panorama-Block/avax/internal/types"
)

// BlocksAPI manages block-related API operations
type BlocksAPI struct {
	client *Client
}

// NewBlocksAPI creates a new blocks API client
func NewBlocksAPI(client *Client) *BlocksAPI {
	return &BlocksAPI{client: client}
}

// GetBlocks retrieves blocks for a specific chain with pagination support
func (c *BlocksAPI) GetBlocks(chainID string, pageSize int, pageToken string) ([]types.Block, string, error) {
	params := url.Values{}
	params.Set("chainId", chainID)
	if pageSize > 0 {
		params.Set("pageSize", strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	respBody, err := c.client.makeRequest("/blocks?" + params.Encode())
	if err != nil {
		log.Printf("Error listing blocks for chain %s: %v", chainID, err)
		return nil, "", err
	}
	var result struct {
		Blocks        []types.Block `json:"blocks"`
		NextPageToken string        `json:"nextPageToken"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("Error decoding blocks for chain %s: %v", chainID, err)
		return nil, "", err
	}
	return result.Blocks, result.NextPageToken, nil
}

// GetBlockByNumberOrHash retrieves a specific block by its number or hash
func (c *BlocksAPI) GetBlockByNumberOrHash(chainID, blockIdentifier string) (*types.Block, error) {
	endpoint := fmt.Sprintf("/chains/%s/blocks/%s", chainID, blockIdentifier)
	respBody, err := c.client.makeRequest(endpoint)
	if err != nil {
		log.Printf("Error fetching block %s (chain %s): %v", blockIdentifier, chainID, err)
		return nil, err
	}
	var block types.Block
	if err := json.Unmarshal(respBody, &block); err != nil {
		return nil, err
	}
	return &block, nil
} 