package avacloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Panorama-Block/avax/internal/config"
)

const (
	DefaultAvaCloudAPIBaseURL = "https://glacier-api.avax.network"
)

// WebhookService handles AvaCloud webhooks
type WebhookService struct {
	client  *http.Client
	apiKey  string
	baseURL string
}

// WebhookCreateRequest represents the request body for creating a webhook
type WebhookCreateRequest struct {
	EventType          string `json:"eventType"`
	ChainID            string `json:"chainId,omitempty"`
	Metadata           Metadata `json:"metadata"`
	IncludeInternalTxs bool   `json:"includeInternalTxs"`
	IncludeLogs        bool   `json:"includeLogs"`
	URL                string `json:"url"`
	Secret             string `json:"secret,omitempty"`
	Status             string `json:"status"`
	Name               string `json:"name"`
	Description        string `json:"description"`
}

// Metadata represents additional filters for the webhook
type Metadata struct {
	Addresses       []string `json:"addresses,omitempty"`
	EventSignatures []string `json:"eventSignatures,omitempty"`
}

// WebhookResponse represents the response from webhook operations
type WebhookResponse struct {
	ID                string    `json:"id"`
	WebhookID         string    `json:"webhookId"`
	EventType         string    `json:"eventType"`
	ChainID           string    `json:"chainId,omitempty"`
	Metadata          Metadata  `json:"metadata,omitempty"`
	IncludeInternalTxs bool      `json:"includeInternalTxs"`
	IncludeLogs       bool      `json:"includeLogs"`
	URL               string    `json:"url"`
	Description       string    `json:"description"`
	Name              string    `json:"name"`
	Status            string    `json:"status"`
	CreatedAt         int64     `json:"createdAt"`
	UpdatedAt         int64     `json:"updatedAt"`
}

// WebhookListResponse represents the response from listing webhooks
type WebhookListResponse struct {
	Webhooks []WebhookResponse `json:"webhooks"`
}

// ManageTransactionWebhook provides utility functions for managing transaction webhooks
type TransactionWebhookManager struct {
	service *WebhookService
	cfg     *config.Config
}

// NewWebhookService creates a new AvaCloud webhook service
func NewWebhookService(apiKey string) *WebhookService {
	return &WebhookService{
		client:  &http.Client{Timeout: 10 * time.Second},
		apiKey:  apiKey,
		baseURL: DefaultAvaCloudAPIBaseURL,
	}
}

// ListWebhooks lists all webhooks
func (s *WebhookService) ListWebhooks() ([]WebhookResponse, error) {
	url := fmt.Sprintf("%s/v1/webhooks", s.baseURL)
	log.Printf("Listing webhooks from URL: %s", url)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-glacier-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Error making list webhooks request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("List webhooks response status: %d, body: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list webhooks: %s (status: %d)", string(respBody), resp.StatusCode)
	}

	// Re-read the body since we already consumed it
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	var listResp WebhookListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		log.Printf("Error decoding list webhooks response: %v", err)
		return nil, err
	}

	return listResp.Webhooks, nil
}

// GetWebhook gets a webhook by ID
func (s *WebhookService) GetWebhook(webhookID string) (*WebhookResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/webhooks/%s", s.baseURL, webhookID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-glacier-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get webhook: %s (status: %d)", string(body), resp.StatusCode)
	}

	var webhook WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// CreateWebhook creates a new webhook
func (s *WebhookService) CreateWebhook(request WebhookCreateRequest) (*WebhookResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		log.Printf("Erro ao serializar requisição de webhook: %v", err)
		return nil, err
	}

	// Debug log the request body
	log.Printf("Criando webhook com request: %s", string(body))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/webhooks", s.baseURL), bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Erro ao criar requisição HTTP: %v", err)
		return nil, err
	}

	req.Header.Set("x-glacier-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("Enviando requisição para: %s", fmt.Sprintf("%s/v1/webhooks", s.baseURL))
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Erro ao enviar requisição: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("Resposta de criação do webhook - status: %d, body: %s", resp.StatusCode, string(respBody))

	// Handle specific status codes
	if resp.StatusCode == http.StatusConflict {
		log.Printf("Webhook com URL %s já existe, tentando obter webhook existente", request.URL)
		
		// Try to get webhooks and find the one with matching URL
		webhooks, err := s.ListWebhooks()
		if err != nil {
			return nil, fmt.Errorf("webhook já existe mas não foi possível listar webhooks: %v", err)
		}
		
		for _, webhook := range webhooks {
			if webhook.URL == request.URL {
				log.Printf("Encontrado webhook existente com URL %s, ID: %s", request.URL, webhook.ID)
				return &webhook, nil
			}
		}
		
		return nil, fmt.Errorf("webhook já existe mas não foi possível encontrá-lo")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("falha ao criar webhook: %s (status: %d)", string(respBody), resp.StatusCode)
	}

	// Re-read the body since we already consumed it
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))

	var webhook WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		log.Printf("Erro ao decodificar resposta: %v", err)
		return nil, err
	}

	return &webhook, nil
}

// DeleteWebhook deletes a webhook by ID
func (s *WebhookService) DeleteWebhook(webhookID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/webhooks/%s", s.baseURL, webhookID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("x-glacier-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete webhook: %s (status: %d)", string(body), resp.StatusCode)
	}

	return nil
}

// UpdateWebhook updates a webhook by ID
func (s *WebhookService) UpdateWebhook(webhookID string, request WebhookCreateRequest) (*WebhookResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/webhooks/%s", s.baseURL, webhookID), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-glacier-api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update webhook: %s (status: %d)", string(body), resp.StatusCode)
	}

	var webhook WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
		return nil, err
	}

	return &webhook, nil
}

// CreateTransactionWebhook creates a webhook to monitor addresses for transaction activity
func CreateTransactionWebhook(cfg *config.Config, addresses []string, eventSignatures []string, name string, description string) (string, error) {
	// Get required config variables
	webhookURL := cfg.WebhookURL
	webhookSecret := cfg.WebhookSecret
	apiKey := cfg.AvaCloudAPIKey
	baseURL := cfg.AvaCloudAPIBaseURL
	chainID := "43114" // Default to Avalanche C-Chain
	
	// Use default values if not provided
	if name == "" {
		name = "transaction-monitoring-webhook"
	}
	
	if description == "" {
		description = "Transaction monitoring webhook"
	}
	
	log.Printf("Configuring transaction webhook with URL=%s, API Key=%s...", webhookURL, apiKey[:min(10, len(apiKey))]+"***")

	if apiKey == "" {
		return "", fmt.Errorf("AVACLOUD_API_KEY not set")
	}
	
	if webhookURL == "" {
		return "", fmt.Errorf("WEBHOOK_URL not set")
	}
	
	if webhookSecret == "" {
		return "", fmt.Errorf("WEBHOOK_SECRET not set")
	}

	if len(addresses) == 0 {
		return "", fmt.Errorf("no addresses to monitor provided")
	}

	// Create webhook service
	service := NewWebhookService(apiKey)
	
	// Use custom base URL if provided
	if baseURL != "" {
		service.baseURL = baseURL
		log.Printf("Using custom API base URL: %s", baseURL)
	} else {
		log.Printf("Using default API base URL: %s", service.baseURL)
	}

	// Create webhook request
	request := WebhookCreateRequest{
		EventType: "address_activity",
		ChainID:   chainID,
		Metadata: Metadata{
			Addresses:       addresses,
			EventSignatures: eventSignatures,
		},
		IncludeInternalTxs: true,
		IncludeLogs:        true,
		URL:                webhookURL,
		Secret:             webhookSecret,
		Status:             "active",
		Description:        description,
		Name:               name,
	}

	response, err := service.CreateWebhook(request)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction webhook: %v", err)
	}

	log.Printf("Created transaction webhook with ID %s", response.ID)
	return response.ID, nil
}

// Helper function for safe string slicing
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// EnsureCChainWebhook ensures that a C-Chain webhook exists, creating it if it doesn't
func EnsureCChainWebhook(cfg *config.Config) (string, error) {
	// Get required config variables
	webhookURL := cfg.WebhookURL
	webhookSecret := cfg.WebhookSecret
	apiKey := cfg.AvaCloudAPIKey
	baseURL := cfg.AvaCloudAPIBaseURL

	log.Printf("Configurando webhook com URL=%s, API Key=%s...", webhookURL, apiKey[:10]+"***")

	if apiKey == "" {
		return "", fmt.Errorf("AVACLOUD_API_KEY not set")
	}
	
	if webhookURL == "" {
		return "", fmt.Errorf("WEBHOOK_URL not set")
	}
	
	if webhookSecret == "" {
		return "", fmt.Errorf("WEBHOOK_SECRET not set")
	}

	// Create webhook service
	service := NewWebhookService(apiKey)
	
	// Use custom base URL if provided
	if baseURL != "" {
		service.baseURL = baseURL
		log.Printf("Using custom API base URL: %s", baseURL)
	} else {
		log.Printf("Using default API base URL: %s", service.baseURL)
	}

	log.Printf("Criando novo webhook para C-Chain...")

	// Create webhook request
	request := WebhookCreateRequest{
		EventType: "address_activity",
		ChainID:   "43114",
		Metadata: Metadata{
			Addresses: []string{"0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"},
		},
		IncludeInternalTxs: true,
		IncludeLogs:        true,
		URL:                webhookURL,
		Secret:             webhookSecret,
		Status:             "active",
		Description:        "c-chain address stream",
		Name:               "c-chain",
	}

	response, err := service.CreateWebhook(request)
	if err != nil {
		return "", fmt.Errorf("failed to create c-chain webhook: %v", err)
	}

	log.Printf("Webhook C-Chain criado com ID %s", response.ID)
	return response.ID, nil
}

// NewTransactionWebhookManager creates a new manager for transaction webhooks
func NewTransactionWebhookManager(cfg *config.Config) (*TransactionWebhookManager, error) {
	apiKey := cfg.AvaCloudAPIKey
	baseURL := cfg.AvaCloudAPIBaseURL

	if apiKey == "" {
		return nil, fmt.Errorf("AVACLOUD_API_KEY not set")
	}

	service := NewWebhookService(apiKey)
	
	// Use custom base URL if provided
	if baseURL != "" {
		service.baseURL = baseURL
	}

	return &TransactionWebhookManager{
		service: service,
		cfg:     cfg,
	}, nil
}

// ListTransactionWebhooks lists all transaction monitoring webhooks
func (m *TransactionWebhookManager) ListTransactionWebhooks() ([]WebhookResponse, error) {
	webhooks, err := m.service.ListWebhooks()
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %v", err)
	}

	// Filter only address_activity webhooks
	var transactionWebhooks []WebhookResponse
	for _, webhook := range webhooks {
		if webhook.EventType == "address_activity" {
			transactionWebhooks = append(transactionWebhooks, webhook)
		}
	}

	return transactionWebhooks, nil
}

// GetTransactionWebhook gets a specific webhook by ID
func (m *TransactionWebhookManager) GetTransactionWebhook(webhookID string) (*WebhookResponse, error) {
	return m.service.GetWebhook(webhookID)
}

// UpdateTransactionWebhook updates a transaction webhook with new addresses or event signatures
func (m *TransactionWebhookManager) UpdateTransactionWebhook(webhookID string, addresses []string, eventSignatures []string) (*WebhookResponse, error) {
	// Get webhook to preserve existing properties
	webhook, err := m.service.GetWebhook(webhookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook %s: %v", webhookID, err)
	}

	// Create update request
	request := WebhookCreateRequest{
		EventType: webhook.EventType,
		ChainID:   webhook.ChainID,
		Metadata: Metadata{
			Addresses:       addresses,
			EventSignatures: eventSignatures,
		},
		IncludeInternalTxs: webhook.IncludeInternalTxs,
		IncludeLogs:        webhook.IncludeLogs,
		URL:                webhook.URL,
		Secret:             m.cfg.WebhookSecret, // Use the current secret
		Status:             webhook.Status,
		Description:        webhook.Description,
		Name:               webhook.Name,
	}

	return m.service.UpdateWebhook(webhookID, request)
}

// DeleteTransactionWebhook deletes a transaction webhook by ID
func (m *TransactionWebhookManager) DeleteTransactionWebhook(webhookID string) error {
	return m.service.DeleteWebhook(webhookID)
} 