package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
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
    fullURL := c.BaseURL + endpoint
    log.Printf("[DEBUG] makeRequest => %s", fullURL)

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

    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        log.Printf("[makeRequest] body error=%s", string(bodyBytes))
        return nil, fmt.Errorf("erro na requisição: %s", resp.Status)
    }
    return io.ReadAll(resp.Body)
}

func (c *Client) SendRequest(method, path string, queryParams url.Values, body io.Reader) ([]byte, error) {
    fullURL, err := url.Parse(c.BaseURL + path)
    if err != nil {
        return nil, fmt.Errorf("URL inválida: %w", err)
    }
    if queryParams != nil {
        fullURL.RawQuery = queryParams.Encode()
    }

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
