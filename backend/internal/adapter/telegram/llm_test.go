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

// fakeLLMService is an in-memory LLMSettingsService for handler tests. It
// runs the same domain validation as the real *llmsettings.Service so the
// handler sees identical errors and behaviour.
type fakeLLMService struct {
	mu       sync.Mutex
	data     map[int64]*domain.LLMSettings
	getErr   error
	setErr   error
	clearErr error
	setCalls int
	clrCalls int
}

func newFakeLLMService() *fakeLLMService {
	return &fakeLLMService{data: map[int64]*domain.LLMSettings{}}
}

func (f *fakeLLMService) Get(_ context.Context, chatID int64) (*domain.LLMSettings, bool, error) {
	if f.getErr != nil {
		return nil, false, f.getErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg, ok := f.data[chatID]
	return cfg, ok, nil
}

func (f *fakeLLMService) Update(_ context.Context, chatID int64, provider, baseURL, apiKey, model string) (*domain.LLMSettings, error) {
	cfg, err := domain.NewLLMSettings(provider, baseURL, apiKey, model)
	if err != nil {
		return nil, err
	}
	if f.setErr != nil {
		return nil, f.setErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[chatID] = cfg
	f.setCalls++
	return cfg, nil
}

func (f *fakeLLMService) Clear(_ context.Context, chatID int64) error {
	if f.clearErr != nil {
		return f.clearErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, chatID)
	f.clrCalls++
	return nil
}

// --- /llm command --------------------------------------------------------

func TestLLMHandler_NoSettings_ShowsEmptyHint(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(rep.texts[0], "/llm edit") {
		t.Errorf("empty-hint reply should mention /llm edit, got %q", rep.texts[0])
	}
}

func TestLLMHandler_ExistingSettings_ShowsMaskedRender(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	cfg, err := domain.NewLLMSettings("openai", "https://openrouter.ai/api/v1", "sk-or-supersecret1234abcd", "anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	svc.data[42] = cfg
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	text := rep.texts[0]
	if !strings.Contains(text, "openai") {
		t.Errorf("reply missing provider, got %q", text)
	}
	if !strings.Contains(text, "openrouter.ai") {
		t.Errorf("reply missing base URL, got %q", text)
	}
	if !strings.Contains(text, "anthropic/claude-sonnet-4") {
		t.Errorf("reply missing model, got %q", text)
	}
	// Secret body MUST NOT leak — only the last-4 fingerprint is allowed.
	if strings.Contains(text, "supersecret1234") {
		t.Errorf("reply leaked secret body, got %q", text)
	}
	if !strings.Contains(text, "abcd") {
		t.Errorf("reply missing last-4 fingerprint, got %q", text)
	}
}

func TestLLMHandler_ExistingSettings_NoBaseURL_ShowsDefaultMarker(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	cfg, err := domain.NewLLMSettings("openai", "", "sk-test1234", "gpt-4o")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	svc.data[42] = cfg
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if !strings.Contains(rep.texts[0], "по умолчанию") {
		t.Errorf("empty base URL must surface as 'по умолчанию' marker, got %q", rep.texts[0])
	}
}

func TestLLMHandler_Edit_StartsWizardAtStepProvider(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm edit"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	state, ok := sessions.Get(42)
	if !ok {
		t.Fatal("expected wizard session created for chat 42")
	}
	if state.Step != telegram.StepLLMProvider {
		t.Errorf("Step = %q, want %q", state.Step, telegram.StepLLMProvider)
	}
	if state.Draft == nil {
		t.Error("Draft must be initialised")
	}
	if state.StartedAt.IsZero() {
		t.Error("StartedAt must be set so the sweeper can age the session")
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(strings.ToLower(rep.texts[0]), "provider") {
		t.Errorf("first wizard prompt should ask for provider, got %q", rep.texts[0])
	}
}

func TestLLMHandler_Edit_OverwritesExistingSession(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	sessions.Set(42, &telegram.LLMWizardState{
		ChatID: 42, Step: telegram.StepLLMModel,
		Draft: &telegram.LLMSettingsDraft{Provider: "old"},
	})
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm edit"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	state, _ := sessions.Get(42)
	if state.Step != telegram.StepLLMProvider {
		t.Errorf("Step after edit = %q, want %q (reset)", state.Step, telegram.StepLLMProvider)
	}
	if state.Draft == nil || state.Draft.Provider != "" {
		t.Errorf("Draft should be fresh, got %+v", state.Draft)
	}
}

func TestLLMHandler_Clear_DeletesSettingsAndReplies(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	cfg, _ := domain.NewLLMSettings("openai", "", "sk-test1234", "gpt-4o")
	svc.data[42] = cfg
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm clear"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if svc.clrCalls != 1 {
		t.Errorf("svc.Clear calls = %d, want 1", svc.clrCalls)
	}
	if _, ok := svc.data[42]; ok {
		t.Error("settings should be removed")
	}
	if !strings.Contains(rep.texts[0], "сброш") {
		t.Errorf("reply should confirm reset, got %q", rep.texts[0])
	}
}

func TestLLMHandler_StoreGetError_PropagatesAsReply(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	svc.getErr = errors.New("disk gone")
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm"})
	if err != nil {
		t.Fatalf("HandleCommand should not surface store errors directly: %v", err)
	}
	if len(rep.texts) == 0 || !strings.Contains(rep.texts[0], "❌") {
		t.Errorf("expected error reply (❌ marker), got %v", rep.texts)
	}
}

func TestLLMHandler_UnknownSubcommand_RepliesUsageHint(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm garbage"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
	if len(rep.texts) != 1 {
		t.Fatalf("Reply called %d times, want 1", len(rep.texts))
	}
	if !strings.Contains(rep.texts[0], "/llm") {
		t.Errorf("usage hint must mention /llm, got %q", rep.texts[0])
	}
}

// --- /llm wizard ---------------------------------------------------------

// runFullLLMWizard pushes the four wizard answers through HandleWizardInput.
func runFullLLMWizard(t *testing.T, h *telegram.LLMHandler, chatID int64, answers []string) {
	t.Helper()
	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: chatID, Text: "/llm edit"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	for i, ans := range answers {
		if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: chatID, Text: ans}); err != nil {
			t.Fatalf("HandleWizardInput #%d (%q): %v", i, ans, err)
		}
	}
}

func TestLLMHandler_WizardInput_NoSession_NoOp(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: 42, Text: "openai"}); err != nil {
		t.Errorf("HandleWizardInput on absent session should be no-op, got err=%v", err)
	}
	if len(rep.texts) != 0 {
		t.Errorf("expected no reply on no-session input, got %v", rep.texts)
	}
}

func TestLLMHandler_WizardInput_Cancel_ClearsSessionAndReplies(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	sessions.Set(42, &telegram.LLMWizardState{
		ChatID: 42, Step: telegram.StepLLMProvider,
		Draft: &telegram.LLMSettingsDraft{},
	})
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: 42, Text: "/cancel"}); err != nil {
		t.Fatalf("HandleWizardInput: %v", err)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after /cancel")
	}
	if len(rep.texts) != 1 || !strings.Contains(strings.ToLower(rep.texts[0]), "отмен") {
		t.Errorf("expected cancellation reply, got %v", rep.texts)
	}
	if svc.setCalls != 0 {
		t.Error("Cancel must not persist anything")
	}
}

func TestLLMHandler_WizardInput_StepProvider_AdvancesToBaseURL(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm edit"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	rep.texts = nil // ignore start reply

	if err := h.HandleWizardInput(context.Background(), &telegram.Update{ChatID: 42, Text: "openai"}); err != nil {
		t.Fatalf("HandleWizardInput: %v", err)
	}
	state, ok := sessions.Get(42)
	if !ok {
		t.Fatal("session should still exist mid-wizard")
	}
	if state.Step != telegram.StepLLMBaseURL {
		t.Errorf("Step = %q, want %q", state.Step, telegram.StepLLMBaseURL)
	}
	if state.Draft.Provider != "openai" {
		t.Errorf("Draft.Provider = %q, want openai", state.Draft.Provider)
	}
	if !strings.Contains(strings.ToLower(rep.texts[0]), "url") {
		t.Errorf("expected base-URL question, got %v", rep.texts)
	}
}

func TestLLMHandler_WizardInput_FullFlow_PersistsSettings(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	runFullLLMWizard(t, h, 42, []string{
		"openai",                       // provider
		"https://openrouter.ai/api/v1", // base URL
		"sk-or-test1234secretABCD",     // api key
		"anthropic/claude-sonnet-4",    // model
	})

	if svc.setCalls != 1 {
		t.Fatalf("svc.Update calls = %d, want 1", svc.setCalls)
	}
	saved, ok := svc.data[42]
	if !ok {
		t.Fatal("settings should be saved for chat 42")
	}
	if saved.Provider() != "openai" || saved.BaseURL() != "https://openrouter.ai/api/v1" ||
		saved.APIKey() != "sk-or-test1234secretABCD" || saved.Model() != "anthropic/claude-sonnet-4" {
		t.Errorf("saved settings mismatch: %+v", saved)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after final step")
	}
	if len(rep.texts) == 0 {
		t.Fatal("expected confirmation reply")
	}
	last := rep.texts[len(rep.texts)-1]
	if !strings.Contains(last, "openai") {
		t.Errorf("confirmation reply should embed provider, got %q", last)
	}
	if strings.Contains(last, "test1234secretA") {
		t.Errorf("confirmation reply must not leak secret body, got %q", last)
	}
}

func TestLLMHandler_WizardInput_DashOnBaseURL_TreatedAsEmpty(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	runFullLLMWizard(t, h, 42, []string{
		"openai",
		"-", // skip base URL → server default
		"sk-test1234",
		"gpt-4o",
	})

	saved := svc.data[42]
	if saved == nil {
		t.Fatal("settings should be saved")
	}
	if saved.BaseURL() != "" {
		t.Errorf("BaseURL after dash = %q, want empty (provider default)", saved.BaseURL())
	}
}

func TestLLMHandler_WizardInput_InvalidInput_RepliesValidationError(t *testing.T) {
	// Empty provider — finalisation must reject via domain.ErrEmptyLLMProvider.
	// The wizard collects an empty provider answer ("-"), then the rest, then
	// finalize tries to construct domain.LLMSettings and fails. User sees a
	// validation reply, session is cleared so they restart fresh.
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	runFullLLMWizard(t, h, 42, []string{
		"-",          // provider empty — should fail at finalize
		"-",          // base URL skip
		"sk-test123", // valid api key
		"gpt-4o",     // valid model
	})

	if svc.setCalls != 0 {
		t.Errorf("svc.Update must not persist invalid settings; got %d calls", svc.setCalls)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared after validation rejection")
	}
	last := rep.texts[len(rep.texts)-1]
	if !strings.Contains(last, "❌") {
		t.Errorf("expected validation error reply (❌ marker), got %q", last)
	}
}

func TestLLMHandler_WizardInput_ServiceSetError_RepliesError(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	svc.setErr = errors.New("disk full")
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	runFullLLMWizard(t, h, 42, []string{
		"openai", "-", "sk-test1234", "gpt-4o",
	})

	last := rep.texts[len(rep.texts)-1]
	if !strings.Contains(last, "❌") {
		t.Errorf("expected save-error reply, got %q", last)
	}
	if _, ok := sessions.Get(42); ok {
		t.Error("session should be cleared even on save error to avoid stuck state")
	}
}

func TestLLMHandler_NilLogger_DoesNotPanic(t *testing.T) {
	rep := &fakeReplier{}
	svc := newFakeLLMService()
	sessions := telegram.NewInMemoryLLMWizardSessions()
	h := telegram.NewLLMHandler(svc, sessions, rep)

	if err := h.HandleCommand(context.Background(), &telegram.Update{ChatID: 42, Text: "/llm edit"}); err != nil {
		t.Fatalf("HandleCommand: %v", err)
	}
}
