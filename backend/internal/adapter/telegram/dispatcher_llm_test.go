package telegram_test

import (
	"context"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

// newLLMHandlerForRouting wires a real LLMHandler with in-memory deps and
// returns it together with the handles tests can poke. Mirrors
// newProfileHandlerForRouting in dispatcher_test.go.
func newLLMHandlerForRouting(t *testing.T) (*telegram.LLMHandler, *telegram.InMemoryLLMWizardSessions, *fakeLLMService, *fakeReplier) {
	t.Helper()
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	rep := &fakeReplier{}
	h := telegram.NewLLMHandler(svc, sessions, rep)
	return h, sessions, svc, rep
}

// --- ShouldRouteToLLM (matcher predicate) -------------------------------

func TestShouldRouteToLLM(t *testing.T) {
	sessions := telegram.NewInMemoryLLMWizardSessions()
	sessions.Set(7, &telegram.LLMWizardState{
		ChatID: 7, Step: telegram.StepLLMProvider,
		Draft: &telegram.LLMSettingsDraft{},
	})

	tests := []struct {
		name   string
		text   string
		chatID int64
		want   bool
	}{
		{"llm command", "/llm", 42, true},
		{"llm edit", "/llm edit", 42, true},
		{"llm clear", "/llm clear", 42, true},
		{"llm with leading whitespace", "  /llm", 42, true},
		{"reply-keyboard llm button", telegram.ButtonLLM, 42, true},
		{"free text + active session", "openai", 7, true},
		{"cancel + active session", "/cancel", 7, true},
		{"free text + no session", "hello bot", 42, false},
		{"cancel + no session", "/cancel", 42, false},
		{"analyze command + no session", "/analyze", 42, false},
		{"profile command + active llm session for other chat", "/profile", 99, false},
		{"empty text + no session", "", 42, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := telegram.ShouldRouteToLLM(tt.text, tt.chatID, sessions)
			if got != tt.want {
				t.Errorf("ShouldRouteToLLM(%q, %d) = %v, want %v", tt.text, tt.chatID, got, tt.want)
			}
		})
	}
}

// --- RouteWizardOrLLM ---------------------------------------------------

func TestRouteWizardOrLLM_Command_Handled(t *testing.T) {
	h, sessions, _, rep := newLLMHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/llm"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if !handled {
		t.Fatal("/llm must be handled")
	}
	if len(rep.texts) != 1 {
		t.Errorf("expected one reply, got %d", len(rep.texts))
	}
}

func TestRouteWizardOrLLM_EditCommand_StartsSession(t *testing.T) {
	h, sessions, _, _ := newLLMHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/llm edit"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if !handled {
		t.Fatal("/llm edit must be handled")
	}
	if _, ok := sessions.Get(42); !ok {
		t.Error("llm wizard session should have been started")
	}
}

func TestRouteWizardOrLLM_FreeTextWithActiveSession_RoutedToWizardInput(t *testing.T) {
	h, sessions, _, _ := newLLMHandlerForRouting(t)
	sessions.Set(42, &telegram.LLMWizardState{
		ChatID: 42, Step: telegram.StepLLMProvider,
		Draft: &telegram.LLMSettingsDraft{},
	})

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "openai"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if !handled {
		t.Fatal("free text during wizard session must be handled")
	}
	state, ok := sessions.Get(42)
	if !ok {
		t.Fatal("session still expected after first input")
	}
	if state.Draft.Provider != "openai" {
		t.Errorf("Draft.Provider = %q, want openai", state.Draft.Provider)
	}
	if state.Step != telegram.StepLLMBaseURL {
		t.Errorf("Step = %q, want advance to StepLLMBaseURL", state.Step)
	}
}

func TestRouteWizardOrLLM_CancelWithSession_HandledAndClearsSession(t *testing.T) {
	h, sessions, _, rep := newLLMHandlerForRouting(t)
	sessions.Set(42, &telegram.LLMWizardState{
		ChatID: 42, Step: telegram.StepLLMAPIKey,
		Draft: &telegram.LLMSettingsDraft{Provider: "openai"},
	})

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/cancel"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if !handled {
		t.Fatal("/cancel with session must be handled")
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after /cancel")
	}
	if len(rep.texts) == 0 || !strings.Contains(strings.ToLower(rep.texts[0]), "отмен") {
		t.Errorf("expected cancellation reply, got %v", rep.texts)
	}
}

func TestRouteWizardOrLLM_FreeTextWithoutSession_NotHandled(t *testing.T) {
	h, sessions, _, rep := newLLMHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "hello bot"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if handled {
		t.Error("free text without session should NOT be handled by llm router")
	}
	if len(rep.texts) != 0 {
		t.Errorf("router should not reply when not handling, got %v", rep.texts)
	}
}

func TestRouteWizardOrLLM_CancelWithoutSession_NotHandled(t *testing.T) {
	// /cancel outside an /llm wizard isn't ours — caller's profile router
	// or fallback handles it. Keeps the routers composable.
	h, sessions, _, _ := newLLMHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/cancel"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if handled {
		t.Error("/cancel without session should NOT be claimed by llm router")
	}
}

func TestRouteWizardOrLLM_ButtonLLM_DispatchesToCommand(t *testing.T) {
	// Tap on the reply-keyboard "🤖 Настройки LLM" button arrives as a
	// plain text message; the router must treat it as /llm.
	h, sessions, _, rep := newLLMHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: telegram.ButtonLLM},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if !handled {
		t.Fatal("button-llm tap must be claimed by the llm router")
	}
	if len(rep.texts) != 1 {
		t.Errorf("expected one reply from /llm, got %d", len(rep.texts))
	}
}

func TestRouteWizardOrLLM_LLMCommandTakesPrecedenceOverSession(t *testing.T) {
	// User typed /llm edit while a wizard was already running — handle
	// the command (resets wizard) instead of routing as wizard input.
	h, sessions, _, _ := newLLMHandlerForRouting(t)
	sessions.Set(42, &telegram.LLMWizardState{
		ChatID: 42, Step: telegram.StepLLMModel,
		Draft: &telegram.LLMSettingsDraft{Provider: "old"},
	})

	handled, err := telegram.RouteWizardOrLLM(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/llm edit"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrLLM: %v", err)
	}
	if !handled {
		t.Fatal("/llm command must be handled")
	}
	state, _ := sessions.Get(42)
	if state.Step != telegram.StepLLMProvider {
		t.Errorf("Step = %q, want fresh StepLLMProvider after /llm edit", state.Step)
	}
	if state.Draft.Provider != "" {
		t.Errorf("Draft.Provider = %q, want fresh empty draft", state.Draft.Provider)
	}
}
