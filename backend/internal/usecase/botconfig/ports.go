// Package botconfig is the use-case layer for the Telegram bot's runtime
// configuration: load, validate, persist. The Repository port lives here
// (next to its consumer, the Service) per the Dependency Inversion
// principle — adapters/* implementations import this package, not the
// other way around.
package botconfig

import (
	"context"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// Repository abstracts persistent storage of *domain.BotConfig. Implemented
// by adapter/botconfigstore (JSON file, in production) and stub repos in
// tests. The interface is intentionally narrow: a single config record,
// no listing or history.
type Repository interface {
	// Load returns the persisted config. The bool reports presence —
	// false with a nil config and nil error means "no config has ever
	// been saved" (fresh deployment). Any I/O failure surfaces via err.
	Load(ctx context.Context) (*domain.BotConfig, bool, error)

	// Save atomically replaces the persisted config. Implementations
	// must guarantee the file on disk is either the previous version
	// or the new one — never a half-written state.
	Save(ctx context.Context, cfg *domain.BotConfig) error
}
