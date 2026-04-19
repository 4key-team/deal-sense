package http

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

const maxUploadSize = 50 << 20 // 50MB

func (h *Handler) HandleAnalyzeTender(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	companyProfile := strings.TrimSpace(r.FormValue("company_profile"))
	if companyProfile == "" {
		writeError(w, http.StatusBadRequest, "company_profile is required")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "at least one file is required")
		return
	}

	var inputs []usecase.FileInput
	for _, fh := range files {
		ext := strings.TrimPrefix(filepath.Ext(fh.Filename), ".")
		ft, err := domain.ParseFileType(ext)
		if err != nil {
			writeError(w, http.StatusBadRequest, "unsupported file type: "+fh.Filename)
			return
		}

		f, _ := fh.Open()
		data, _ := io.ReadAll(f)
		f.Close()

		inputs = append(inputs, usecase.FileInput{
			Name: fh.Filename,
			Data: data,
			Type: ft,
		})
	}

	uc := usecase.NewAnalyzeTender(h.llm, h.parser)
	result, err := uc.Execute(r.Context(), inputs, companyProfile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verdict": string(result.Verdict()),
		"risk":    string(result.Risk()),
		"score":   result.Score().Value(),
		"summary": result.Summary(),
	})
}
