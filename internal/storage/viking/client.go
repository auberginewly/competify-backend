// Package viking wraps the OpenViking REST SDK.
// All Agent memory, raw collected data, and provenance chains are stored under
// the viking:// namespace. See docs/storage.md for the directory schema.
package viking

import (
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

// TODO Phase 6:
//   - AddResource(ctx, *AddResourceRequest) (*AddResourceResponse, error)
//   - MonitorCompetitorSite(ctx, url, vfsPath string, interval int)
//   - L0/L1/L2 layered fetching
