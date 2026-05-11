package http

import "net/http"

// HandleLiveness reports whether the process is alive. It performs no
// dependency checks — failure here means the process should be killed
// and rescheduled. Used as the Kubernetes/Docker livenessProbe.
func (h *Handler) HandleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleReadiness reports whether the process is ready to serve traffic.
// Returns 200 only when the LLM provider has been wired (sanity check
// that dependency-injection completed). Used as the readinessProbe.
func (h *Handler) HandleReadiness(w http.ResponseWriter, _ *http.Request) {
	if h.llm == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"reason": "llm provider not configured",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
