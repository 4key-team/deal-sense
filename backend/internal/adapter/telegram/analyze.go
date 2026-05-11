package telegram

import (
	"context"

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

// Handle is a stub for the RED step — returns nil without contacting any
// dependency, so every behavioural test fails on missing side effects.
func (h *AnalyzeHandler) Handle(ctx context.Context, u *Update) error {
	return nil
}

// FormatAnalyzeReply renders a backend response into a chat-friendly message.
// Stub for RED.
func FormatAnalyzeReply(r *usecase.AnalyzeTenderResponse) string {
	return ""
}
