package botconfig

import (
	"context"
	"fmt"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// Service is the use-case façade over the Repository. Handlers depend on
// *Service, not on Repository directly, so domain validation always runs
// before persistence.
type Service struct {
	repo Repository
}

// NewService constructs a Service backed by the given repository. The repo
// must be non-nil; the constructor does not defensively check this — DI is
// expected to wire a valid implementation.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Get returns the currently persisted configuration. The bool is false
// when nothing has ever been saved; in that case cfg is nil and err is nil.
// Any I/O failure surfaces as a wrapped error.
func (s *Service) Get(ctx context.Context) (*domain.BotConfig, bool, error) {
	cfg, ok, err := s.repo.Load(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("botconfig: load: %w", err)
	}
	return cfg, ok, nil
}

// Update validates the input via the domain constructor and, on success,
// persists the resulting config. The returned *BotConfig is the same
// instance that was saved. Validation errors surface from the domain
// constructor unchanged (errors.Is works against domain.ErrInvalidBotToken,
// domain.ErrInvalidLogLevel, auth.ErrInvalidUserID).
func (s *Service) Update(ctx context.Context, token string, allowlistUserIDs []int64, logLevel string) (*domain.BotConfig, error) {
	cfg, err := domain.NewBotConfig(token, allowlistUserIDs, logLevel)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, cfg); err != nil {
		return nil, fmt.Errorf("botconfig: save: %w", err)
	}
	return cfg, nil
}
