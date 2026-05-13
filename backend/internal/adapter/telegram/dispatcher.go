package telegram

import (
	"context"
	"strings"
)

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
