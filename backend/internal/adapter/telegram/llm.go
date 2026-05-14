package telegram

import (
	"context"
	"errors"
	"io"
	"log/slog"

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
// per-chat LLM provider settings. Persistence + validation lives in
// LLMSettingsService; per-chat wizard state in LLMWizardSessions.
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

// errLLMNotImplemented is the RED-phase sentinel returned by every method
// so the package compiles but the behavioural tests fail.
var errLLMNotImplemented = errors.New("llm: not implemented")

// HandleCommand dispatches on the subcommand following "/llm".
//
//	/llm         → show current settings (masked) or hint to create them
//	/llm edit    → reset the wizard at StepLLMProvider and ask the first
//	               question
//	/llm clear   → delete saved settings (revert to server default)
func (h *LLMHandler) HandleCommand(ctx context.Context, u *Update) error {
	_ = ctx
	_ = u
	return errLLMNotImplemented
}

// HandleWizardInput consumes one user message while an /llm wizard session
// is active. /cancel aborts the wizard; otherwise the message fills the
// current step's field, the session advances, and the next question (or
// the final confirmation) is sent.
func (h *LLMHandler) HandleWizardInput(ctx context.Context, u *Update) error {
	_ = ctx
	_ = u
	return errLLMNotImplemented
}
