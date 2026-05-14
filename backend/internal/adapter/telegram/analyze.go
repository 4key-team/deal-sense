package telegram

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// Update is the minimal slice of an incoming Telegram update the bot's
// command handlers consume. The runtime adapter in cmd/telegram-bot
// translates models.Update from the library into this DTO so handlers
// stay free of third-party types.
type Update struct {
	UserID   int64
	ChatID   int64
	Text     string
	Document *Document
}

// Document is a Telegram document attached to a message, with its body
// already downloaded by the runtime.
type Document struct {
	FileID   string
	Filename string
	Data     []byte
}

// Replier posts a text message or a document back to the originating chat.
// Implementations wrap bot.SendMessage / bot.SendDocument.
type Replier interface {
	Reply(ctx context.Context, chatID int64, text string) error
	ReplyDocument(ctx context.Context, chatID int64, filename string, data []byte, caption string) error
}

// AnalyzeOption tunes an AnalyzeHandler. Use WithAnalyzeLogger to wire an
// slog.Logger; further options can be added without breaking callers.
type AnalyzeOption func(*AnalyzeHandler)

// WithAnalyzeLogger injects a logger for structured observability events
// (profile lookup outcomes, store errors). Nil is ignored — handler keeps
// its discard default.
func WithAnalyzeLogger(l *slog.Logger) AnalyzeOption {
	return func(h *AnalyzeHandler) {
		if l != nil {
			h.logger = l
		}
	}
}

// WithAnalyzeLLMService wires the per-chat LLM settings service. When set
// and a chat has saved settings, /analyze forwards them to the backend
// as an LLMOverride; missing settings degrade silently to backend default.
// Nil is ignored.
func WithAnalyzeLLMService(svc LLMSettingsService) AnalyzeOption {
	return func(h *AnalyzeHandler) {
		if svc != nil {
			h.llm = svc
		}
	}
}

// WithAnalyzeRequirePerChatLLM toggles BYOK enforcement. When true, /analyze
// short-circuits with msgLLMRequired for any chat that has not configured
// /llm — the backend is never called. Default (false here) preserves the
// legacy single-tenant behaviour; production wires this from
// cfg.RequirePerChatLLM which itself defaults to true.
func WithAnalyzeRequirePerChatLLM(v bool) AnalyzeOption {
	return func(h *AnalyzeHandler) {
		h.requireLLM = v
	}
}

// AnalyzeHandler implements the /analyze command flow. The per-chat company
// profile is fetched from ProfileStore; if it is missing or the store errors
// the fallback string is used so analyze never blocks on profile lookup.
// llm is optional — when set, per-chat LLM settings override the backend
// default for this request.
type AnalyzeHandler struct {
	api        usecase.APIClient
	profiles   usecase.ProfileStore
	replier    Replier
	fallback   string
	logger     *slog.Logger
	llm        LLMSettingsService
	requireLLM bool
}

// NewAnalyzeHandler wires the dependencies for /analyze. profiles may be nil
// — analyze degrades gracefully to fallback in that case. Options are
// applied after defaults; omitting WithAnalyzeLogger discards events.
func NewAnalyzeHandler(api usecase.APIClient, profiles usecase.ProfileStore, replier Replier, fallback string, opts ...AnalyzeOption) *AnalyzeHandler {
	h := &AnalyzeHandler{
		api:      api,
		profiles: profiles,
		replier:  replier,
		fallback: fallback,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Handle routes /analyze. Without an attached document it asks the user to
// reply with one; with a document it calls the backend and posts the result.
func (h *AnalyzeHandler) Handle(ctx context.Context, u *Update) error {
	if u.Document == nil {
		return h.replier.Reply(ctx, u.ChatID, msgAttachFile)
	}
	resp, err := h.api.AnalyzeTender(ctx, usecase.AnalyzeTenderRequest{
		File:           u.Document.Data,
		Filename:       u.Document.Filename,
		CompanyProfile: h.profileFor(ctx, u.ChatID),
		LLM:            h.llmOverrideFor(ctx, u.ChatID),
	})
	if err != nil {
		return h.replier.Reply(ctx, u.ChatID, fmt.Sprintf("%s %s", msgAnalysisErrorPrefix, err.Error()))
	}
	return h.replier.Reply(ctx, u.ChatID, FormatAnalyzeReply(resp))
}

// llmOverrideFor returns the per-chat LLM provider override or a zero
// LLMOverride when the chat has no settings or the service is not wired.
// Lookup errors degrade to the zero value (backend default) and are
// logged so operators can tell silent fallback from misconfiguration.
func (h *AnalyzeHandler) llmOverrideFor(ctx context.Context, chatID int64) usecase.LLMOverride {
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

// profileFor returns the per-chat profile's rendered text or the fallback.
// Lookup errors fall back rather than aborting — a missing profile is a soft
// failure for analyze. The lookup outcome is logged so operators can tell
// "user used per-chat profile" from "user fell back to defaults".
func (h *AnalyzeHandler) profileFor(ctx context.Context, chatID int64) string {
	if h.profiles == nil {
		return h.fallback
	}
	p, ok, err := h.profiles.Get(ctx, chatID)
	if err != nil {
		h.logger.ErrorContext(ctx, "profile lookup failed; using fallback", "chat_id", chatID, "err", err)
		return h.fallback
	}
	if !ok {
		h.logger.InfoContext(ctx, "no per-chat profile; using fallback", "chat_id", chatID)
		return h.fallback
	}
	h.logger.DebugContext(ctx, "per-chat profile applied", "chat_id", chatID)
	return p.Render()
}

// FormatAnalyzeReply renders a backend response into a chat-friendly message.
func FormatAnalyzeReply(r *usecase.AnalyzeTenderResponse) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Verdict: %s | Score: %.2f", r.Verdict, r.Score)
	if r.Risk != "" {
		fmt.Fprintf(&b, " | Risk: %s", r.Risk)
	}
	b.WriteString("\n")
	if r.Summary != "" {
		b.WriteString(r.Summary)
		b.WriteString("\n")
	}
	if len(r.Pros) > 0 {
		b.WriteString("\nPros:\n")
		for _, p := range r.Pros {
			fmt.Fprintf(&b, "+ %s — %s\n", p.Title, p.Desc)
		}
	}
	if len(r.Cons) > 0 {
		b.WriteString("\nCons:\n")
		for _, c := range r.Cons {
			fmt.Fprintf(&b, "- %s — %s\n", c.Title, c.Desc)
		}
	}
	if len(r.Requirements) > 0 {
		b.WriteString("\nRequirements:\n")
		for _, q := range r.Requirements {
			fmt.Fprintf(&b, "• %s (%s)\n", q.Label, q.Status)
		}
	}
	if r.Effort != "" {
		fmt.Fprintf(&b, "\nEffort: %s\n", r.Effort)
	}
	return b.String()
}
