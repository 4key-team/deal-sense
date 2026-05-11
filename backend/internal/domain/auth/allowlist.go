// Package auth defines authentication invariants for entry-points such as
// the Telegram bot. The Allowlist VO is the single point of truth for
// "who may interact with this bot" and lives in domain because deciding
// who is allowed is a business rule, not an infrastructure concern.
package auth

import "errors"

// ErrEmptyAllowlist indicates an attempt to construct an Allowlist with no
// user IDs — that would silently allow no one, which is almost always a
// misconfiguration. Empty is rejected loudly.
var ErrEmptyAllowlist = errors.New("auth: allowlist must contain at least one user ID")

// ErrInvalidUserID indicates a non-positive ID was supplied. Telegram user
// IDs are always positive int64.
var ErrInvalidUserID = errors.New("auth: user ID must be positive")

// Allowlist is the immutable set of Telegram user IDs permitted to interact
// with the bot. Construct via NewAllowlist — the zero value is invalid.
type Allowlist struct{}

// NewAllowlist validates and returns an Allowlist. Stub for RED step.
func NewAllowlist(ids []int64) (*Allowlist, error) {
	return nil, nil
}

// IsAllowed reports whether the given user ID is in the allowlist.
func (a *Allowlist) IsAllowed(id int64) bool {
	return false
}
