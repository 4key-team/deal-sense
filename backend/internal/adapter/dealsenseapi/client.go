// Package dealsenseapi implements the bot's APIClient port via HTTP calls
// to the Deal Sense backend.
package dealsenseapi

import (
	"context"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// HTTPClient calls the Deal Sense HTTP API with optional X-API-Key auth.
type HTTPClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewHTTPClient constructs an HTTPClient. Pass an empty apiKey when the
// backend is running in open-access mode.
func NewHTTPClient(baseURL, apiKey string, c *http.Client) *HTTPClient {
	if c == nil {
		c = http.DefaultClient
	}
	return &HTTPClient{baseURL: baseURL, apiKey: apiKey, http: c}
}

// AnalyzeTender is a stub for the RED step — returns nil/nil so the
// accompanying test fails at runtime instead of compile-time.
func (c *HTTPClient) AnalyzeTender(ctx context.Context, req telegram.AnalyzeTenderRequest) (*telegram.AnalyzeTenderResponse, error) {
	return nil, nil
}
