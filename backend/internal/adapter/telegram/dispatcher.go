package telegram

import (
	"context"
	"strings"
)

// ShouldRouteToProfile reports whether a text message should be handled by
// the profile/wizard router rather than the generic fallback. It is the
// predicate behind RouteWizardOrProfile, exposed so the runtime can use it
// as a bot.MatchFunc without duplicating the heuristic.
//
// Returns true if either the message starts with the /profile command or a
// wizard session is currently active for chatID.
func ShouldRouteToProfile(text string, chatID int64, sessions WizardSessions) bool {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "/profile") {
		return true
	}
	if trimmed == "" {
		return false
	}
	_, ok := sessions.Get(chatID)
	return ok
}

// RouteWizardOrProfile routes a text message to the profile handler when the
// message either targets /profile or is part of an in-flight wizard. It
// returns true when the profile handler consumed the message; the caller's
// fallback flow should run only when this returns false.
//
// /profile commands always win — typing "/profile edit" mid-wizard resets the
// wizard instead of being treated as the answer to the current question.
func RouteWizardOrProfile(ctx context.Context, u *Update, ph *ProfileHandler, sessions WizardSessions) (bool, error) {
	text := strings.TrimSpace(u.Text)
	if strings.HasPrefix(text, "/profile") {
		return true, ph.HandleCommand(ctx, u)
	}
	if _, ok := sessions.Get(u.ChatID); ok {
		return true, ph.HandleWizardInput(ctx, u)
	}
	return false, nil
}
