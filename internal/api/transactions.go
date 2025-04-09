package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Panorama-Block/avax/internal/types"
)

// TransactionsAPI manages transaction-related API operations
type TransactionsAPI struct {
	client *Client
}

// NewTransactionsAPI creates a new transactions API client
func NewTransactionsAPI(client *Client) *TransactionsAPI {
	return &TransactionsAPI{client: client}
}

// GetTransaction retrieves a specific transaction by hash
func (c *TransactionsAPI) GetTransaction(chainID, txHash string) (*types.TransactionDetail, error) {
	endpoint := fmt.Sprintf("/v1/chains/%s/transactions/%s", chainID, txHash)
	respBody, err := c.client.SendRequest("GET", endpoint, nil, nil)
	if err != nil {
		log.Printf("Error fetching transaction %s on chain %s: %v", txHash, chainID, err)
		return nil, err
	}
	
	var response types.TransactionDetail
	if err := json.Unmarshal(respBody, &response); err != nil {
		log.Printf("Error decoding transaction response: %v", err)
		return nil, fmt.Errorf("error decoding response: %w", err)
	}
	
	return &response, nil
} 