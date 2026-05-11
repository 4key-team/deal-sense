package telegram

import (
	"context"
	"fmt"
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

// Replier posts a text message back to the originating chat. Implementations
// wrap bot.SendMessage.
type Replier interface {
	Reply(ctx context.Context, chatID int64, text string) error
}

// AnalyzeHandler implements the /analyze command flow.
type AnalyzeHandler struct {
	api     usecase.APIClient
	replier Replier
	profile string
}

// NewAnalyzeHandler wires the dependencies for /analyze.
func NewAnalyzeHandler(api usecase.APIClient, replier Replier, profile string) *AnalyzeHandler {
	return &AnalyzeHandler{api: api, replier: replier, profile: profile}
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
		CompanyProfile: h.profile,
	})
	if err != nil {
		return h.replier.Reply(ctx, u.ChatID, fmt.Sprintf("%s %s", msgAnalysisErrorPrefix, err.Error()))
	}
	return h.replier.Reply(ctx, u.ChatID, FormatAnalyzeReply(resp))
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
