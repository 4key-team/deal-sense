package main

import (
	"context"
	"errors"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	telegramadapter "github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

// defaultCompanyProfile is the static company description fed into every
// /analyze call until Session 2.5 introduces a per-user profile.
const defaultCompanyProfile = "Software development company"

// docDownloader returns the file body and resolved filename for a Telegram
// Document. The default impl talks to the real Telegram API; tests inject
// a deterministic version.
type docDownloader func(ctx context.Context, b *bot.Bot, doc *models.Document) ([]byte, string, error)

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// main is a stub for the RED step — the real wiring lands in the GREEN
// commit. Leaving it empty keeps `go build` green so contract tests run.
func main() {}

// run is a stub: returns nil without starting a bot. Every behavioural test
// (TestRun_StartsAndStops, TestRun_EmptyAllowlistFails, TestRun_BotInitFails,
// TestRun_DenyMiddlewareSendsNotice) fails on the missing side-effects.
func run(ctx context.Context, logger *slog.Logger, cfg telegramadapter.Config, extraOpts []bot.Option) error {
	return nil
}

// botReplier stub — Reply does nothing so TestBotReplier_PropagatesError
// fails on the missing error.
type botReplier struct {
	b      *bot.Bot
	logger *slog.Logger
}

func (r *botReplier) Reply(ctx context.Context, chatID int64, text string) error {
	return nil
}

// defaultHandler stub — handler that never sends, so the fallback tests
// fail their sendMessage-count assertions.
func defaultHandler(logger *slog.Logger) bot.HandlerFunc {
	return func(context.Context, *bot.Bot, *models.Update) {}
}

// makeAnalyzeHandler stub — no-op handler so the /analyze suite fails on
// missing API calls / replies.
func makeAnalyzeHandler(h *telegramadapter.AnalyzeHandler, b *bot.Bot, dl docDownloader, logger *slog.Logger) bot.HandlerFunc {
	return func(context.Context, *bot.Bot, *models.Update) {}
}

// documentFromMessage is a pure helper — included in the RED commit because
// it has no behaviour to drive separately (it's a struct accessor) and the
// test suite needs it compiled to run.
func documentFromMessage(m *models.Message) *models.Document {
	if m.Document != nil {
		return m.Document
	}
	if m.ReplyToMessage != nil && m.ReplyToMessage.Document != nil {
		return m.ReplyToMessage.Document
	}
	return nil
}

// defaultDocDownloader stub — returns an error so download tests fail.
func defaultDocDownloader(ctx context.Context, b *bot.Bot, doc *models.Document) ([]byte, string, error) {
	return nil, "", errors.New("not implemented")
}
