package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

func TestHandleListProviders(t *testing.T) {
	h := handler.NewHandler(&stubLLM{name: "test"}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/llm/providers", nil)
	rec := httptest.NewRecorder()

	h.HandleListProviders(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	providers, ok := resp["providers"].([]any)
	if !ok || len(providers) == 0 {
		t.Fatal("expected non-empty providers list")
	}

	// Should have at least anthropic, openai, gemini, groq, ollama
	if len(providers) < 5 {
		t.Errorf("got %d providers, want >= 5", len(providers))
	}
}
