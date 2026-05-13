package telegram

import (
	"context"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// ProfileStore persists per-chat company profiles. Implementations live in
// adapter/profilestore; the interface stays here so handlers depend on the
// use-case-owned contract (DIP).
type ProfileStore interface {
	// Get returns the profile for chatID. The boolean is false when no
	// profile is registered for that chat; in that case the returned
	// *domain.CompanyProfile is nil and err is nil.
	Get(ctx context.Context, chatID int64) (*domain.CompanyProfile, bool, error)
	// Set writes the profile, overwriting any existing entry.
	Set(ctx context.Context, chatID int64, p *domain.CompanyProfile) error
	// Clear removes the profile. Absent chats are a no-op.
	Clear(ctx context.Context, chatID int64) error
}
