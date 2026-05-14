package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// LLMSettingsService is the narrow adapter-side contract over the
// usecase/llmsettings.Service. Defined here so handler tests can swap in a
// fake without depending on the concrete service struct. *llmsettings.Service
// satisfies this interface implicitly.
type LLMSettingsService interface {
	Get(ctx context.Context, chatID int64) (*domain.LLMSettings, bool, error)
	Update(ctx context.Context, chatID int64, provider, baseURL, apiKey, model string) (*domain.LLMSettings, error)
	Clear(ctx context.Context, chatID int64) error
}

// LLMOption tunes an LLMHandler. WithLLMLogger is the only knob for now.
type LLMOption func(*LLMHandler)

// WithLLMLogger injects an slog.Logger for structured event logging.
// Defaults to a discard handler if omitted.
func WithLLMLogger(l *slog.Logger) LLMOption {
	return func(h *LLMHandler) {
		if l != nil {
			h.logger = l
		}
	}
}

// LLMHandler implements the /llm command and the 4-step wizard that fills
// per-chat LLM provider settings.
type LLMHandler struct {
	settings LLMSettingsService
	sessions LLMWizardSessions
	replier  Replier
	logger   *slog.Logger
}

// NewLLMHandler wires the dependencies for /llm.
func NewLLMHandler(svc LLMSettingsService, sessions LLMWizardSessions, replier Replier, opts ...LLMOption) *LLMHandler {
	h := &LLMHandler{
		settings: svc,
		sessions: sessions,
		replier:  replier,
		logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// HandleCommand dispatches on the subcommand following "/llm".
func (h *LLMHandler) HandleCommand(ctx context.Context, u *Update) error {
	arg := strings.TrimSpace(strings.TrimPrefix(u.Text, "/llm"))
	switch arg {
	case "":
		return h.showSettings(ctx, u.ChatID)
	case "edit":
		return h.startWizard(ctx, u.ChatID)
	case "clear":
		return h.clearSettings(ctx, u.ChatID)
	default:
		return h.replier.Reply(ctx, u.ChatID, msgLLMUnknownCmd)
	}
}

func (h *LLMHandler) showSettings(ctx context.Context, chatID int64) error {
	cfg, ok, err := h.settings.Get(ctx, chatID)
	if err != nil {
		h.logger.ErrorContext(ctx, "llm settings load failed", "chat_id", chatID, "err", err)
		return h.replier.Reply(ctx, chatID, msgLLMLoadError)
	}
	if !ok {
		return h.replier.Reply(ctx, chatID, msgLLMEmpty)
	}
	baseURL := cfg.BaseURL()
	if baseURL == "" {
		baseURL = msgLLMShowDefaultURL
	}
	return h.replier.Reply(ctx, chatID, fmt.Sprintf(msgLLMShowFmt,
		cfg.Provider(), baseURL, cfg.MaskedAPIKey(), cfg.Model()))
}

func (h *LLMHandler) startWizard(ctx context.Context, chatID int64) error {
	h.sessions.Set(chatID, &LLMWizardState{
		ChatID:    chatID,
		Step:      StepLLMProvider,
		Draft:     &LLMSettingsDraft{},
		StartedAt: time.Now(),
	})
	h.logger.InfoContext(ctx, "llm wizard started", "chat_id", chatID)
	return h.replier.Reply(ctx, chatID, msgLLMWizardStart)
}

func (h *LLMHandler) clearSettings(ctx context.Context, chatID int64) error {
	if err := h.settings.Clear(ctx, chatID); err != nil {
		h.logger.ErrorContext(ctx, "llm settings clear failed", "chat_id", chatID, "err", err)
		return h.replier.Reply(ctx, chatID, msgLLMSaveError)
	}
	h.logger.InfoContext(ctx, "llm settings cleared", "chat_id", chatID)
	return h.replier.Reply(ctx, chatID, msgLLMCleared)
}

// HandleWizardInput consumes one user message while an /llm wizard session
// is active. /cancel aborts the wizard; otherwise the message fills the
// current step's field, the session advances, and the next question (or
// the final confirmation) is sent.
func (h *LLMHandler) HandleWizardInput(ctx context.Context, u *Update) error {
	state, ok := h.sessions.Get(u.ChatID)
	if !ok {
		// Defensive: dispatcher should not call this without a session.
		return nil
	}
	text := strings.TrimSpace(u.Text)
	if text == "/cancel" {
		h.sessions.Clear(u.ChatID)
		h.logger.InfoContext(ctx, "llm wizard cancelled", "chat_id", u.ChatID, "step", string(state.Step))
		return h.replier.Reply(ctx, u.ChatID, msgLLMWizardCancelled)
	}

	switch state.Step {
	case StepLLMProvider:
		state.Draft.Provider = parseSentinel(text)
		state.Step = StepLLMBaseURL
		return h.replier.Reply(ctx, u.ChatID, msgLLMWizardBaseURL)
	case StepLLMBaseURL:
		state.Draft.BaseURL = parseSentinel(text)
		state.Step = StepLLMAPIKey
		return h.replier.Reply(ctx, u.ChatID, msgLLMWizardAPIKey)
	case StepLLMAPIKey:
		state.Draft.APIKey = parseSentinel(text)
		state.Step = StepLLMModel
		return h.replier.Reply(ctx, u.ChatID, msgLLMWizardModel)
	case StepLLMModel:
		state.Draft.Model = parseSentinel(text)
		return h.finalize(ctx, u.ChatID, state.Draft)
	default:
		// Unknown step — clear to recover from corrupted state.
		h.sessions.Clear(u.ChatID)
		return h.replier.Reply(ctx, u.ChatID, msgLLMWizardCancelled)
	}
}

// finalize commits the draft to the service. Validation errors from the
// domain constructor are mapped to a user-friendly reply with a short
// explanation; any failure clears the session so the user is not stuck
// answering the same question repeatedly.
func (h *LLMHandler) finalize(ctx context.Context, chatID int64, d *LLMSettingsDraft) error {
	h.sessions.Clear(chatID)

	cfg, err := h.settings.Update(ctx, chatID, d.Provider, d.BaseURL, d.APIKey, d.Model)
	if err != nil {
		// Distinguish domain validation (user-correctable) from infra failure.
		if isLLMValidationError(err) {
			h.logger.InfoContext(ctx, "llm wizard rejected invalid input", "chat_id", chatID, "err", err)
			return h.replier.Reply(ctx, chatID, fmt.Sprintf(msgLLMWizardInvalidFmt, llmValidationHint(err)))
		}
		h.logger.ErrorContext(ctx, "llm settings save failed", "chat_id", chatID, "err", err)
		return h.replier.Reply(ctx, chatID, msgLLMSaveError)
	}
	h.logger.InfoContext(ctx, "llm wizard completed", "chat_id", chatID)
	baseURL := cfg.BaseURL()
	if baseURL == "" {
		baseURL = msgLLMShowDefaultURL
	}
	return h.replier.Reply(ctx, chatID, fmt.Sprintf(msgLLMWizardConfirmFmt,
		cfg.Provider(), baseURL, cfg.MaskedAPIKey(), cfg.Model()))
}

// isLLMValidationError reports whether err comes from the domain
// constructor (user-correctable) rather than from infra (repo, I/O).
func isLLMValidationError(err error) bool {
	return errors.Is(err, domain.ErrEmptyLLMProvider) ||
		errors.Is(err, domain.ErrEmptyLLMAPIKey) ||
		errors.Is(err, domain.ErrEmptyLLMModel) ||
		errors.Is(err, domain.ErrInvalidLLMBaseURL)
}

// llmValidationHint returns a short Russian phrase pinpointing which field
// failed. Defaults to the raw error text — better than nothing if a new
// domain error appears without a matching hint.
func llmValidationHint(err error) string {
	switch {
	case errors.Is(err, domain.ErrEmptyLLMProvider):
		return "Provider обязателен."
	case errors.Is(err, domain.ErrEmptyLLMAPIKey):
		return "API key обязателен."
	case errors.Is(err, domain.ErrEmptyLLMModel):
		return "Модель обязательна."
	case errors.Is(err, domain.ErrInvalidLLMBaseURL):
		return "Base URL должен быть полным (https://…) или пустым."
	default:
		return err.Error()
	}
}
