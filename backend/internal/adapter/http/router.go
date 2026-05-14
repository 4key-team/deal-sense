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
	mux.HandleFunc("POST /api/proposal/generate-stream", h.HandleGenerateProposalStream)

	return mux
}

// RegisterAdminRoutes mounts admin endpoints on an existing mux. Kept
// separate from NewRouter so callers can opt out of admin features (e.g.
// minimal test fixtures) and so the dependency on usecase/botconfig stays
// out of NewRouter's signature.
func RegisterAdminRoutes(mux *http.ServeMux, botCfg http.Handler) {
	if botCfg != nil {
		mux.Handle("/api/admin/bot-config", botCfg)
	}
}
