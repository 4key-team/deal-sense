package http

import (
	"net/http"
	"strings"

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
		companyProfile = "Software development company"
	}

	langName := resolveLang(r)

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "at least one file is required")
		return
	}

	var inputs []usecase.FileInput
	for _, fh := range files {
		fi, err := usecase.NewFileInput(fh.Filename, mustReadMultipartFile(fh))
		if err != nil {
			writeError(w, http.StatusBadRequest, "unsupported file type: "+fh.Filename)
			return
		}
		inputs = append(inputs, fi)
	}

	uc := usecase.NewAnalyzeTender(h.resolveLLM(r), h.parser, h.tenderPrompt(langName))
	result, usage, err := uc.Execute(r.Context(), inputs, companyProfile)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pros := make([]map[string]string, len(result.Pros()))
	for i, p := range result.Pros() {
		pros[i] = map[string]string{"title": p.Title(), "desc": p.Desc()}
	}
	cons := make([]map[string]string, len(result.Cons()))
	for i, c := range result.Cons() {
		cons[i] = map[string]string{"title": c.Title(), "desc": c.Desc()}
	}
	reqs := make([]map[string]string, len(result.Requirements()))
	for i, r := range result.Requirements() {
		reqs[i] = map[string]string{"label": r.Label(), "status": string(r.Status())}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"verdict":      string(result.Verdict()),
		"risk":         string(result.Risk()),
		"score":        result.Score().Value(),
		"summary":      result.Summary(),
		"pros":         pros,
		"cons":         cons,
		"requirements": reqs,
		"effort":       result.Effort(),
		"usage": map[string]int{
			"prompt_tokens":     usage.PromptTokens(),
			"completion_tokens": usage.CompletionTokens(),
			"total_tokens":      usage.TotalTokens(),
		},
	})
}
