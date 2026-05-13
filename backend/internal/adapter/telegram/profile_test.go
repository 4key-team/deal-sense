package telegram_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

// fakeProfileStore is an in-memory ProfileStore for handler tests. It tracks
// call counts so tests can assert side-effects on the store directly.
type fakeProfileStore struct {
	mu       sync.Mutex
	data     map[int64]*domain.CompanyProfile
	getErr   error
	setErr   error
	clearErr error
	setCalls int
	clrCalls int
}

func newFakeProfileStore() *fakeProfileStore {
	return &fakeProfileStore{data: map[int64]*domain.CompanyProfile{}}
}

func (f *fakeProfileStore) Get(_ context.Context, chatID int64) (*domain.CompanyProfile, bool, error) {
	if f.getErr != nil {
		return nil, false, f.getErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.data[chatID]
	return p, ok, nil
}

func (f *fakeProfileStore) Set(_ context.Context, chatID int64, p *domain.CompanyProfile) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[chatID] = p
	f.setCalls++
	return nil
}

func (f *fakeProfileStore) Clear(_ context.Context, chatID int64) error {
	if f.clearErr != nil {
		return f.clearErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, chatID)
	f.clrCalls++
	return nil
}

func TestProfileHandler_NoProfile_ShowsEmptyHint(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile"})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(rep.texts[0], "не заполнен") {
		t.Errorf("reply %q should mention 'не заполнен'", rep.texts[0])
	}
	if !strings.Contains(rep.texts[0], "/profile edit") {
		t.Errorf("reply %q should mention '/profile edit' hint", rep.texts[0])
	}
}

func TestProfileHandler_ExistingProfile_ShowsRenderedText(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	prof, err := domain.NewCompanyProfile("Acme", "15", "", []string{"Go"}, nil, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	store.data[42] = prof
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	err = h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile"})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(rep.texts[0], "Company: Acme") {
		t.Errorf("reply should embed profile Render(), got %q", rep.texts[0])
	}
	if !strings.Contains(rep.texts[0], "/profile edit") {
		t.Errorf("reply should hint edit/clear, got %q", rep.texts[0])
	}
}

func TestProfileHandler_Edit_StartsWizardAtStepName(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile edit"})
	if err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	state, ok := sessions.Get(42)
	if !ok {
		t.Fatal("expected wizard session created for chat 42")
	}
	if state.ChatID != 42 {
		t.Errorf("state.ChatID = %d, want 42", state.ChatID)
	}
	if state.Step != telegram.StepName {
		t.Errorf("state.Step = %q, want %q", state.Step, telegram.StepName)
	}
	if state.Draft == nil {
		t.Error("state.Draft should be initialised (non-nil)")
	}
	if state.StartedAt.IsZero() {
		t.Error("state.StartedAt should be set")
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	// First wizard prompt must mention the company name question in Russian.
	if !strings.Contains(rep.texts[0], "компани") {
		t.Errorf("wizard start reply should mention 'компани', got %q", rep.texts[0])
	}
}

func TestProfileHandler_Edit_OverwritesExistingSession(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	// Pre-existing session at StepExtra — /profile edit must reset to StepName.
	sessions.Set(42, &telegram.WizardState{ChatID: 42, Step: telegram.StepExtra, Draft: &telegram.ProfileDraft{Name: "Old"}})
	h := telegram.NewProfileHandler(store, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile edit"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	state, _ := sessions.Get(42)
	if state.Step != telegram.StepName {
		t.Errorf("Step = %q after edit, want %q", state.Step, telegram.StepName)
	}
	if state.Draft == nil || state.Draft.Name != "" {
		t.Errorf("Draft should be fresh, got %+v", state.Draft)
	}
}

func TestProfileHandler_Clear_DeletesProfileAndReplies(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	prof, _ := domain.NewCompanyProfile("Acme", "", "", nil, nil, nil, "", "")
	store.data[42] = prof
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile clear"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if store.clrCalls != 1 {
		t.Errorf("store.Clear calls = %d, want 1", store.clrCalls)
	}
	if _, ok := store.data[42]; ok {
		t.Error("profile should be removed from store")
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(rep.texts[0], "удал") {
		t.Errorf("reply should confirm deletion, got %q", rep.texts[0])
	}
}

func TestProfileHandler_StoreGetError_PropagatesAsReply(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	store.getErr = errors.New("disk read failed")
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile"})
	if err != nil {
		// Handlers in this codebase report user-facing failures via Reply and
		// surface only transport-level errors via return — assert that pattern.
		t.Fatalf("HandleCommand should not return store-read errors directly: %v", err)
	}
	if len(rep.texts) == 0 || !strings.Contains(rep.texts[0], "❌") {
		t.Errorf("expected error reply (❌ marker), got %v", rep.texts)
	}
}

// --- HandleWizardInput tests --------------------------------------------

// runFullWizard pushes the eight wizard answers through HandleWizardInput in
// order and returns the final reply texts collected by rep. Helper for the
// happy-path full-flow test plus its dash-sentinel variant.
func runFullWizard(t *testing.T, h *telegram.ProfileHandler, chatID int64, answers []string) {
	t.Helper()
	// Start the wizard so the session exists at StepName.
	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: chatID, Text: "/profile edit"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	for i, ans := range answers {
		if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: chatID, Text: ans}); err != nil {
			t.Fatalf("HandleWizardInput #%d (%q): %v", i, ans, err)
		}
	}
}

func TestProfileHandler_WizardInput_NoSession_NoOp(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: 42, Text: "Acme"})
	if err != nil {
		t.Errorf("HandleWizardInput on absent session should be no-op, got err=%v", err)
	}
	if len(rep.texts) != 0 {
		t.Errorf("expected no reply on no-session input, got %v", rep.texts)
	}
}

func TestProfileHandler_WizardInput_Cancel_ClearsSessionAndReplies(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	sessions.Set(42, &telegram.WizardState{ChatID: 42, Step: telegram.StepName, Draft: &telegram.ProfileDraft{Name: "Acme"}})
	h := telegram.NewProfileHandler(store, sessions, rep)

	if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: 42, Text: "/cancel"}); err != nil {
		t.Fatalf("HandleWizardInput: %v", err)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after /cancel")
	}
	if len(rep.texts) != 1 || !strings.Contains(strings.ToLower(rep.texts[0]), "отмен") {
		t.Errorf("expected cancellation reply, got %v", rep.texts)
	}
	if store.setCalls != 0 {
		t.Error("Cancel must not persist anything")
	}
}

func TestProfileHandler_WizardInput_StepNameAdvancesToTeamSize(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)
	// Start wizard.
	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile edit"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	rep.texts = nil // ignore start reply

	if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: 42, Text: "Acme Corp"}); err != nil {
		t.Fatalf("HandleWizardInput: %v", err)
	}
	state, ok := sessions.Get(42)
	if !ok {
		t.Fatal("session should still exist mid-wizard")
	}
	if state.Step != telegram.StepTeamSize {
		t.Errorf("Step = %q, want %q", state.Step, telegram.StepTeamSize)
	}
	if state.Draft.Name != "Acme Corp" {
		t.Errorf("Draft.Name = %q, want %q", state.Draft.Name, "Acme Corp")
	}
	if len(rep.texts) != 1 || !strings.Contains(rep.texts[0], "команд") {
		t.Errorf("expected team-size question, got %v", rep.texts)
	}
}

func TestProfileHandler_WizardInput_FullFlow_PersistsProfile(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	runFullWizard(t, h, 42, []string{
		"Acme Corp",           // name
		"15",                  // team size
		"7",                   // experience
		"Go, React, Postgres", // tech stack
		"ISO 9001, SOC2",      // certs
		"backend, mobile",     // specs
		"Sberbank, Yandex",    // key clients
		"Remote-first",        // extra
	})

	if store.setCalls != 1 {
		t.Fatalf("store.Set calls = %d, want 1", store.setCalls)
	}
	saved, ok := store.data[42]
	if !ok {
		t.Fatal("profile should be saved for chat 42")
	}
	render := saved.Render()
	wants := []string{
		"Company: Acme Corp",
		"Team: 15 people",
		"Experience: 7 years",
		"Tech stack: Go, React, Postgres",
		"Certifications: ISO 9001, SOC2",
		"Specializations: backend, mobile",
		"Key clients/projects: Sberbank, Yandex",
		"Additional: Remote-first",
	}
	for _, w := range wants {
		if !strings.Contains(render, w) {
			t.Errorf("saved profile missing %q\nrender: %s", w, render)
		}
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after final step")
	}
	// Last reply must echo the saved profile preview.
	if len(rep.texts) == 0 {
		t.Fatal("expected confirmation reply at end of wizard")
	}
	last := rep.texts[len(rep.texts)-1]
	if !strings.Contains(last, "Company: Acme Corp") {
		t.Errorf("confirmation reply should embed Render(), got %q", last)
	}
}

func TestProfileHandler_WizardInput_DashSentinelTreatedAsEmpty(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	runFullWizard(t, h, 42, []string{
		"Acme Corp",
		"15",
		"",   // experience left blank
		"Go", // stack
		"-",  // certs skipped
		"backend",
		"-", // clients skipped
		"-", // extra skipped
	})

	saved := store.data[42]
	if saved == nil {
		t.Fatal("profile should be saved")
	}
	render := saved.Render()
	for _, forbidden := range []string{"Experience:", "Certifications:", "Key clients", "Additional:"} {
		if strings.Contains(render, forbidden) {
			t.Errorf("Render should omit %q for blank/dash answers, got %q", forbidden, render)
		}
	}
	for _, want := range []string{"Company: Acme Corp", "Team: 15 people", "Tech stack: Go", "Specializations: backend"} {
		if !strings.Contains(render, want) {
			t.Errorf("Render missing %q\nrender: %s", want, render)
		}
	}
}

func TestProfileHandler_WizardInput_AllDash_ReplyEmptyProfileHint(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	// All answers are dashes / empty — NewCompanyProfile must reject.
	runFullWizard(t, h, 42, []string{
		"-", "-", "-", "-", "-", "-", "-", "-",
	})

	if store.setCalls != 0 {
		t.Errorf("store.Set must not be called when profile is empty (got %d)", store.setCalls)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after empty-profile rejection")
	}
	last := rep.texts[len(rep.texts)-1]
	if !strings.Contains(strings.ToLower(last), "пуст") {
		t.Errorf("expected empty-profile hint, got %q", last)
	}
}

func TestProfileHandler_WizardInput_StoreSetError_RepliesError(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	store.setErr = errors.New("disk full")
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	runFullWizard(t, h, 42, []string{
		"Acme", "15", "7", "Go", "-", "backend", "-", "-",
	})

	last := rep.texts[len(rep.texts)-1]
	if !strings.Contains(last, "❌") {
		t.Errorf("expected save-error reply (❌ marker), got %q", last)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared even on save error to avoid stuck state")
	}
}

func TestProfileHandler_UnknownSubcommand_RepliesUsageHint(t *testing.T) {
	rep := &fakeReplier{}
	store := newFakeProfileStore()
	sessions := telegram.NewInMemoryWizardSessions()
	h := telegram.NewProfileHandler(store, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/profile garbage"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	// User should see the available subcommands so they self-correct.
	if !strings.Contains(rep.texts[0], "/profile") {
		t.Errorf("usage hint should mention /profile, got %q", rep.texts[0])
	}
}
