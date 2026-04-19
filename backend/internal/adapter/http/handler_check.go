package http

import (
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func (h *Handler) HandleCheckConnection(w http.ResponseWriter, r *http.Request) {
	uc := usecase.NewCheckLLMConnection(h.llm)
	result, err := uc.Execute(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":       false,
			"provider": h.llm.Name(),
			"error":    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       result.OK,
		"provider": result.Provider,
	})
}
