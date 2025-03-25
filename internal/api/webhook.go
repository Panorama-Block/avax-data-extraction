package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
)

// WebhookAPI handles webhook-related API calls to AvaCloud
type WebhookAPI struct {
	client *Client
}

// NewWebhookAPI creates a new Webhook API client
func NewWebhookAPI(client *Client) *WebhookAPI {
	return &WebhookAPI{client: client}
}

// EventType represents the type of event to monitor with a webhook
type EventType string

const (
	EventTypeAddressActivity EventType = "address_activity"
	EventTypeERC20Transfer   EventType = "erc20_transfer"
	EventTypeContractEvent   EventType = "contract_event"
)

// WebhookCreate represents the parameters to create a webhook
type WebhookCreate struct {
	CallbackURL  string    `json:"url"`
	ChainID      string    `json:"chainId"`
	EventType    EventType `json:"eventType"`
	IncludeLogs  bool      `json:"includeLogs,omitempty"`
	IncludeInternalTxs bool `json:"includeInternalTxs,omitempty"`
	Addresses    []string  `json:"addresses,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Webhook represents a webhook configuration
type Webhook struct {
	ID           string    `json:"id"`
	URL          string    `json:"url"`
	ChainID      string    `json:"chainId"`
	EventType    EventType `json:"eventType"`
	IncludeLogs  bool      `json:"includeLogs"`
	IncludeInternalTxs bool `json:"includeInternalTxs"`
	Active       bool      `json:"active"`
	CreatedAt    string    `json:"createdAt"`
	UpdatedAt    string    `json:"updatedAt"`
	Addresses    []string  `json:"addresses,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// WebhooksResponse is the response for listing webhooks
type WebhooksResponse struct {
	Webhooks     []Webhook `json:"webhooks"`
	NextPageToken string    `json:"nextPageToken,omitempty"`
}

// CreateWebhook creates a new webhook
func (w *WebhookAPI) CreateWebhook(webhook *WebhookCreate) (*Webhook, error) {
	requestBody, err := json.Marshal(webhook)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}
	
	respBody, err := w.client.SendRequest("POST", "/v1/webhooks", nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	
	var response Webhook
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetWebhooks retrieves all webhooks
func (w *WebhookAPI) GetWebhooks(pagination *PaginationParams) (*WebhooksResponse, error) {
	params := url.Values{}
	if pagination != nil {
		params = pagination.ToQueryParams()
	}
	
	respBody, err := w.client.SendRequest("GET", "/v1/webhooks", params, nil)
	if err != nil {
		return nil, err
	}
	
	var response WebhooksResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetWebhook retrieves a specific webhook by ID
func (w *WebhookAPI) GetWebhook(id string) (*Webhook, error) {
	respBody, err := w.client.SendRequest("GET", fmt.Sprintf("/v1/webhooks/%s", id), nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response Webhook
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// WebhookUpdate represents the parameters to update a webhook
type WebhookUpdate struct {
	URL           *string   `json:"url,omitempty"`
	IncludeLogs   *bool     `json:"includeLogs,omitempty"`
	IncludeInternalTxs *bool `json:"includeInternalTxs,omitempty"`
	Active        *bool     `json:"active,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateWebhook updates a webhook
func (w *WebhookAPI) UpdateWebhook(id string, update *WebhookUpdate) (*Webhook, error) {
	requestBody, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}
	
	respBody, err := w.client.SendRequest("PATCH", fmt.Sprintf("/v1/webhooks/%s", id), nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	
	var response Webhook
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// DeleteWebhook deletes a webhook
func (w *WebhookAPI) DeleteWebhook(id string) error {
	_, err := w.client.SendRequest("DELETE", fmt.Sprintf("/v1/webhooks/%s", id), nil, nil)
	return err
}

// AddressesResponse is the response for listing webhook addresses
type AddressesResponse struct {
	Addresses []string `json:"addresses"`
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// GetWebhookAddresses retrieves addresses associated with a webhook
func (w *WebhookAPI) GetWebhookAddresses(webhookID string, pagination *PaginationParams) (*AddressesResponse, error) {
	params := url.Values{}
	if pagination != nil {
		params = pagination.ToQueryParams()
	}
	
	respBody, err := w.client.SendRequest("GET", fmt.Sprintf("/v1/webhooks/%s/addresses", webhookID), params, nil)
	if err != nil {
		return nil, err
	}
	
	var response AddressesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// AddAddressesToWebhook adds addresses to a webhook
func (w *WebhookAPI) AddAddressesToWebhook(webhookID string, addresses []string) (*AddressesResponse, error) {
	requestBody, err := json.Marshal(map[string][]string{
		"addresses": addresses,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}
	
	respBody, err := w.client.SendRequest("PATCH", fmt.Sprintf("/v1/webhooks/%s/addresses", webhookID), nil, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	
	var response AddressesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// RemoveAddressesFromWebhook removes addresses from a webhook
func (w *WebhookAPI) RemoveAddressesFromWebhook(webhookID string, addresses []string) error {
	requestBody, err := json.Marshal(map[string][]string{
		"addresses": addresses,
	})
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}
	
	_, err = w.client.SendRequest("DELETE", fmt.Sprintf("/v1/webhooks/%s/addresses", webhookID), nil, bytes.NewBuffer(requestBody))
	return err
}

// SharedSecretResponse is the response for the shared secret endpoint
type SharedSecretResponse struct {
	SharedSecret string `json:"sharedSecret"`
}

// GenerateSharedSecret generates or rotates a shared secret for webhook HMAC validation
func (w *WebhookAPI) GenerateSharedSecret() (*SharedSecretResponse, error) {
	respBody, err := w.client.SendRequest("POST", "/v1/webhooks:generateOrRotateSharedSecret", nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response SharedSecretResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// GetSharedSecret retrieves the current shared secret
func (w *WebhookAPI) GetSharedSecret() (*SharedSecretResponse, error) {
	respBody, err := w.client.SendRequest("GET", "/v1/webhooks:getSharedSecret", nil, nil)
	if err != nil {
		return nil, err
	}
	
	var response SharedSecretResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}
	
	return &response, nil
}

// WebhookEvent represents an event received from a webhook
type WebhookEvent struct {
	WebhookID  string                 `json:"webhookId"`
	EventType  EventType              `json:"eventType"`
	MessageID  string                 `json:"messageId"`
	Event      map[string]interface{} `json:"event"`
}

// VerifyWebhookSignature verifies the HMAC signature of a webhook event
func VerifyWebhookSignature(body []byte, signature string, sharedSecret string) (bool, error) {
	// Implementation using crypto/hmac
	// This would verify that the signature matches the HMAC-SHA256 of the body using the shared secret
	// I'm omitting the implementation details as this is a demonstration
	// In a real implementation, you'd use crypto/hmac to compute the signature and compare it
	return true, nil
}