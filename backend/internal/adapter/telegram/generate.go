package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// GenerateOption tunes a GenerateHandler.
type GenerateOption func(*GenerateHandler)

// WithGenerateLogger injects an slog.Logger for structured logs. Nil is
// ignored — the handler keeps its discard default.
func WithGenerateLogger(l *slog.Logger) GenerateOption {
	return func(h *GenerateHandler) {
		if l != nil {
			h.logger = l
		}
	}
}

// WithGenerateLLMService wires the per-chat LLM settings service. When set
// and a chat has saved settings, /generate forwards them to the backend
// as an LLMOverride; missing settings degrade silently to backend default.
// Nil is ignored.
func WithGenerateLLMService(svc LLMSettingsService) GenerateOption {
	return func(h *GenerateHandler) {
		if svc != nil {
			h.llm = svc
		}
	}
}

// WithGenerateRequirePerChatLLM toggles BYOK enforcement on /generate;
// see WithAnalyzeRequirePerChatLLM.
func WithGenerateRequirePerChatLLM(v bool) GenerateOption {
	return func(h *GenerateHandler) {
		h.requireLLM = v
	}
}

// GenerateHandler implements the /generate command flow. llm is optional;
// when wired, the per-chat LLM settings override the backend default for
// this request.
type GenerateHandler struct {
	api        usecase.APIClient
	replier    Replier
	logger     *slog.Logger
	llm        LLMSettingsService
	requireLLM bool
}

// NewGenerateHandler wires the dependencies for /generate. Options are
// applied after defaults; omitting them keeps the previous behaviour.
func NewGenerateHandler(api usecase.APIClient, replier Replier, opts ...GenerateOption) *GenerateHandler {
	h := &GenerateHandler{
		api:     api,
		replier: replier,
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Handle routes /generate. Without a template document it asks the user
// to reply with one; with a document it calls the backend, attaches the
// resulting DOCX (or falls back to MD), and posts a caption summary.
func (h *GenerateHandler) Handle(ctx context.Context, u *Update) error {
	if u.Document == nil {
		return h.replier.Reply(ctx, u.ChatID, msgAttachTemplate)
	}

	llmOverride := h.llmOverrideFor(ctx, u.ChatID)
	if h.requireLLM && llmOverride.Provider == "" {
		h.logger.InfoContext(ctx, "blocked: BYOK enforce, no per-chat llm", "chat_id", u.ChatID)
		return h.replier.Reply(ctx, u.ChatID, msgLLMRequired)
	}

	resp, err := h.api.GenerateProposal(ctx, usecase.GenerateProposalRequest{
		Template:         u.Document.Data,
		TemplateFilename: u.Document.Filename,
		LLM:              llmOverride,
	})
	if err != nil {
		return h.replier.Reply(ctx, u.ChatID, fmt.Sprintf("%s %s", msgGenerationErrPrefix, err.Error()))
	}

	data, filename := pickArtifact(resp, u.Document.Filename)
	if len(data) == 0 {
		// Nothing to send back — surface mode+summary as text.
		return h.replier.Reply(ctx, u.ChatID, fmt.Sprintf(msgGenerateCaptionFmt, resp.Mode, len(resp.Sections)))
	}

	caption := fmt.Sprintf(msgGenerateCaptionFmt, resp.Mode, len(resp.Sections))
	return h.replier.ReplyDocument(ctx, u.ChatID, filename, data, caption)
}

// llmOverrideFor returns the per-chat LLM provider override or a zero
// LLMOverride when the chat has no settings or the service is not wired.
func (h *GenerateHandler) llmOverrideFor(ctx context.Context, chatID int64) usecase.LLMOverride {
	if h.llm == nil {
		return usecase.LLMOverride{}
	}
	cfg, ok, err := h.llm.Get(ctx, chatID)
	if err != nil {
		h.logger.ErrorContext(ctx, "llm settings lookup failed; using backend default", "chat_id", chatID, "err", err)
		return usecase.LLMOverride{}
	}
	if !ok {
		return usecase.LLMOverride{}
	}
	return usecase.LLMOverride{
		Provider: cfg.Provider(),
		BaseURL:  cfg.BaseURL(),
		APIKey:   cfg.APIKey(),
		Model:    cfg.Model(),
	}
}

// pickArtifact prefers DOCX, then PDF, then MD. Filename swaps the
// extension on the template name to match the chosen artifact.
func pickArtifact(resp *usecase.GenerateProposalResponse, templateName string) ([]byte, string) {
	base := stripExt(templateName)
	if base == "" {
		base = "proposal"
	}
	switch {
	case len(resp.DOCX) > 0:
		return resp.DOCX, base + ".docx"
	case len(resp.PDF) > 0:
		return resp.PDF, base + ".pdf"
	case len(resp.MD) > 0:
		return resp.MD, base + ".md"
	}
	return nil, ""
}

func stripExt(name string) string {
	if i := strings.LastIndex(name, "."); i > 0 {
		return name[:i]
	}
	return name
}
