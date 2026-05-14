package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
)

// TestBuildHandler_BypassChain composes the production middleware stack
// (Recover→Logger→CORS→MetricsRequests→combined→APIKeyAuth+RateLimit→mux)
// and pins that /metrics, /healthz, /readyz skip authentication while
// /api/llm/providers requires X-API-Key.
//
// Regression target: a refactor that puts APIKeyAuth ahead of the bypass
// branch will flip these expectations and the test fails loudly.
func TestBuildHandler_BypassChain(t *testing.T) {
	// Stub mux that answers 200 on every registered route. The chain itself
	// is under test, not the handlers behind it.
	mux := http.NewServeMux()
	for _, p := range []string{"/healthz", "/readyz", "/metrics", "/api/llm/providers"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}

	h := buildHandler(
		mux,
		"secret-key", // APIKeyAuth requires this header
		0.0, 0,       // rate limit disabled — out of scope for this test
		metrics.NewCollector(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	tests := []struct {
		name     string
		path     string
		apiKey   string
		wantCode int
	}{
		{"healthz bypasses auth", "/healthz", "", http.StatusOK},
		{"readyz bypasses auth", "/readyz", "", http.StatusOK},
		{"metrics bypasses auth", "/metrics", "", http.StatusOK},
		{"gated route without key → 401", "/api/llm/providers", "", http.StatusUnauthorized},
		{"gated route with valid key → 200", "/api/llm/providers", "secret-key", http.StatusOK},
		{"gated route with wrong key → 401", "/api/llm/providers", "wrong", http.StatusUnauthorized},
		{"bypass route with junk key → still 200", "/healthz", "wrong", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantCode {
				t.Errorf("path=%q key=%q → %d, want %d", tt.path, tt.apiKey, rec.Code, tt.wantCode)
			}
		})
	}
}

func TestBuildHandler_BypassedPathsCount(t *testing.T) {
	// Defensive: if someone removes a route from BypassedPaths but forgets
	// the chain wiring, this counts the 200-without-key set and fails on
	// drift. Locks the bypass surface explicitly at 3.
	mux := http.NewServeMux()
	candidates := []string{"/healthz", "/readyz", "/metrics"}
	for _, p := range candidates {
		mux.HandleFunc(p, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	h := buildHandler(mux, "secret", 0.0, 0, metrics.NewCollector(),
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	bypassed := 0
	for _, p := range candidates {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			bypassed++
		}
	}
	if bypassed != 3 {
		t.Errorf("bypassed paths reachable without key = %d, want 3 (regression: chain reshuffled)", bypassed)
	}
}
