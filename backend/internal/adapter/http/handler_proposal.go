package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func (h *Handler) HandleGenerateProposal(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	_, header, err := r.FormFile("template")
	var templateData []byte
	var templateName string
	if err != nil {
		// No template file — allow clean mode if generative engine is available.
		if h.generativeEngine == nil {
			writeError(w, http.StatusBadRequest, "template file is required")
			return
		}
	} else {
		templateData = mustReadMultipartFile(header)
		templateName = header.Filename
	}

	var userParams map[string]string
	if raw := r.FormValue("params"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &userParams); err != nil {
			writeError(w, http.StatusBadRequest, "invalid params JSON")
			return
		}
	}

	// Parse context files (skip unsupported types)
	var contextFiles []usecase.FileInput
	for _, fh := range r.MultipartForm.File["context"] {
		fi, err := usecase.NewFileInput(fh.Filename, mustReadMultipartFile(fh))
		if err != nil {
			continue
		}
		contextFiles = append(contextFiles, fi)
	}

	langName := resolveLang(r)

	llmProvider := h.resolveLLM(r)
	h.logger.Debug("proposal generation request",
		"template", templateName,
		"context_files", len(contextFiles),
		"provider", llmProvider.Name(),
		"lang", langName,
	)

	uc := usecase.NewGenerateProposal(llmProvider, h.parser, h.template, h.proposalPrompt(langName))
	uc.SetLogger(h.logger)
	if h.generativeEngine != nil && h.generativePrompt != nil {
		uc.SetGenerativeEngine(h.generativeEngine, h.generativePrompt(langName))
	}
	if h.pdfGen != nil {
		uc.SetPDFGenerator(h.pdfGen)
	}
	if h.docxToPDF != nil {
		uc.SetDOCXToPDFConverter(h.docxToPDF)
	}
	if h.mdGen != nil {
		uc.SetMDGenerator(h.mdGen)
	}
	result, usage, err := uc.Execute(r.Context(), templateName, templateData, contextFiles, userParams)
	if err != nil {
		h.logger.Error("proposal generation failed", "err", err)
		writeError(w, http.StatusInternalServerError, mapErrorToUserMessage(err.Error(), langName))
		return
	}

	h.logger.Info("proposal generated",
		"template", templateName,
		"sections", len(result.Sections()),
		"docx_size", len(result.Result()),
		"prompt_tokens", usage.PromptTokens(),
		"completion_tokens", usage.CompletionTokens(),
	)

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

	pdfBase64 := ""
	if len(result.PDFResult()) > 0 {
		pdfBase64 = base64.StdEncoding.EncodeToString(result.PDFResult())
	}

	mdContent := ""
	if len(result.MDResult()) > 0 {
		mdContent = string(result.MDResult())
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
		"pdf":      pdfBase64,
		"md":       mdContent,
		"mode":     string(result.Mode()),
		"usage": map[string]int{
			"prompt_tokens":     usage.PromptTokens(),
			"completion_tokens": usage.CompletionTokens(),
			"total_tokens":      usage.TotalTokens(),
		},
	})
}

// HandleDownloadProposal serves the generated .docx file.