package http

import "net/http"

// HandleLiveness reports whether the process is alive. It performs no
// dependency checks — failure here means the process should be killed
// and rescheduled. Used as the Kubernetes/Docker livenessProbe.
//
// Stub for the RED step: returns 500 so the test fails on the wrong
// status code.
func (h *Handler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

// HandleReadiness reports whether the process is ready to serve traffic.
// Returns 200 only when the LLM provider has been wired (sanity check
// that dependency-injection completed). Used as the readinessProbe.
//
// Stub for the RED step.
func (h *Handler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}
