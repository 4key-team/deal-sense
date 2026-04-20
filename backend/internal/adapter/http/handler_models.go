package http

import (
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func (h *Handler) HandleListModels(w http.ResponseWriter, r *http.Request) {
	uc := usecase.NewListModels(h.resolveLLM(r))
	models, err := uc.Execute(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"models": []string{},
			"error":  err.Error(),
		})
		return
	}
	if models == nil {
		models = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"models": models,
	})
}
