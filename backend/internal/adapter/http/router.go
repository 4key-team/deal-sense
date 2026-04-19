package http

import "net/http"

func NewRouter(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/llm/check", h.HandleCheckConnection)
	mux.HandleFunc("GET /api/llm/providers", h.HandleListProviders)
	mux.HandleFunc("POST /api/tender/analyze", h.HandleAnalyzeTender)
	mux.HandleFunc("POST /api/proposal/generate", h.HandleGenerateProposal)

	return mux
}
