package http

import "net/http"

func (h *Handler) HandleListProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": h.providers,
	})
}
