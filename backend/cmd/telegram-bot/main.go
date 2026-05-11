package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/daniil/deal-sense/backend/internal/adapter/dealsenseapi"
	telegramadapter "github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
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

func main() {
	cfg, err := telegramadapter.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger, cfg, nil); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

// run wires the bot. Pass extraOpts to inject test-only options (e.g.
// WithServerURL pointing at httptest.Server).
func run(ctx context.Context, logger *slog.Logger, cfg telegramadapter.Config, extraOpts []bot.Option) error {
	allowlist, err := auth.NewAllowlist(cfg.AllowlistUserIDs)
	if err != nil {
		return fmt.Errorf("allowlist: %w", err)
	}

	api := dealsenseapi.NewHTTPClient(cfg.APIBaseURL, cfg.APIKey, &http.Client{Timeout: 5 * time.Minute})

	// botRef captures the constructed bot for closures that need to send
	// messages but are themselves constructed before bot.New (middlewares
	// must be passed to bot.New as options). Using a pointer-to-pointer
	// closure rather than a package-level mutable seam.
	var b *bot.Bot
	denySender := func(ctx context.Context, chatID int64, text string) {
		if b == nil {
			return
		}
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text}); err != nil {
			logger.Error("deny notice send", "err", err)
		}
	}

	opts := []bot.Option{
		bot.WithMiddlewares(allowlistMiddleware(allowlist, denySender)),
		bot.WithDefaultHandler(defaultHandler(logger)),
	}
	opts = append(opts, extraOpts...)

	b, err = bot.New(cfg.BotToken, opts...)
	if err != nil {
		return fmt.Errorf("bot.New: %w", err)
	}

	replier := &botReplier{b: b, logger: logger}
	analyzeHandler := telegramadapter.NewAnalyzeHandler(api, replier, defaultCompanyProfile)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/analyze", bot.MatchTypePrefix,
		makeAnalyzeHandler(analyzeHandler, b, defaultDocDownloader, logger))

	logger.Info("telegram bot starting",
		"api_base", cfg.APIBaseURL,
		"allowlist_size", len(cfg.AllowlistUserIDs),
		"api_key_set", cfg.APIKey != "",
	)
	b.Start(ctx)
	logger.Info("telegram bot stopped")
	return nil
}

// botReplier wraps *bot.Bot so handlers can post messages without taking
// the third-party type as a dependency.
type botReplier struct {
	b      *bot.Bot
	logger *slog.Logger
}

func (r *botReplier) Reply(ctx context.Context, chatID int64, text string) error {
	_, err := r.b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		r.logger.Error("send message", "chat_id", chatID, "err", err)
	}
	return err
}

// ReplyDocument uploads data as a Telegram document with the given filename
// and caption.
func (r *botReplier) ReplyDocument(ctx context.Context, chatID int64, filename string, data []byte, caption string) error {
	_, err := r.b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileUpload{
			Filename: filename,
			Data:     bytes.NewReader(data),
		},
		Caption: caption,
	})
	if err != nil {
		r.logger.Error("send document", "chat_id", chatID, "filename", filename, "err", err)
	}
	return err
}

func defaultHandler(logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, u *models.Update) {
		if u.Message == nil {
			return
		}
		logger.Debug("fallback", "text", u.Message.Text)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: u.Message.Chat.ID,
			Text:   telegramadapter.MsgFallbackHint,
		})
	}
}

// makeAnalyzeHandler converts the library Update into our DTO (downloading
// any attached document via dl) and delegates to the AnalyzeHandler.
func makeAnalyzeHandler(h *telegramadapter.AnalyzeHandler, b *bot.Bot, dl docDownloader, logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, _ *bot.Bot, u *models.Update) {
		if u.Message == nil {
			return
		}
		dto := &telegramadapter.Update{
			ChatID: u.Message.Chat.ID,
			Text:   u.Message.Text,
		}
		if u.Message.From != nil {
			dto.UserID = u.Message.From.ID
		}

		doc := documentFromMessage(u.Message)
		if doc != nil {
			data, filename, err := dl(ctx, b, doc)
			if err != nil {
				logger.Error("download document", "err", err)
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: u.Message.Chat.ID,
					Text:   telegramadapter.MsgDownloadFailed + " " + err.Error(),
				})
				return
			}
			dto.Document = &telegramadapter.Document{
				FileID:   doc.FileID,
				Filename: filename,
				Data:     data,
			}
		}

		if err := h.Handle(ctx, dto); err != nil {
			logger.Error("analyze handle", "err", err)
		}
	}
}

// documentFromMessage finds an attached document — directly on the message
// or in a reply target, whichever comes first.
func documentFromMessage(m *models.Message) *models.Document {
	if m.Document != nil {
		return m.Document
	}
	if m.ReplyToMessage != nil && m.ReplyToMessage.Document != nil {
		return m.ReplyToMessage.Document
	}
	return nil
}

// defaultDocDownloader resolves the file URL via GetFile and downloads the
// body. Network and read errors are surfaced for user-facing reporting.
func defaultDocDownloader(ctx context.Context, b *bot.Bot, doc *models.Document) ([]byte, string, error) {
	file, err := b.GetFile(ctx, &bot.GetFileParams{FileID: doc.FileID})
	if err != nil {
		return nil, "", fmt.Errorf("get file: %w", err)
	}
	url := b.FileDownloadLink(file)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("new request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("download returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}
	filename := doc.FileName
	if filename == "" {
		filename = "document"
	}
	return data, filename, nil
}
