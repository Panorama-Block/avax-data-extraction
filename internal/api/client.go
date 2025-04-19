package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	rate       float64     // tokens per second
	burst      int         // maximum bucket size
	mu         sync.Mutex  // mutex for thread safety
	tokens     float64     // current token count
	lastRefill time.Time   // last time tokens were added
}

// NewRateLimiter creates a new rate limiter with the given rate and burst limit
func NewRateLimiter(rate float64, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

// Wait blocks until a token is available
func (r *RateLimiter) Wait() {
	for {
		if r.TryAcquire() {
			return
		}
		// Sleep a little bit before trying again
		time.Sleep(10 * time.Millisecond)
	}
}

// TryAcquire attempts to acquire a token, returns true if successful
func (r *RateLimiter) TryAcquire() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	
	// Add tokens based on elapsed time
	r.tokens = r.tokens + elapsed*r.rate
	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}
	
	r.lastRefill = now
	
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	
	return false
}

type Client struct {
    BaseURL     string
    APIKey      string
    HTTPClient  *http.Client
    RateLimiter *RateLimiter
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        BaseURL: baseURL,
        APIKey:  apiKey,
        HTTPClient: &http.Client{
            Timeout: 10 * time.Second,
        },
        // Default rate limiter: 5 requests per second with burst of 10
        RateLimiter: NewRateLimiter(5, 10),
    }
}

// SetRateLimits allows configuring the rate limits
func (c *Client) SetRateLimits(ratePerSecond float64, burst int) {
    c.RateLimiter = NewRateLimiter(ratePerSecond, burst)
}

// makeRequest makes a request to the API with the specified endpoint
func (c *Client) makeRequest(endpoint string) ([]byte, error) {
	return c.makeRequestWithTimeout(endpoint, 10*time.Second)
}

// makeRequestWithTimeout makes a request to the API with a custom timeout
func (c *Client) makeRequestWithTimeout(endpoint string, timeout time.Duration) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, endpoint)
	log.Printf("[DEBUG] makeRequest => %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set API key if provided
	if c.APIKey != "" {
		req.Header.Set("x-glacier-api-key", c.APIKey)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req = req.WithContext(ctx)

	// Apply rate limiting if configured
	if c.RateLimiter != nil {
		c.RateLimiter.Wait()
	}

	// Send request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%s (status: %d)", string(body), resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (c *Client) SendRequest(method, path string, queryParams url.Values, body io.Reader) ([]byte, error) {
    fullURL, err := url.Parse(c.BaseURL + path)
    if err != nil {
        return nil, fmt.Errorf("URL inválida: %w", err)
    }
    if queryParams != nil {
        fullURL.RawQuery = queryParams.Encode()
    }

    // Wait for rate limiter to allow the request
    c.RateLimiter.Wait()

    req, err := http.NewRequest(method, fullURL.String(), body)
    if err != nil {
        return nil, fmt.Errorf("erro ao criar requisição: %w", err)
    }

    req.Header.Add("Authorization", "Bearer "+c.APIKey)
    req.Header.Add("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("erro ao enviar requisição: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusTooManyRequests {
        // Implement retry with backoff for 429 errors
        retryAfter := 1 * time.Second
        if retryHeaderValue := resp.Header.Get("Retry-After"); retryHeaderValue != "" {
            if seconds, err := strconv.Atoi(retryHeaderValue); err == nil {
                retryAfter = time.Duration(seconds) * time.Second
            }
        }
        
        log.Printf("[SendRequest] Rate limited. Retrying after %v", retryAfter)
        time.Sleep(retryAfter)
        return c.SendRequest(method, path, queryParams, body) // Recursive retry
    }

    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("erro ao ler corpo da resposta: %w", err)
    }
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("requisição API falhou com status %d: %s", resp.StatusCode, string(respBody))
    }
    
    return respBody, nil
}

type PaginationResponse struct {
    NextPageToken string `json:"nextPageToken,omitempty"`
}

type TimeRange struct {
    StartTime int64
    EndTime   int64
}

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

type BlockRange struct {
    StartBlock int64
    EndBlock   int64
}

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

type PaginationParams struct {
    PageSize    int
    PageToken   string
    SortOrder   string // "asc" or "desc"
}

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
