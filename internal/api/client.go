package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client represents an AvaCloud API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new AvaCloud API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// SendRequest sends a request to the AvaCloud API
func (c *Client) SendRequest(method, path string, queryParams url.Values, body io.Reader) ([]byte, error) {
	// Construct full URL with query parameters
	fullURL, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	
	if queryParams != nil {
		fullURL.RawQuery = queryParams.Encode()
	}
	
	// Create request
	req, err := http.NewRequest(method, fullURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	
	// Add headers
	req.Header.Add("Authorization", "Bearer "+c.APIKey)
	req.Header.Add("Content-Type", "application/json")
	
	// Send request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	
	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

// Pagination helper struct used in responses
type PaginationResponse struct {
	NextPageToken string `json:"nextPageToken,omitempty"`
}

// TimeRange represents a time range for filtering API requests
type TimeRange struct {
	StartTime int64
	EndTime   int64
}

// ToQueryParams converts a TimeRange to URL query parameters
func (t *TimeRange) ToQueryParams() url.Values {
	params := url.Values{}
	if t.StartTime > 0 {
		params.Add("startTime", strconv.FormatInt(t.StartTime, 10))
	}
	if t.EndTime > 0 {
		params.Add("endTime", strconv.FormatInt(t.EndTime, 10))
	}
	return params
}

// BlockRange represents a block range for filtering API requests
type BlockRange struct {
	StartBlock int64
	EndBlock   int64
}

// ToQueryParams converts a BlockRange to URL query parameters
func (b *BlockRange) ToQueryParams() url.Values {
	params := url.Values{}
	if b.StartBlock > 0 {
		params.Add("startBlock", strconv.FormatInt(b.StartBlock, 10))
	}
	if b.EndBlock > 0 {
		params.Add("endBlock", strconv.FormatInt(b.EndBlock, 10))
	}
	return params
}

// PaginationParams represents pagination parameters for API requests
type PaginationParams struct {
	PageSize    int
	PageToken   string
	SortOrder   string // "asc" or "desc"
}

// ToQueryParams converts PaginationParams to URL query parameters
func (p *PaginationParams) ToQueryParams() url.Values {
	params := url.Values{}
	if p.PageSize > 0 {
		params.Add("pageSize", strconv.Itoa(p.PageSize))
	}
	if p.PageToken != "" {
		params.Add("pageToken", p.PageToken)
	}
	if p.SortOrder != "" {
		params.Add("sortOrder", p.SortOrder)
	}
	return params
}

// MergeQueryParams merges multiple url.Values objects
func MergeQueryParams(paramsList ...url.Values) url.Values {
	result := url.Values{}
	for _, params := range paramsList {
		for key, values := range params {
			for _, value := range values {
				result.Add(key, value)
			}
		}
	}
	return result
}