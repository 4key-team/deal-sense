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
	"github.com/daniil/deal-sense/backend/internal/adapter/llmsettingsstore"
	"github.com/daniil/deal-sense/backend/internal/adapter/metrics"
	"github.com/daniil/deal-sense/backend/internal/adapter/profilestore"
	telegramadapter "github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
	"github.com/daniil/deal-sense/backend/internal/usecase/llmsettings"
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
	llmStore, err := llmsettingsstore.NewFileStore(cfg.LLMStorePath)
	if err != nil {
		return fmt.Errorf("llm settings store: %w", err)
	}
	llmService := llmsettings.NewService(llmStore)
	wizardSessions := telegramadapter.NewInMemoryWizardSessions()
	llmWizardSessions := telegramadapter.NewInMemoryLLMWizardSessions()
	pendingSessions := telegramadapter.NewInMemoryPendingCommandSessions()
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

	// Sync the Telegram "/" autocomplete popup with what the bot actually
	// supports. Failure here is non-fatal — popup is UX sugar, the bot
	// still works without it.
	if _, err := b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: botCommandsList(),
	}); err != nil {
		logger.Warn("set my commands failed; /-popup may be stale", "err", err)
	}

	replier := &botReplier{b: b, logger: logger}
	profileHandler := telegramadapter.NewProfileHandler(
		profiles, wizardSessions, replier,
		telegramadapter.WithProfileLogger(logger),
		telegramadapter.WithProfileEventCounter(collector),
	)
	analyzeHandler := telegramadapter.NewAnalyzeHandler(
		api, profiles, replier, telegramadapter.DefaultCompanyFallback,
		telegramadapter.WithAnalyzeLogger(logger),
		telegramadapter.WithAnalyzeLLMService(llmService),
		telegramadapter.WithAnalyzeRequirePerChatLLM(cfg.RequirePerChatLLM),
	)
	generateHandler := telegramadapter.NewGenerateHandler(
		api, replier,
		telegramadapter.WithGenerateLogger(logger),
		telegramadapter.WithGenerateLLMService(llmService),
		telegramadapter.WithGenerateRequirePerChatLLM(cfg.RequirePerChatLLM),
	)
	llmHandler := telegramadapter.NewLLMHandler(
		llmService, llmWizardSessions, replier,
		telegramadapter.WithLLMLogger(logger),
	)

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix,
		startHandler(logger, llmService, cfg.RequirePerChatLLM))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix,
		helpHandler(logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, telegramadapter.ButtonHelp, bot.MatchTypeExact,
		helpHandler(logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/analyze", bot.MatchTypePrefix,
		makeAnalyzeHandler(analyzeHandler, b, defaultDocDownloader, pendingSessions, logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, telegramadapter.ButtonAnalyze, bot.MatchTypeExact,
		makeAnalyzeHandler(analyzeHandler, b, defaultDocDownloader, pendingSessions, logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/generate", bot.MatchTypePrefix,
		makeGenerateHandler(generateHandler, b, defaultDocDownloader, pendingSessions, logger))
	b.RegisterHandler(bot.HandlerTypeMessageText, telegramadapter.ButtonGenerate, bot.MatchTypeExact,
		makeGenerateHandler(generateHandler, b, defaultDocDownloader, pendingSessions, logger))
	b.RegisterHandlerMatchFunc(
		profileMatcher(wizardSessions),
		makeProfileRouteHandler(profileHandler, wizardSessions, logger),
	)
	b.RegisterHandlerMatchFunc(
		llmMatcher(llmWizardSessions),
		makeLLMRouteHandler(llmHandler, llmWizardSessions, logger),
	)
	b.RegisterHandlerMatchFunc(
		pendingMatcher(pendingSessions),
		makePendingRouteHandler(pendingSessions, analyzeHandler, generateHandler, replier, b, defaultDocDownloader, logger),
	)

	logger.Info("telegram bot starting",
		"api_base", cfg.APIBaseURL,
		"allowlist_size", len(cfg.AllowlistUserIDs),
		"api_key_set", cfg.APIKey != "",
		"metrics_port", cfg.MetricsPort,
	)

	go runWizardSweeper(ctx, wizardSessions, wizardSweepInterval, logger, collector)
	go runLLMWizardSweeper(ctx, llmWizardSessions, wizardSweepInterval, logger, collector)

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

// botCommandsList is the catalogue surfaced by Telegram's "/" autocomplete
// popup via SetMyCommands. Order matches the rough discovery flow:
// onboarding (/start, /help), the two product commands (/analyze,
// /generate), then per-chat configuration (/profile, /llm) and the
// universal escape hatch (/cancel). Descriptions are kept short —
// Telegram truncates aggressively in the popup.
func botCommandsList() []models.BotCommand {
	return []models.BotCommand{
		{Command: "start", Description: "Начало работы"},
		{Command: "help", Description: "Список команд"},
		{Command: "analyze", Description: "Анализ тендера"},
		{Command: "generate", Description: "Создать КП по шаблону"},
		{Command: "go", Description: "Запустить накопленные файлы"},
		{Command: "cancel", Description: "Прервать активный wizard"},
		{Command: "profile", Description: "Профиль компании"},
		{Command: "llm", Description: "Настройки LLM для чата"},
	}
}

// mainReplyKeyboard is the persistent reply keyboard surfaced by /start
// and /help: four top-level commands laid out as two rows of two buttons.
// Telegram delivers a tapped button as a regular text message containing
// the button label, which we route via MatchTypeExact aliases in run().
func mainReplyKeyboard() *models.ReplyKeyboardMarkup {
	return &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{{Text: telegramadapter.ButtonAnalyze}, {Text: telegramadapter.ButtonGenerate}},
			{{Text: telegramadapter.ButtonProfile}, {Text: telegramadapter.ButtonLLM}},
			{{Text: telegramadapter.ButtonHelp}},
		},
		ResizeKeyboard: true,
		IsPersistent:   true,
	}
}

// startHandler greets the user on /start with the same command list
// MsgHelp uses plus the persistent reply keyboard. The greeting body is
// computed by telegramadapter.WelcomeMessage so BYOK-mode chats without
// a per-chat /llm see an explicit onboarding CTA appended.
func startHandler(logger *slog.Logger, llm telegramadapter.LLMSettingsService, requireLLM bool) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, u *models.Update) {
		if u.Message == nil {
			return
		}
		logger.Debug("start", "chat_id", u.Message.Chat.ID)
		text := telegramadapter.WelcomeMessage(ctx, u.Message.Chat.ID, llm, requireLLM)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      u.Message.Chat.ID,
			Text:        text,
			ReplyMarkup: mainReplyKeyboard(),
		})
	}
}

// helpHandler responds to /help with the full command reference and the
// same persistent reply keyboard so the user can re-pin it any time.
func helpHandler(logger *slog.Logger) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, u *models.Update) {
		if u.Message == nil {
			return
		}
		logger.Debug("help", "chat_id", u.Message.Chat.ID)
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      u.Message.Chat.ID,
			Text:        telegramadapter.MsgHelp,
			ReplyMarkup: mainReplyKeyboard(),
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

// llmMatcher mirrors profileMatcher for the /llm command + its wizard.
// Registered AFTER profileMatcher so /cancel inside a profile wizard
// session goes to the profile router, not the llm one.
func llmMatcher(sessions telegramadapter.LLMWizardSessions) bot.MatchFunc {
	return func(u *models.Update) bool {
		if u.Message == nil {
			return false
		}
		return telegramadapter.ShouldRouteToLLM(u.Message.Text, u.Message.Chat.ID, sessions)
	}
}

// makeLLMRouteHandler adapts the library's HandlerFunc to our Update DTO
// and delegates routing to RouteWizardOrLLM. Mirrors makeProfileRouteHandler.
func makeLLMRouteHandler(
	lh *telegramadapter.LLMHandler,
	sessions telegramadapter.LLMWizardSessions,
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
		if _, err := telegramadapter.RouteWizardOrLLM(ctx, dto, lh, sessions); err != nil {
			logger.Error("llm/wizard route", "chat_id", dto.ChatID, "err", err)
		}
	}
}

// runLLMWizardSweeper drives the /llm-wizard-session TTL sweep on a ticker.
// Mirrors runWizardSweeper; lives at the cmd layer so observability stays
// out of the adapter. The "wizard_evicted" counter is shared with the
// profile sweeper — operators count the union by design (both speak the
// same eviction language to the same user).
func runLLMWizardSweeper(ctx context.Context, sessions *telegramadapter.InMemoryLLMWizardSessions, tick time.Duration, logger *slog.Logger, events *metrics.Collector) {
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
				logger.Info("llm wizard sessions swept", "removed", n)
				if events != nil {
					events.AddBotEvent("wizard_evicted", float64(n))
				}
			}
		}
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
// The pending sessions store tracks the two-step "command, then file" flow:
// when the user types /generate without a template, we mark the chat as
// awaiting a generate-template so the next bare upload is routed correctly.
func makeGenerateHandler(h *telegramadapter.GenerateHandler, b *bot.Bot, dl docDownloader, pending *telegramadapter.InMemoryPendingCommandSessions, logger *slog.Logger) bot.HandlerFunc {
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
			if pending != nil {
				pending.Clear(u.Message.Chat.ID)
			}
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
		} else if pending != nil {
			pending.Set(u.Message.Chat.ID, telegramadapter.PendingGenerate)
		}

		if err := h.Handle(ctx, dto); err != nil {
			logger.Error("generate handle", "err", err)
		}
	}
}

// makeAnalyzeHandler converts the library Update into our DTO (downloading
// any attached document via dl) and delegates to the AnalyzeHandler.
// See makeGenerateHandler for the pending-sessions contract.
func makeAnalyzeHandler(h *telegramadapter.AnalyzeHandler, b *bot.Bot, dl docDownloader, pending *telegramadapter.InMemoryPendingCommandSessions, logger *slog.Logger) bot.HandlerFunc {
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
			if pending != nil {
				pending.Clear(u.Message.Chat.ID)
			}
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
		} else if pending != nil {
			pending.Set(u.Message.Chat.ID, telegramadapter.PendingAnalyze)
		}

		if err := h.Handle(ctx, dto); err != nil {
			logger.Error("analyze handle", "err", err)
		}
	}
}

// pendingMatcher returns a MatchFunc that fires whenever an incoming
// message belongs to the multi-file collection wizard — a document or a
// /go / /cancel / free-text response during an active pending session.
// Slash commands other than /go and /cancel fall through so /profile,
// /llm and friends keep working even mid-collection.
func pendingMatcher(pending *telegramadapter.InMemoryPendingCommandSessions) bot.MatchFunc {
	return func(u *models.Update) bool {
		if u == nil || u.Message == nil {
			return false
		}
		hasDoc := documentFromMessage(u.Message) != nil
		return telegramadapter.ShouldRoutePending(u.Message.Text, hasDoc, u.Message.Chat.ID, pending)
	}
}

// makePendingRouteHandler turns each incoming pending-flow message into
// a typed *telegramadapter.Update (downloading the attached document if
// any) and delegates routing to RoutePending. /go finalises the
// dispatch with the collected files; documents append; free text
// receives msgPendingTextHint; /cancel clears.
func makePendingRouteHandler(
	pending *telegramadapter.InMemoryPendingCommandSessions,
	ah *telegramadapter.AnalyzeHandler,
	gh *telegramadapter.GenerateHandler,
	replier telegramadapter.Replier,
	b *bot.Bot,
	dl docDownloader,
	logger *slog.Logger,
) bot.HandlerFunc {
	return func(ctx context.Context, _ *bot.Bot, u *models.Update) {
		if u == nil || u.Message == nil {
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
				logger.Error("download pending document", "err", err)
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

		if _, err := telegramadapter.RoutePending(ctx, dto, pending, ah, gh, replier); err != nil {
			logger.Error("pending route", "chat_id", dto.ChatID, "err", err)
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
