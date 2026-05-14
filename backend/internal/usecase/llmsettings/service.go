package llmsettings

import (
	"context"
	"errors"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// Service is the use-case façade over the Repository. Handlers depend on
// *Service, not on Repository directly, so domain validation always runs
// before persistence.
type Service struct {
	repo Repository
}

// NewService constructs a Service backed by the given repository. The repo
// must be non-nil; the constructor does not defensively check this — DI
// is expected to wire a valid implementation.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// errNotImplemented keeps the RED-phase stubs compiling without leaking
// fake-success behaviour into call sites.
var errNotImplemented = errors.New("llmsettings: not implemented")

// Get returns the persisted settings for chatID. (nil, false, nil) when
// no settings are registered. Any I/O failure surfaces as a wrapped error.
func (s *Service) Get(ctx context.Context, chatID int64) (*domain.LLMSettings, bool, error) {
	return nil, false, errNotImplemented
}

// Update validates the input via the domain constructor and persists the
// resulting settings on success. The returned *LLMSettings is the same
// instance that was saved. Validation errors surface from the domain
// constructor unchanged (errors.Is matches domain.ErrEmptyLLM*).
func (s *Service) Update(ctx context.Context, chatID int64, provider, baseURL, apiKey, model string) (*domain.LLMSettings, error) {
	return nil, errNotImplemented
}

// Clear removes the settings for chatID. Absent chats are a no-op.
func (s *Service) Clear(ctx context.Context, chatID int64) error {
	return errNotImplemented
}
