package api

import "github.com/Panorama-Block/avax/internal/config"

// API provides a centralized access point to all available APIs
type API struct {
	Client        *Client
	Chains        *ChainsAPI
	Blocks        *BlocksAPI
	Validators    *ValidatorsAPI
	Delegators    *DelegatorsAPI
	Subnets       *SubnetsAPI
	Blockchains   *BlockchainsAPI
	Teleporter    *TeleporterAPI
	Tokens        *TokensAPI
	Transactions  *TransactionsAPI
}

// NewAPI creates a new centralized API client with all submodules
func NewAPI(baseURL, apiKey string) *API {
	client := NewClient(baseURL, apiKey)
	
	return &API{
		Client:        client,
		Chains:        NewChainsAPI(client),
		Blocks:        NewBlocksAPI(client),
		Validators:    NewValidatorsAPI(client),
		Delegators:    NewDelegatorsAPI(client),
		Subnets:       NewSubnetsAPI(client),
		Blockchains:   NewBlockchainsAPI(client),
		Teleporter:    NewTeleporterAPI(client),
		Tokens:        NewTokensAPI(client),
		Transactions:  NewTransactionsAPI(client),
	}
}

// NewAPIWithConfig creates a new API client using configuration values
func NewAPIWithConfig(cfg *config.Config) *API {
	client := NewClient(cfg.APIUrl, cfg.APIKey)
	
	// Set rate limits from config
	client.SetRateLimits(cfg.APIRateLimit, cfg.APIRateBurst)
	
	return &API{
		Client:        client,
		Chains:        NewChainsAPI(client),
		Blocks:        NewBlocksAPI(client),
		Validators:    NewValidatorsAPI(client),
		Delegators:    NewDelegatorsAPI(client),
		Subnets:       NewSubnetsAPI(client),
		Blockchains:   NewBlockchainsAPI(client),
		Teleporter:    NewTeleporterAPI(client),
		Tokens:        NewTokensAPI(client),
		Transactions:  NewTransactionsAPI(client),
	}
} 