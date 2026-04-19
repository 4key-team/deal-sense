package http

import (
	"encoding/json"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// Handler holds use cases and exposes HTTP endpoints.
type Handler struct {
	llm      usecase.LLMProvider
	parser   usecase.DocumentParser
	template usecase.TemplateEngine
}

func NewHandler(llm usecase.LLMProvider, parser usecase.DocumentParser, template usecase.TemplateEngine) *Handler {
	return &Handler{llm: llm, parser: parser, template: template}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
