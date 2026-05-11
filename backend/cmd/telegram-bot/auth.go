package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

// extractUserID pulls the originating user ID out of any update kind we
// care about. Returns 0 if the update has no user attached.
func extractUserID(u *models.Update) int64 {
	switch {
	case u == nil:
		return 0
	case u.Message != nil && u.Message.From != nil:
		return u.Message.From.ID
	case u.CallbackQuery != nil:
		return u.CallbackQuery.From.ID
	}
	return 0
}

// extractChatID returns the chat ID for messaging back, or 0 if unknown.
func extractChatID(u *models.Update) int64 {
	switch {
	case u == nil:
		return 0
	case u.Message != nil:
		return u.Message.Chat.ID
	case u.CallbackQuery != nil && u.CallbackQuery.Message.Message != nil:
		return u.CallbackQuery.Message.Message.Chat.ID
	}
	return 0
}

// allowlistMiddleware stub — passes every update through without checking,
// so TestAllowlistMiddleware/denied_user_blocked_and_notified fails on the
// missing denial side-effect. The GREEN commit replaces it with the real
// guard.
func allowlistMiddleware(list *auth.Allowlist, send func(ctx context.Context, chatID int64, text string)) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return next
	}
}
