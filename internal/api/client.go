package api

import (
    "errors"
    "io"
    "log"
    "net/http"
    "time"
)

type Client struct {
    BaseURL    string
    APIKey     string
    HTTPClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        BaseURL: baseURL,
        APIKey:  apiKey,
        HTTPClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

func (c *Client) makeRequest(endpoint string) ([]byte, error) {
    // Constrói a URL final
    fullURL := c.BaseURL + endpoint
    log.Printf("[DEBUG] makeRequest => fullURL=%s", fullURL)

    req, err := http.NewRequest("GET", fullURL, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("x-glacier-api-key", c.APIKey)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        log.Printf("[makeRequest] erro HTTPClient.Do: %v", err)
        return nil, err
    }
    defer resp.Body.Close()

    log.Printf("[DEBUG] makeRequest => status=%d %s", resp.StatusCode, resp.Status)
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        log.Printf("[makeRequest] body error=%s", string(bodyBytes))
        return nil, errors.New("erro na requisição: " + resp.Status)
    }

    return io.ReadAll(resp.Body)
}
