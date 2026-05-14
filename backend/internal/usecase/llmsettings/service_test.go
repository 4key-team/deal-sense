package llmsettings_test

import (
	"context"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase/llmsettings"
)

// stubRepo lets us control the Repository's behaviour and observe what the
// service persisted.
type stubRepo struct {
	got    *domain.LLMSettings
	hasGot bool
	getErr error

	saved   *domain.LLMSettings
	savedID int64
	saveErr error

	clearedID  int64
	clearCalls int
	clearErr   error
}

func (s *stubRepo) Get(_ context.Context, _ int64) (*domain.LLMSettings, bool, error) {
	return s.got, s.hasGot, s.getErr
}

func (s *stubRepo) Set(_ context.Context, chatID int64, cfg *domain.LLMSettings) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = cfg
	s.savedID = chatID
	return nil
}

func (s *stubRepo) Clear(_ context.Context, chatID int64) error {
	s.clearCalls++
	s.clearedID = chatID
	return s.clearErr
}

func newCfgT(t *testing.T) *domain.LLMSettings {
	t.Helper()
	cfg, err := domain.NewLLMSettings("openai", "https://api.openai.com/v1", "sk-test1234", "gpt-4o")
	if err != nil {
		t.Fatalf("setup NewLLMSettings: %v", err)
	}
	return cfg
}

// --- Get -------------------------------------------------------------------

func TestService_Get_NoSettings_ReturnsNilAndFalse(t *testing.T) {
	svc := llmsettings.NewService(&stubRepo{hasGot: false})
	cfg, ok, err := svc.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Error("Get must report ok=false when repository has no settings")
	}
	if cfg != nil {
		t.Errorf("Get must return nil cfg when repository has no settings, got %+v", cfg)
	}
}

func TestService_Get_ExistingSettings_ReturnsThem(t *testing.T) {
	stored := newCfgT(t)
	svc := llmsettings.NewService(&stubRepo{got: stored, hasGot: true})

	cfg, ok, err := svc.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Error("Get must report ok=true when repository has settings")
	}
	if cfg != stored {
		t.Errorf("Get must return the stored settings; got %v want %v", cfg, stored)
	}
}

func TestService_Get_RepositoryError_Propagates(t *testing.T) {
	sentinel := errors.New("disk on fire")
	svc := llmsettings.NewService(&stubRepo{getErr: sentinel})

	_, _, err := svc.Get(context.Background(), 42)
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want wrapping %v", err, sentinel)
	}
}

// --- Update: validation propagation ---------------------------------------

func TestService_Update_InvalidProvider_PropagatesDomainError(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	_, err := svc.Update(context.Background(), 42, "  ", "", "sk-abc", "gpt-4o")
	if !errors.Is(err, domain.ErrEmptyLLMProvider) {
		t.Errorf("err = %v, want wrapping ErrEmptyLLMProvider", err)
	}
	if repo.saved != nil {
		t.Error("Update must NOT persist when provider invalid")
	}
}

func TestService_Update_InvalidAPIKey_PropagatesDomainError(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	_, err := svc.Update(context.Background(), 42, "openai", "", "", "gpt-4o")
	if !errors.Is(err, domain.ErrEmptyLLMAPIKey) {
		t.Errorf("err = %v, want wrapping ErrEmptyLLMAPIKey", err)
	}
	if repo.saved != nil {
		t.Error("Update must NOT persist when api_key invalid")
	}
}

func TestService_Update_InvalidModel_PropagatesDomainError(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	_, err := svc.Update(context.Background(), 42, "openai", "", "sk-abc", "")
	if !errors.Is(err, domain.ErrEmptyLLMModel) {
		t.Errorf("err = %v, want wrapping ErrEmptyLLMModel", err)
	}
	if repo.saved != nil {
		t.Error("Update must NOT persist when model invalid")
	}
}

func TestService_Update_InvalidBaseURL_PropagatesDomainError(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	_, err := svc.Update(context.Background(), 42, "openai", "not a url", "sk-abc", "gpt-4o")
	if !errors.Is(err, domain.ErrInvalidLLMBaseURL) {
		t.Errorf("err = %v, want wrapping ErrInvalidLLMBaseURL", err)
	}
	if repo.saved != nil {
		t.Error("Update must NOT persist when base_url invalid")
	}
}

// --- Update: success paths -------------------------------------------------

func TestService_Update_ValidInput_PersistsAndReturns(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	cfg, err := svc.Update(context.Background(), 7, "openai", "https://openrouter.ai/api/v1", "sk-abc1234", "anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg == nil {
		t.Fatal("Update returned nil cfg without error")
	}
	if cfg.Provider() != "openai" {
		t.Errorf("Provider = %q, want openai", cfg.Provider())
	}
	if cfg.BaseURL() != "https://openrouter.ai/api/v1" {
		t.Errorf("BaseURL = %q, want round-trip", cfg.BaseURL())
	}
	if cfg.Model() != "anthropic/claude-sonnet-4" {
		t.Errorf("Model = %q, want round-trip", cfg.Model())
	}
	if repo.saved != cfg {
		t.Error("Update must persist the exact cfg it returned")
	}
	if repo.savedID != 7 {
		t.Errorf("Saved chatID = %d, want 7", repo.savedID)
	}
}

func TestService_Update_EmptyBaseURL_AcceptedAsDefault(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	cfg, err := svc.Update(context.Background(), 7, "openai", "", "sk-abc1234", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.BaseURL() != "" {
		t.Errorf("BaseURL = %q, want empty (provider default)", cfg.BaseURL())
	}
	if repo.saved == nil {
		t.Error("Update must persist default-base-URL cfg")
	}
}

func TestService_Update_RepositorySaveError_Propagates(t *testing.T) {
	sentinel := errors.New("disk full")
	repo := &stubRepo{saveErr: sentinel}
	svc := llmsettings.NewService(repo)

	_, err := svc.Update(context.Background(), 42, "openai", "", "sk-abc1234", "gpt-4o")
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want wrapping %v", err, sentinel)
	}
}

// --- Clear -----------------------------------------------------------------

func TestService_Clear_DelegatesToRepository(t *testing.T) {
	repo := &stubRepo{}
	svc := llmsettings.NewService(repo)

	if err := svc.Clear(context.Background(), 99); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.clearCalls != 1 {
		t.Errorf("Clear delegated %d times, want 1", repo.clearCalls)
	}
	if repo.clearedID != 99 {
		t.Errorf("Clear chatID = %d, want 99", repo.clearedID)
	}
}

func TestService_Clear_RepositoryError_Propagates(t *testing.T) {
	sentinel := errors.New("disk yelled at me")
	repo := &stubRepo{clearErr: sentinel}
	svc := llmsettings.NewService(repo)

	err := svc.Clear(context.Background(), 99)
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want wrapping %v", err, sentinel)
	}
}
