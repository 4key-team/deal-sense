package http

import (
	"net/http"
	"path/filepath"
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

	h.logger.Debug("tender analysis request", "files", len(files), "lang", langName)

	var inputs []usecase.FileInput
	for _, fh := range files {
		data := mustReadMultipartFile(fh)
		ext := strings.ToLower(filepath.Ext(fh.Filename))

		if ext == ".zip" {
			extracted, err := usecase.ExtractZip(data)
			if err != nil {
				h.logger.Debug("zip extraction failed", "file", fh.Filename, "err", err)
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			h.logger.Debug("zip extracted", "file", fh.Filename, "extracted", len(extracted))
			inputs = append(inputs, extracted...)
			continue
		}

		fi, err := usecase.NewFileInput(fh.Filename, data)
		if err != nil {
			writeError(w, http.StatusBadRequest, "unsupported file type: "+fh.Filename)
			return
		}
		inputs = append(inputs, fi)
	}

	llmProvider := h.resolveLLM(r)
	h.logger.Debug("starting tender analysis", "provider", llmProvider.Name(), "documents", len(inputs))

	uc := usecase.NewAnalyzeTender(llmProvider, h.parser, h.tenderPrompt(langName))
	result, usage, err := uc.Execute(r.Context(), inputs, companyProfile)
	if err != nil {
		h.logger.Error("tender analysis failed", "err", err)
		writeError(w, http.StatusInternalServerError, mapErrorToUserMessage(err.Error(), langName))
		return
	}

	h.logger.Info("tender analysis complete",
		"verdict", string(result.Verdict()),
		"score", result.Score().Value(),
		"prompt_tokens", usage.PromptTokens(),
		"completion_tokens", usage.CompletionTokens(),
	)

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
