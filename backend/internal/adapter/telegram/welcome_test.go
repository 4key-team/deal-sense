package telegram_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

var errStoreDown = errors.New("store down")

func TestWelcomeMessage_NoService_ReturnsBaseGreeting(t *testing.T) {
	// Defensive: a bot wired without an LLM settings service can't detect
	// onboarding state, so it falls back to the plain greeting.
	got := telegram.WelcomeMessage(context.Background(), 42, nil, false)
	if got != telegram.MsgStart {
		t.Errorf("WelcomeMessage(nil) = %q, want MsgStart", got)
	}
}

func TestWelcomeMessage_NoLLMSettings_RequireMode_AppendsOnboardingCTA(t *testing.T) {
	// BYOK enforce + chat without /llm settings: the welcome must call out
	// /llm edit as the very next step so the user knows what to do before
	// /analyze fails.
	svc := newFakeLLMService() // empty store
	got := telegram.WelcomeMessage(context.Background(), 42, svc, true)
	if !strings.Contains(got, telegram.MsgStart) {
		t.Errorf("WelcomeMessage missing base MsgStart, got %q", got)
	}
	if !strings.Contains(got, "/llm edit") {
		t.Errorf("WelcomeMessage missing /llm edit CTA, got %q", got)
	}
}

func TestWelcomeMessage_NoLLMSettings_LegacyMode_NoCTA(t *testing.T) {
	// Legacy single-tenant mode (require=false) doesn't force the user to
	// set up /llm, so the welcome stays clean — env LLM_* is the fallback.
	svc := newFakeLLMService()
	got := telegram.WelcomeMessage(context.Background(), 42, svc, false)
	if got != telegram.MsgStart {
		t.Errorf("WelcomeMessage in legacy mode = %q, want MsgStart unchanged", got)
	}
}

func TestWelcomeMessage_HasLLMSettings_RequireMode_NoCTA(t *testing.T) {
	// Configured chat: the user is already set up; no need to nag.
	svc := newFakeLLMService()
	cfg, err := domain.NewLLMSettings("openai", "", "sk-test1234", "gpt-4o")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	svc.data[42] = cfg

	got := telegram.WelcomeMessage(context.Background(), 42, svc, true)
	if got != telegram.MsgStart {
		t.Errorf("WelcomeMessage for configured chat = %q, want MsgStart unchanged", got)
	}
}

func TestWelcomeMessage_SettingsLookupError_FailsClosed_ToCTA(t *testing.T) {
	// Treat a store error in BYOK mode as "no settings" so we steer the
	// user toward /llm edit. The alternative (silently dropping the CTA)
	// would hide the onboarding cue behind a transient infra blip.
	svc := newFakeLLMService()
	svc.getErr = errStoreDown
	got := telegram.WelcomeMessage(context.Background(), 42, svc, true)
	if !strings.Contains(got, "/llm edit") {
		t.Errorf("WelcomeMessage on store error should still nudge to /llm edit, got %q", got)
	}
}
