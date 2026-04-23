package http

import (
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func (h *Handler) HandleCheckConnection(w http.ResponseWriter, r *http.Request) {
	llm := h.resolveLLM(r)
	h.logger.Debug("checking LLM connection", "provider", llm.Name())

	uc := usecase.NewCheckLLMConnection(llm)
	result, err := uc.Execute(r.Context())
	if err != nil {
		h.logger.Warn("LLM connection check failed", "provider", llm.Name(), "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"ok":       false,
			"provider": llm.Name(),
			"error":    err.Error(),
		})
		return
	}

	h.logger.Debug("LLM connection OK", "provider", result.Provider)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       result.OK,
		"provider": result.Provider,
	})
}
