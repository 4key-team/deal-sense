package http

import (
	"encoding/json"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// Handler holds use cases and exposes HTTP endpoints.
type Handler struct {
	llm        usecase.LLMProvider
	llmFactory usecase.LLMProviderFactory
	parser     usecase.DocumentParser
	template   usecase.TemplateEngine
}

func NewHandler(llm usecase.LLMProvider, factory usecase.LLMProviderFactory, parser usecase.DocumentParser, template usecase.TemplateEngine) *Handler {
	return &Handler{llm: llm, llmFactory: factory, parser: parser, template: template}
}

// resolveLLM returns an LLMProvider from request headers, or the default one.
func (h *Handler) resolveLLM(r *http.Request) usecase.LLMProvider {
	provider := r.Header.Get("X-LLM-Provider")
	if provider == "" || h.llmFactory == nil {
		return h.llm
	}
	p, err := h.llmFactory.Create(usecase.LLMProviderConfig{
		Provider: provider,
		BaseURL:  r.Header.Get("X-LLM-URL"),
		APIKey:   r.Header.Get("X-LLM-Key"),
		Model:    r.Header.Get("X-LLM-Model"),
	})
	if err != nil {
		return h.llm
	}
	return p
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
