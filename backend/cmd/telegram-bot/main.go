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
	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
	"github.com/daniil/deal-sense/backend/internal/adapter/profilestore"
	telegramadapter "github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

// wizardSweepInterval is how often the in-memory wizard session store is
// scanned for stale entries. The TTL itself is owned by the session store
// (defaultSessionTTL); the sweep cadence just needs to be << TTL.
const wizardSweepInterval = 5 * time.Minute

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
	allowlist, err := auth.ParseAllowlist(cfg.AllowlistUserIDs)
	if err != nil {
		return fmt.Errorf("allowlist: %w", err)
	}
	if allowlist.IsOpen() {
		logger.Warn("allowlist is open — any Telegram user can interact with this bot; set ALLOWLIST_USER_IDS or configure via /settings for production")
	}

	api := dealsenseapi.NewHTTPClient(cfg.APIBaseURL, cfg.APIKey, &http.Client{Timeout: 5 * time.Minute})

	profiles, err := profilestore.NewFileStore(cfg.ProfileStorePath)
	if err != nil {
		return fmt.Errorf("profile store: %w", err)
	}
	wizardSessions := telegramadapter.NewInMemoryWizardSessions()
	collector := metrics.NewCollector()

	// botRef captures the constructed bot so the deny-notice middleware can
	// send messages. The middleware is built before bot.New (it has to be —
	// it goes into the Option set), so we hand it a pointer that's filled in
	// by the line below; b stays nil only for the duration of bot.New itself.
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
		bot.WithMiddlewares(allowlistMiddleware(allowlist, denySender, collector)),
		bot.WithDefaultHandler(defaultHandler(logger)),
	}
	opts = append(opts, extraOpts...)

	b, err = bot.New(cfg.BotToken, opts...)
	if err != nil {
		return fmt.Errorf("bot.New: %w", err)
	}

	replier := &botReplier{b: b, logger: logger}
	profileHandler := telegramadapter.NewProfileHandler(
		profiles, wizardSessions, replier,
		telegramadapter.WithProfileLogger(logger),
		telegramadapter.WithProfileEventCounter(collector),
	)
	analyzeHandler := telegramadapter.NewAnalyzeHandler(api, profiles, replier, telegramadapter.DefaultCompanyFallback, telegramadapter.WithAnalyzeLogger(logger))
	generateHandler := telegramadapter.NewGenerateHandler(api, replier)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix,
		startHandler(logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix,
		helpHandler(logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/analyze", bot.MatchTypePrefix,
		makeAnalyzeHandler(analyzeHandler, b, defaultDocDownloader, logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/generate", bot.MatchTypePrefix,
		makeGenerateHandler(generateHandler, b, defaultDocDownloader, logger))
	b.RegisterHandlerMatchFunc(
		profileMatcher(wizardSessions),
		makeProfileRouteHandler(profileHandler, wizardSessions, logger),
	)

	logger.Info("telegram bot starting",
		"api_base", cfg.APIBaseURL,
		"allowlist_size", len(cfg.AllowlistUserIDs),
		"api_key_set", cfg.APIKey != "",
		"metrics_port", cfg.MetricsPort,
	)

	go runWizardSweeper(ctx, wizardSessions, wizardSweepInterval, logger, collector)

	if cfg.MetricsPort > 0 {
		addr := fmt.Sprintf(":%d", cfg.MetricsPort)
		go func() {
			if err := runMetricsServer(ctx, addr, collector, logger); err != nil {
				logger.Error("metrics listener", "addr", addr, "err", err)
			}
		}()
	}

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

// startHandler greets the user on /start with the same command list
// MsgHelp uses, so the very first interaction is self-documenting.
func startHandler(logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, u *models.Update) {
		if u.Message == nil {
			return
		}
		logger.Debug("start", "chat_id", u.Message.Chat.ID)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: u.Message.Chat.ID,
			Text:   telegramadapter.MsgStart,
		})
	}
}

// helpHandler responds to /help with the full command reference.
func helpHandler(logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, u *models.Update) {
		if u.Message == nil {
			return
		}
		logger.Debug("help", "chat_id", u.Message.Chat.ID)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: u.Message.Chat.ID,
			Text:   telegramadapter.MsgHelp,
		})
	}
}

// runWizardSweeper drives the wizard-session TTL sweep on a ticker, logs
// non-zero evictions and increments dealsense_bot_events_total{event="wizard_evicted"}
// when a counter is supplied. Lives at the cmd layer so observability stays
// out of the adapter (sessions.Run remains a pure utility for unit tests).
// events may be nil — sweep still runs and logs.
func runWizardSweeper(ctx context.Context, sessions *telegramadapter.InMemoryWizardSessions, tick time.Duration, logger *slog.Logger, events *metrics.Collector) {
	if err := ctx.Err(); err != nil {
		return
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := sessions.Sweep(); n > 0 {
				logger.Info("wizard sessions swept", "removed", n)
				if events != nil {
					events.AddBotEvent("wizard_evicted", float64(n))
				}
			}
		}
	}
}

// profileMatcher returns a bot.MatchFunc that delegates to the adapter's
// ShouldRouteToProfile predicate. The bot library invokes the matcher per
// incoming update; if it returns true the paired handler runs and the
// default handler is skipped.
func profileMatcher(sessions telegramadapter.WizardSessions) bot.MatchFunc {
	return func(u *models.Update) bool {
		if u.Message == nil {
			return false
		}
		return telegramadapter.ShouldRouteToProfile(u.Message.Text, u.Message.Chat.ID, sessions)
	}
}

// makeProfileRouteHandler adapts the library's HandlerFunc to our Update DTO
// and delegates routing to RouteWizardOrProfile. The match guarantees the
// route returns handled=true, so we only log transport errors here.
func makeProfileRouteHandler(
	ph *telegramadapter.ProfileHandler,
	sessions telegramadapter.WizardSessions,
	logger *slog.Logger,
) bot.HandlerFunc {
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
		if _, err := telegramadapter.RouteWizardOrProfile(ctx, dto, ph, sessions); err != nil {
			logger.Error("profile/wizard route", "chat_id", dto.ChatID, "err", err)
		}
	}
}

// makeGenerateHandler converts the library Update into our DTO (downloading
// the attached template via dl) and delegates to the GenerateHandler.
func makeGenerateHandler(h *telegramadapter.GenerateHandler, b *bot.Bot, dl docDownloader, logger *slog.Logger) bot.HandlerFunc {
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
				logger.Error("download template", "err", err)
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
			logger.Error("generate handle", "err", err)
		}
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
