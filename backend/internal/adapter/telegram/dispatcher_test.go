package telegram_test

import (
	"context"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

// newProfileHandlerForRouting wires a real ProfileHandler with in-memory
// dependencies and returns it together with handles tests can poke. The
// dispatcher tests below don't double the ProfileHandler — they assert
// observable side-effects of routing (session state, replies).
func newProfileHandlerForRouting(t *testing.T) (*telegram.ProfileHandler, *telegram.InMemoryWizardSessions, *fakeProfileStore, *fakeReplier) {
	t.Helper()
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	rep := &fakeReplier{}
	h := telegram.NewProfileHandler(store, sessions, rep)
	return h, sessions, store, rep
}

func TestRouteWizardOrProfile_ProfileCommand_HandledByProfileHandler(t *testing.T) {
	h, sessions, _, rep := newProfileHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/profile"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if !handled {
		t.Fatal("/profile must be handled")
	}
	if len(rep.texts) != 1 {
		t.Errorf("expected one reply from ProfileHandler, got %d", len(rep.texts))
	}
}

func TestRouteWizardOrProfile_ProfileEditCommand_Handled(t *testing.T) {
	h, sessions, _, _ := newProfileHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/profile edit"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if !handled {
		t.Fatal("/profile edit must be handled")
	}
	if _, ok := sessions.Get(42); !ok {
		t.Error("wizard session should have been started")
	}
}

func TestRouteWizardOrProfile_FreeTextWithActiveSession_RoutedToWizardInput(t *testing.T) {
	h, sessions, _, _ := newProfileHandlerForRouting(t)
	sessions.Set(42, &telegram.WizardState{ChatID: 42, Step: telegram.StepName, Draft: &telegram.ProfileDraft{}})

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "Acme Corp"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if !handled {
		t.Fatal("free text during wizard session must be handled")
	}
	state, ok := sessions.Get(42)
	if !ok {
		t.Fatal("session still expected after first input")
	}
	if state.Draft.Name != "Acme Corp" {
		t.Errorf("Draft.Name = %q, want %q (wizard input not applied)", state.Draft.Name, "Acme Corp")
	}
	if state.Step != telegram.StepTeamSize {
		t.Errorf("Step = %q, want advance to StepTeamSize", state.Step)
	}
}

func TestRouteWizardOrProfile_CancelWithSession_RoutedAndClearsSession(t *testing.T) {
	h, sessions, _, rep := newProfileHandlerForRouting(t)
	sessions.Set(42, &telegram.WizardState{ChatID: 42, Step: telegram.StepTeamSize, Draft: &telegram.ProfileDraft{Name: "Acme"}})

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/cancel"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
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

func TestRouteWizardOrProfile_FreeTextWithoutSession_NotHandled(t *testing.T) {
	h, sessions, _, rep := newProfileHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "hello bot"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if handled {
		t.Error("free text without session should NOT be handled by profile router")
	}
	if len(rep.texts) != 0 {
		t.Errorf("router should not reply when not handling, got %v", rep.texts)
	}
}

func TestRouteWizardOrProfile_CancelWithoutSession_NotHandled(t *testing.T) {
	// /cancel outside an active wizard isn't ours — the caller's fallback
	// handler can decide what to do. This keeps the router minimal.
	h, sessions, _, _ := newProfileHandlerForRouting(t)

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/cancel"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if handled {
		t.Error("/cancel without session should NOT be claimed")
	}
}

// --- ShouldRouteToProfile (matcher predicate) ---------------------------

func TestShouldRouteToProfile(t *testing.T) {
	sessions := telegram.NewInMemoryWizardSessions()
	sessions.Set(7, &telegram.WizardState{ChatID: 7, Step: telegram.StepName, Draft: &telegram.ProfileDraft{}})

	tests := []struct {
		name   string
		text   string
		chatID int64
		want   bool
	}{
		{"profile command", "/profile", 42, true},
		{"profile edit", "/profile edit", 42, true},
		{"profile clear", "/profile clear", 42, true},
		{"profile with leading whitespace", "  /profile", 42, true},
		{"reply-keyboard profile button", telegram.ButtonProfile, 42, true},
		{"free text + active session", "Acme Corp", 7, true},
		{"cancel + active session", "/cancel", 7, true},
		{"free text + no session", "hello bot", 42, false},
		{"cancel + no session", "/cancel", 42, false},
		{"analyze command + no session", "/analyze", 42, false},
		{"generate + active session for other chat", "/generate", 99, false},
		{"empty text + no session", "", 42, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := telegram.ShouldRouteToProfile(tt.text, tt.chatID, sessions)
			if got != tt.want {
				t.Errorf("ShouldRouteToProfile(%q, %d) = %v, want %v", tt.text, tt.chatID, got, tt.want)
			}
		})
	}
}

func TestRouteWizardOrProfile_ButtonProfile_DispatchesToCommand(t *testing.T) {
	// Tap on the reply-keyboard "👤 Профиль компании" button arrives as a
	// plain text message. RouteWizardOrProfile must treat it as /profile.
	h, sessions, store, _ := newProfileHandlerForRouting(t)
	// No active session, no pre-saved profile → /profile shows the "empty
	// profile" message.
	_ = store // suppress unused

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: telegram.ButtonProfile},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if !handled {
		t.Fatal("button-profile tap must be claimed by the profile router")
	}
}

func TestRouteWizardOrProfile_ProfileCommandTakesPrecedenceOverSession(t *testing.T) {
	// User typed /profile edit while a wizard was already running — handle
	// the command (resets wizard) instead of routing as wizard input.
	h, sessions, _, _ := newProfileHandlerForRouting(t)
	sessions.Set(42, &telegram.WizardState{ChatID: 42, Step: telegram.StepExtra, Draft: &telegram.ProfileDraft{Name: "Old"}})

	handled, err := telegram.RouteWizardOrProfile(
		context.Background(),
		&telegram.Update{ChatID: 42, Text: "/profile edit"},
		h, sessions,
	)
	if err != nil {
		t.Fatalf("RouteWizardOrProfile: %v", err)
	}
	if !handled {
		t.Fatal("/profile command must be handled")
	}
	state, _ := sessions.Get(42)
	if state.Step != telegram.StepName {
		t.Errorf("Step = %q, want fresh StepName after /profile edit", state.Step)
	}
	if state.Draft.Name != "" {
		t.Errorf("Draft.Name = %q, want fresh empty draft", state.Draft.Name)
	}
}
