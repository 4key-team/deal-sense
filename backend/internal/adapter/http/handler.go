package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// PromptResolver returns a system prompt for a given language name (e.g. "Russian").
type PromptResolver func(langName string) string

// Handler holds use cases and exposes HTTP endpoints.
// ProviderInfo describes an LLM provider for the frontend.
type ProviderInfo struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Models []string `json:"models"`
}

// Handler holds use cases and exposes HTTP endpoints.
type Handler struct {
	llm            usecase.LLMProvider
	llmFactory     usecase.LLMProviderFactory
	parser         usecase.DocumentParser
	template       usecase.TemplateEngine
	tenderPrompt   PromptResolver
	proposalPrompt PromptResolver
	providers      []ProviderInfo
	logger         *slog.Logger
}

func NewHandler(
	llm usecase.LLMProvider,
	factory usecase.LLMProviderFactory,
	parser usecase.DocumentParser,
	template usecase.TemplateEngine,
	tenderPrompt PromptResolver,
	proposalPrompt PromptResolver,
	providers []ProviderInfo,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		llm: llm, llmFactory: factory,
		parser: parser, template: template,
		tenderPrompt: tenderPrompt, proposalPrompt: proposalPrompt,
		providers: providers,
		logger:    logger.With("component", "handler"),
	}
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

func resolveLang(r *http.Request) string {
	if r.FormValue("lang") == "en" {
		return "English"
	}
	return "Russian"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// mustReadMultipartFile reads a multipart file entry that was already buffered by ParseMultipartForm.
// Open and ReadAll on in-memory multipart data cannot fail.
func mustReadMultipartFile(fh *multipart.FileHeader) []byte {
	f, _ := fh.Open() //nolint:errcheck // in-memory open after ParseMultipartForm
	data, _ := io.ReadAll(f)
	f.Close()
	return data
}
