// Package auth defines authentication invariants for entry-points such as
// the Telegram bot. The Allowlist VO is the single point of truth for
// "who may interact with this bot" and lives in domain because deciding
// who is allowed is a business rule, not an infrastructure concern.
package auth

import (
	"errors"
	"fmt"
)

// ErrEmptyAllowlist indicates an attempt to construct a restricted Allowlist
// with no user IDs. For dev/bootstrap, NewOpenAllowlist returns an explicitly
// open allowlist instead.
var ErrEmptyAllowlist = errors.New("auth: restricted allowlist must contain at least one user ID")

// ErrInvalidUserID indicates a non-positive ID was supplied. Telegram user
// IDs are always positive int64.
var ErrInvalidUserID = errors.New("auth: user ID must be positive")

// Allowlist is the immutable set of Telegram user IDs permitted to interact
// with the bot. An Allowlist is either open (anyone allowed) or restricted
// (specific IDs only). Construct via NewOpenAllowlist, NewRestrictedAllowlist
// or ParseAllowlist — the zero value is invalid.
type Allowlist struct {
	open    bool
	members map[int64]struct{}
}

// NewOpenAllowlist returns an allowlist that admits any user ID. Intended for
// dev/bootstrap; production deployments should use NewRestrictedAllowlist.
func NewOpenAllowlist() *Allowlist {
	return &Allowlist{open: true}
}

// NewRestrictedAllowlist validates the provided IDs and returns an Allowlist
// admitting only those members. Returns ErrEmptyAllowlist if ids is nil/empty,
// ErrInvalidUserID wrapped with the offending value if any ID is ≤ 0.
// Duplicates are silently collapsed.
func NewRestrictedAllowlist(ids []int64) (*Allowlist, error) {
	if len(ids) == 0 {
		return nil, ErrEmptyAllowlist
	}
	members := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("%w: %d", ErrInvalidUserID, id)
		}
		members[id] = struct{}{}
	}
	return &Allowlist{members: members}, nil
}

// ParseAllowlist is a smart factory for config-loading contexts: empty input
// produces an open allowlist, non-empty input is validated as restricted.
// Use this when the caller cannot statically distinguish "bootstrap" from
// "production" — e.g. when reading env vars or JSON config files.
func ParseAllowlist(ids []int64) (*Allowlist, error) {
	if len(ids) == 0 {
		return NewOpenAllowlist(), nil
	}
	return NewRestrictedAllowlist(ids)
}

// IsOpen reports whether the allowlist admits any user (no membership check).
func (a *Allowlist) IsOpen() bool {
	return a.open
}

// IsAllowed reports whether the given user ID may interact with the bot.
// Open allowlists return true for any ID.
func (a *Allowlist) IsAllowed(id int64) bool {
	if a.open {
		return true
	}
	_, ok := a.members[id]
	return ok
}

// Members returns the restricted set of allowed IDs in unspecified order.
// Open allowlists return nil — the concept of "members" doesn't apply when
// everyone is admitted. Use IsOpen to disambiguate before relying on this.
func (a *Allowlist) Members() []int64 {
	if a.open {
		return nil
	}
	out := make([]int64, 0, len(a.members))
	for id := range a.members {
		out = append(out, id)
	}
	return out
}
