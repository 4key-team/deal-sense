package botconfig_test

import (
	"context"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase/botconfig"
)

const fixtureValidToken = "8829614348:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"

// stubRepo lets us control the Repository's behaviour and observe what the
// service persisted.
type stubRepo struct {
	loaded    *domain.BotConfig
	hasLoaded bool
	loadErr   error

	saved   *domain.BotConfig
	saveErr error
}

func (s *stubRepo) Load(context.Context) (*domain.BotConfig, bool, error) {
	return s.loaded, s.hasLoaded, s.loadErr
}

func (s *stubRepo) Save(_ context.Context, cfg *domain.BotConfig) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = cfg
	return nil
}

// --- Get -------------------------------------------------------------------

func TestService_Get_NoConfig_ReturnsNilAndFalse(t *testing.T) {
	svc := botconfig.NewService(&stubRepo{hasLoaded: false})
	cfg, ok, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Error("Get must report ok=false when repository has no config")
	}
	if cfg != nil {
		t.Error("Get must return nil config when repository has no config")
	}
}

func TestService_Get_ExistingConfig_ReturnsIt(t *testing.T) {
	stored, err := domain.NewBotConfig(fixtureValidToken, []int64{42}, "info")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	svc := botconfig.NewService(&stubRepo{loaded: stored, hasLoaded: true})

	cfg, ok, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Error("Get must report ok=true when repository has a config")
	}
	if cfg != stored {
		t.Errorf("Get must return stored config; got %v want %v", cfg, stored)
	}
}

func TestService_Get_RepositoryError_Propagates(t *testing.T) {
	sentinel := errors.New("disk on fire")
	svc := botconfig.NewService(&stubRepo{loadErr: sentinel})

	_, _, err := svc.Get(context.Background())
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want wrapping %v", err, sentinel)
	}
}

// --- Update ---------------------------------------------------------------

func TestService_Update_InvalidToken_PropagatesDomainError(t *testing.T) {
	repo := &stubRepo{}
	svc := botconfig.NewService(repo)

	_, err := svc.Update(context.Background(), "not-a-token", []int64{42}, "info")
	if !errors.Is(err, domain.ErrInvalidBotToken) {
		t.Errorf("err = %v, want wrapping ErrInvalidBotToken", err)
	}
	if repo.saved != nil {
		t.Error("Update must NOT persist when domain validation fails")
	}
}

func TestService_Update_InvalidLogLevel_PropagatesDomainError(t *testing.T) {
	repo := &stubRepo{}
	svc := botconfig.NewService(repo)

	_, err := svc.Update(context.Background(), fixtureValidToken, []int64{42}, "verbose")
	if !errors.Is(err, domain.ErrInvalidLogLevel) {
		t.Errorf("err = %v, want wrapping ErrInvalidLogLevel", err)
	}
	if repo.saved != nil {
		t.Error("Update must NOT persist when log level invalid")
	}
}

func TestService_Update_ValidInput_PersistsAndReturns(t *testing.T) {
	repo := &stubRepo{}
	svc := botconfig.NewService(repo)

	cfg, err := svc.Update(context.Background(), fixtureValidToken, []int64{42, 100}, "warn")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg == nil {
		t.Fatal("Update returned nil config without error")
	}
	if cfg.Token() != fixtureValidToken {
		t.Errorf("returned token = %q, want %q", cfg.Token(), fixtureValidToken)
	}
	if cfg.LogLevel() != domain.LogLevelWarn {
		t.Errorf("returned log level = %q, want warn", cfg.LogLevel())
	}
	if repo.saved != cfg {
		t.Error("Update must persist the exact config it returned")
	}
}

func TestService_Update_EmptyAllowlist_AcceptedAsOpen(t *testing.T) {
	// Domain-level relaxation: empty allowlist is valid (open mode).
	// Update must accept it and persist.
	repo := &stubRepo{}
	svc := botconfig.NewService(repo)

	cfg, err := svc.Update(context.Background(), fixtureValidToken, nil, "info")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !cfg.Allowlist().IsOpen() {
		t.Error("empty allowlist input must yield open mode")
	}
	if repo.saved == nil {
		t.Error("Update must persist open-mode config")
	}
}

func TestService_Update_RepositorySaveError_Propagates(t *testing.T) {
	sentinel := errors.New("disk full")
	repo := &stubRepo{saveErr: sentinel}
	svc := botconfig.NewService(repo)

	_, err := svc.Update(context.Background(), fixtureValidToken, []int64{42}, "info")
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want wrapping %v", err, sentinel)
	}
}
