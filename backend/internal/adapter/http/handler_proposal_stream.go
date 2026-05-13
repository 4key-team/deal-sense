package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// sseKeepAliveMu guards the package-level interval so tests can shrink
// it without racing the production handler.
var (
	sseKeepAliveMu       sync.RWMutex
	sseKeepAliveInterval = 15 * time.Second
)

func currentSSEKeepAlive() time.Duration {
	sseKeepAliveMu.RLock()
	defer sseKeepAliveMu.RUnlock()
	return sseKeepAliveInterval
}

// HandleGenerateProposalStream is the Server-Sent Events twin of
// HandleGenerateProposal. It keeps the TCP connection warm with
// `event: progress` ticks every sseKeepAliveInterval while the
// underlying LLM call runs, then emits a final `event: result` with
// the same JSON payload /generate returns. Errors arrive as
// `event: error`.
//
// Why a separate endpoint instead of bolting streaming onto /generate:
// existing clients (and the Telegram bot's HTTP adapter) consume the
// plain JSON shape; SSE is opt-in for the web frontend where browser
// idle-connection timeouts (~60-120s on Safari/Chrome via Docker
// Desktop networking) kill long requests. The Telegram path is async
// on the TG side and does not need this.
func (h *Handler) HandleGenerateProposalStream(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	_, header, err := r.FormFile("template")
	var templateData []byte
	var templateName string
	if err != nil {
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

	var contextFiles []usecase.FileInput
	for _, fh := range r.MultipartForm.File["context"] {
		data := mustReadMultipartFile(fh)
		if strings.EqualFold(filepath.Ext(fh.Filename), ".zip") {
			extracted, err := usecase.ExtractZip(data)
			if err != nil {
				writeError(w, http.StatusBadRequest, "zip: "+err.Error())
				return
			}
			contextFiles = append(contextFiles, extracted...)
			continue
		}
		fi, err := usecase.NewFileInput(fh.Filename, data)
		if err != nil {
			continue
		}
		contextFiles = append(contextFiles, fi)
	}

	langName := resolveLang(r)
	llmProvider := h.resolveLLM(r)

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

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported by server")
		return
	}

	// Disable the server-wide WriteTimeout for this connection so it
	// can stay open as long as the LLM is thinking. Each progress
	// frame is a separate Write, but http.Server's WriteTimeout is a
	// hard deadline from response-start, not per-Write — without this
	// reset, a generation longer than WriteTimeout drops mid-stream.
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{}) //nolint:errcheck // best-effort; handler still works if not supported
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disables proxy buffering when behind nginx
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	type genResult struct {
		proposal *domain.Proposal
		usage    domain.TokenUsage
		err      error
	}
	done := make(chan genResult, 1)
	// Run generation in its own goroutine so the SSE loop can interleave
	// keep-alive frames. The use case respects r.Context for cancellation.
	go func() {
		proposal, usage, err := uc.Execute(r.Context(), templateName, templateData, contextFiles, userParams)
		done <- genResult{proposal: proposal, usage: usage, err: err}
	}()

	ticker := time.NewTicker(currentSSEKeepAlive())
	defer ticker.Stop()

	writeEvent := func(event, data string) {
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
			h.logger.Debug("sse write failed", "event", event, "err", err)
			return
		}
		flusher.Flush()
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			writeEvent("progress", fmt.Sprintf(`{"ts":%d}`, time.Now().Unix()))
		case res := <-done:
			if res.err != nil {
				h.logger.Error("proposal generation failed (stream)", "err", res.err)
				payload, _ := json.Marshal(map[string]string{
					"error": mapErrorToUserMessage(res.err.Error(), langName),
				})
				writeEvent("error", string(payload))
				return
			}
			writeEvent("result", buildProposalJSON(res.proposal, res.usage))
			return
		}
	}
}

// buildProposalJSON mirrors HandleGenerateProposal's response shape so
// the streaming and non-streaming endpoints stay interchangeable.
func buildProposalJSON(result *domain.Proposal, usage domain.TokenUsage) string {
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

	payload := map[string]any{
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
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

// ensure context import retained even if compiler-optimized.
var _ = context.Background
