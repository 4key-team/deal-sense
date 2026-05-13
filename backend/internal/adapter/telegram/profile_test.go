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
	mu        sync.Mutex
	data      map[int64]*domain.CompanyProfile
	getErr    error
	setErr    error
	clearErr  error
	setCalls  int
	clrCalls  int
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
	if len(rep.texts) == 0 || !strings.Contains(strings.ToLower(rep.texts[0]), "ошибка") {
		t.Errorf("expected error reply, got %v", rep.texts)
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
