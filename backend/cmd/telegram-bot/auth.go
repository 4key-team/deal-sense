package main

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

// declineCounter is the narrow port the allowlist middleware uses to record
// security declines (allowlist kind). nil is tolerated and treated as a
// no-op so the bot keeps working with no metrics configured.
type declineCounter interface {
	Inc(kind string)
}

// declineKindAllowlist is the canonical kind label emitted when the
// allowlist middleware blocks an update. Defined here (next to its only
// caller) to avoid an adapter→cmd back-reference into metrics.
const declineKindAllowlist = "allowlist"

// extractUserID pulls the originating user ID out of any update kind we
// care about. Returns 0 if the update has no user attached — those
// updates fall through unchanged.
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

// allowlistMiddleware blocks updates from users outside the allowlist with
// a polite refusal posted via send. Updates without a user (system events)
// pass through.
func allowlistMiddleware(list *auth.Allowlist, send func(ctx context.Context, chatID int64, text string), counter declineCounter) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, u *models.Update) {
			uid := extractUserID(u)
			if uid == 0 {
				next(ctx, b, u)
				return
			}
			if !list.IsAllowed(uid) {
				if counter != nil {
					counter.Inc(declineKindAllowlist)
				}
				if chatID := extractChatID(u); chatID != 0 {
					send(ctx, chatID, telegram.MsgDenied)
				}
				return
			}
			next(ctx, b, u)
		}
	}
}
