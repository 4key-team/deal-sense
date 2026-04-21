package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func (h *Handler) HandleGenerateProposal(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	_, header, err := r.FormFile("template")
	if err != nil {
		writeError(w, http.StatusBadRequest, "template file is required")
		return
	}

	templateData := mustReadMultipartFile(header)

	var userParams map[string]string
	if raw := r.FormValue("params"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &userParams); err != nil {
			writeError(w, http.StatusBadRequest, "invalid params JSON")
			return
		}
	}

	// Parse context files
	var contextFiles []usecase.FileInput
	for _, fh := range r.MultipartForm.File["context"] {
		ext := strings.TrimPrefix(filepath.Ext(fh.Filename), ".")
		ft, err := domain.ParseFileType(ext)
		if err != nil {
			continue
		}
		data := mustReadMultipartFile(fh)
		contextFiles = append(contextFiles, usecase.FileInput{
			Name: fh.Filename,
			Data: data,
			Type: ft,
		})
	}

	langName := resolveLang(r)

	uc := usecase.NewGenerateProposal(h.resolveLLM(r), h.parser, h.template, h.proposalPrompt(langName))
	result, usage, err := uc.Execute(r.Context(), header.Filename, templateData, contextFiles, userParams)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sections := make([]map[string]any, len(result.Sections()))
	for i, s := range result.Sections() {
		sections[i] = map[string]any{
			"title":  s.Title(),
			"status": string(s.Status()),
			"tokens": s.Tokens(),
		}
	}

	docxBase64 := ""
	if len(result.Result()) > 0 {
		docxBase64 = base64.StdEncoding.EncodeToString(result.Result())
	}

	logEntries := make([]map[string]string, len(result.Log()))
	for i, l := range result.Log() {
		logEntries[i] = map[string]string{"time": l.Time(), "msg": l.Msg()}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"template": result.TemplateName(),
		"summary":  result.Summary(),
		"meta":     result.Meta(),
		"sections": sections,
		"log":      logEntries,
		"docx":     docxBase64,
		"usage": map[string]int{
			"prompt_tokens":     usage.PromptTokens(),
			"completion_tokens": usage.CompletionTokens(),
			"total_tokens":      usage.TotalTokens(),
		},
	})
}

// HandleDownloadProposal serves the generated .docx file.