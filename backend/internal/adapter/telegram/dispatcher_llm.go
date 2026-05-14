package telegram

import (
	"context"
	"strings"
)

// ShouldRouteToLLM reports whether a text message belongs to the /llm
// router rather than the generic fallback. It is the predicate behind
// RouteWizardOrLLM, exposed so the runtime can register it as a
// bot.MatchFunc without duplicating the heuristic.
//
// Returns true when the message starts with "/llm" or a wizard session is
// currently active for chatID. Unlike /profile, there is no reply-keyboard
// button to alias — /llm is an advanced command users discover via /help.
func ShouldRouteToLLM(text string, chatID int64, sessions LLMWizardSessions) bool {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "/llm") {
		return true
	}
	if trimmed == "" {
		return false
	}
	_, ok := sessions.Get(chatID)
	return ok
}

// RouteWizardOrLLM routes a text message to the LLM handler when the
// message either targets /llm or is part of an in-flight /llm wizard.
// Returns handled=true iff the LLM handler consumed the message; callers
// should fall through to their own dispatchers when handled=false.
//
// /llm commands always win — typing "/llm edit" mid-wizard resets the
// wizard instead of being treated as the answer to the current question.
func RouteWizardOrLLM(ctx context.Context, u *Update, lh *LLMHandler, sessions LLMWizardSessions) (bool, error) {
	text := strings.TrimSpace(u.Text)
	if strings.HasPrefix(text, "/llm") {
		return true, lh.HandleCommand(ctx, u)
	}
	if _, ok := sessions.Get(u.ChatID); ok {
		return true, lh.HandleWizardInput(ctx, u)
	}
	return false, nil
}
