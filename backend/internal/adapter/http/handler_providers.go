package http

import "net/http"

type providerInfo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Models []string `json:"models"`
}

var availableProviders = []providerInfo{
	{ID: "anthropic", Name: "Anthropic", Models: []string{"claude-haiku-4-5", "claude-sonnet-4-5", "claude-opus-4-1"}},
	{ID: "openai", Name: "OpenAI", Models: []string{"gpt-4o", "gpt-4o-mini", "o3-mini"}},
	{ID: "gemini", Name: "Google Gemini", Models: []string{"gemini-2.5-pro", "gemini-2.5-flash"}},
	{ID: "groq", Name: "Groq", Models: []string{"llama-3.3-70b", "mixtral-8x7b"}},
	{ID: "ollama", Name: "Ollama (local)", Models: []string{"llama3.1:70b", "qwen2.5:32b"}},
	{ID: "custom", Name: "Custom", Models: []string{}},
}

func (h *Handler) HandleListProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": availableProviders,
	})
}
