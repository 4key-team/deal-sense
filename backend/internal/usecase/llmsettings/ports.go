// Package llmsettings is the use-case layer for per-chat LLM provider
// configuration in the Telegram bot. The Repository port lives here, next
// to its only consumer (the Service), per the Dependency Inversion
// principle — adapter/llmsettingsstore imports this package, not the other
// way around.
package llmsettings

import (
	"context"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// Repository abstracts per-chat persistent storage of *domain.LLMSettings.
// Implemented by adapter/llmsettingsstore (JSON file) and stub repos in
// tests. Each chat owns at most one record.
type Repository interface {
	// Get returns the settings for chatID. The bool reports presence —
	// false with a nil cfg and nil err means "no settings registered for
	// this chat" (fresh chat, or after Clear). Any I/O failure surfaces
	// via err.
	Get(ctx context.Context, chatID int64) (*domain.LLMSettings, bool, error)

	// Set atomically writes the settings, overwriting any existing entry
	// for chatID. Implementations must guarantee the on-disk image is
	// either the previous version or the new one — never a half-write.
	Set(ctx context.Context, chatID int64, cfg *domain.LLMSettings) error

	// Clear removes the entry for chatID. Absent chats are a no-op
	// (no error). Any I/O failure surfaces via err.
	Clear(ctx context.Context, chatID int64) error
}
