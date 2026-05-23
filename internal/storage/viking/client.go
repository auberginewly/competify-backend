// Package viking wraps the OpenViking REST SDK.
// All Agent memory, raw collected data, and provenance chains are stored under
// the viking:// namespace. See docs/storage.md for the directory schema.
package viking

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the OpenViking REST client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates an OpenViking client. baseURL e.g. "http://localhost:8000".
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// AddResourceRequest injects a resource into OpenViking.
type AddResourceRequest struct {
	Path          string `json:"path"`           // local temp file ID or remote URL
	To            string `json:"to,omitempty"`   // target viking://resources/ path
	WatchInterval int    `json:"watch_interval"` // polling interval in minutes
	Wait          bool   `json:"wait"`           // wait for semantic processing
}

// AddResourceResponse returns the assigned URI and task ID.
type AddResourceResponse struct {
	RootURI string `json:"root_uri"` // e.g. viking://resources/ai-tools/qdrant
	TaskID  string `json:"task_id"`
}

// AddResource uploads a resource to OpenViking.
func (c *Client) AddResource(ctx context.Context, req *AddResourceRequest) (*AddResourceResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("viking.AddResource: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/resources", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("viking.AddResource: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("viking.AddResource: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("viking.AddResource: status %d: %s", resp.StatusCode, body)
	}

	var apiResp AddResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("viking.AddResource: decode: %w", err)
	}
	return &apiResp, nil
}

// MonitorCompetitorSite adds a competitor website for periodic monitoring.
func (c *Client) MonitorCompetitorSite(ctx context.Context, targetURL, targetVFSPath string, watchInterval int) (*AddResourceResponse, error) {
	return c.AddResource(ctx, &AddResourceRequest{
		Path:          targetURL,
		To:            targetVFSPath,
		WatchInterval: watchInterval,
		Wait:          true,
	})
}

// FindRequest is a semantic search request.
type FindRequest struct {
	Query string `json:"query"` // natural language query
	Path  string `json:"path"`  // search path e.g. viking://resources/ai-tools/
	Level string `json:"level"` // L0 / L1 / L2 / all
	TopK  int    `json:"top_k"` // max results
}

// FindResult is a single semantic search hit.
type FindResult struct {
	URI        string  `json:"uri"`
	Content    string  `json:"content"`
	Level      string  `json:"level"`       // L0 / L1 / L2
	Score      float64 `json:"score"`       // relevance
	SourceType string  `json:"source_type"` // web / api / social / ...
}

// Find performs deep semantic retrieval across the viking:// namespace.
func (c *Client) Find(ctx context.Context, req *FindRequest) ([]FindResult, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("viking.Find: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/find", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("viking.Find: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("viking.Find: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("viking.Find: status %d: %s", resp.StatusCode, body)
	}

	var results []FindResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("viking.Find: decode: %w", err)
	}
	return results, nil
}

// Grep performs text search (like Unix grep) within a viking:// path.
func (c *Client) Grep(ctx context.Context, pattern, path string) ([]FindResult, error) {
	payload, _ := json.Marshal(map[string]string{
		"pattern": pattern,
		"path":    path,
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/v1/grep", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("viking.Grep: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("viking.Grep: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("viking.Grep: status %d: %s", resp.StatusCode, body)
	}

	var results []FindResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("viking.Grep: decode: %w", err)
	}
	return results, nil
}

// Glob lists resources matching a wildcard pattern under a viking:// path.
func (c *Client) Glob(ctx context.Context, pattern, path string) ([]string, error) {
	q := fmt.Sprintf("path=%s&pattern=%s", path, pattern)
	url := c.baseURL + "/api/v1/glob?" + q

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("viking.Glob: new request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("viking.Glob: do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("viking.Glob: status %d: %s", resp.StatusCode, body)
	}

	var results []string
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("viking.Glob: decode: %w", err)
	}
	return results, nil
}
