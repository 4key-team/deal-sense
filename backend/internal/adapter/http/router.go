package http

import "net/http"

// BypassedPaths returns the set of routes that must skip APIKeyAuth +
// RateLimit when wired in cmd/server. Single source of truth — keep this
// list in sync with the unconditional routes registered in NewRouter.
func BypassedPaths() map[string]struct{} {
	return map[string]struct{}{
		"/healthz": {},
		"/readyz":  {},
		"/metrics": {},
	}
}

func NewRouter(h *Handler, m MetricsRenderer) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.HandleLiveness)
	mux.HandleFunc("GET /readyz", h.HandleReadiness)
	if m != nil {
		mux.Handle("GET /metrics", MetricsHandler(m))
	}

	mux.HandleFunc("POST /api/llm/check", h.HandleCheckConnection)
	mux.HandleFunc("GET /api/llm/providers", h.HandleListProviders)
	mux.HandleFunc("GET /api/llm/models", h.HandleListModels)
	mux.HandleFunc("POST /api/tender/analyze", h.HandleAnalyzeTender)
	mux.HandleFunc("POST /api/proposal/generate", h.HandleGenerateProposal)

	return mux
}
