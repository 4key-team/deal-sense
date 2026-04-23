package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

func TestHandleListProviders(t *testing.T) {
	testProviders := []handler.ProviderInfo{
		{ID: "openai", Name: "OpenAI", Models: []string{"gpt-4o"}},
		{ID: "anthropic", Name: "Anthropic", Models: []string{"claude-sonnet-4-5"}},
		{ID: "gemini", Name: "Gemini", Models: []string{"gemini-2.5-pro"}},
		{ID: "groq", Name: "Groq", Models: []string{"llama-3.3-70b"}},
		{ID: "ollama", Name: "Ollama", Models: []string{"llama3.1:70b"}},
	}
	h := handler.NewHandler(&stubLLM{name: "test"}, nil, nil, nil, stubPrompt, stubPrompt, testProviders, testLogger)
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
