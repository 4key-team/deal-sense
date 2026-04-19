package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func (h *Handler) HandleGenerateProposal(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("template")
	if err != nil {
		writeError(w, http.StatusBadRequest, "template file is required")
		return
	}
	defer file.Close()

	templateData, _ := io.ReadAll(file)

	var params map[string]string
	if raw := r.FormValue("params"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			writeError(w, http.StatusBadRequest, "invalid params JSON")
			return
		}
	}

	uc := usecase.NewGenerateProposal(h.llm, h.template)
	result, err := uc.Execute(r.Context(), header.Filename, templateData, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", "attachment; filename=\"proposal.docx\"")
	w.WriteHeader(http.StatusOK)
	w.Write(result.Result())
}
