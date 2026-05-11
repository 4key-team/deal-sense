package main

import (
	"context"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

type sentMessage struct {
	chatID int64
	text   string
}

type recordingSender struct {
	mu   sync.Mutex
	sent []sentMessage
}

func (r *recordingSender) send(ctx context.Context, chatID int64, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, sentMessage{chatID: chatID, text: text})
}

func mkAllowlist(t *testing.T, ids ...int64) *auth.Allowlist {
	t.Helper()
	a, err := auth.NewAllowlist(ids)
	if err != nil {
		t.Fatalf("allowlist: %v", err)
	}
	return a
}

func TestExtractIDs(t *testing.T) {
	tests := []struct {
		name     string
		update   *models.Update
		wantUser int64
		wantChat int64
	}{
		{"nil update", nil, 0, 0},
		{"message with from", &models.Update{Message: &models.Message{
			From: &models.User{ID: 11},
			Chat: models.Chat{ID: 22},
		}}, 11, 22},
		{"message without from (anonymous)", &models.Update{Message: &models.Message{
			Chat: models.Chat{ID: 22},
		}}, 0, 22},
		{"empty update", &models.Update{}, 0, 0},
		{"callback query", &models.Update{CallbackQuery: &models.CallbackQuery{
			From: models.User{ID: 33},
			Message: models.MaybeInaccessibleMessage{
				Message: &models.Message{Chat: models.Chat{ID: 77}},
			},
		}}, 33, 77},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractUserID(tt.update); got != tt.wantUser {
				t.Errorf("extractUserID = %d, want %d", got, tt.wantUser)
			}
			if got := extractChatID(tt.update); got != tt.wantChat {
				t.Errorf("extractChatID = %d, want %d", got, tt.wantChat)
			}
		})
	}
}

func TestAllowlistMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		update        *models.Update
		wantNextCalls int
		wantDenials   int
	}{
		{
			name: "allowed user passes through",
			update: &models.Update{Message: &models.Message{
				From: &models.User{ID: 42},
				Chat: models.Chat{ID: 100},
			}},
			wantNextCalls: 1,
			wantDenials:   0,
		},
		{
			name: "denied user blocked and notified",
			update: &models.Update{Message: &models.Message{
				From: &models.User{ID: 999},
				Chat: models.Chat{ID: 100},
			}},
			wantNextCalls: 0,
			wantDenials:   1,
		},
		{
			name:          "update with no user passes through silently",
			update:        &models.Update{},
			wantNextCalls: 1,
			wantDenials:   0,
		},
		{
			name: "denied user with no chat (edge case) — not notified but blocked",
			update: &models.Update{Message: &models.Message{
				From: &models.User{ID: 999},
			}},
			wantNextCalls: 0,
			wantDenials:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := mkAllowlist(t, 42)
			sender := &recordingSender{}
			nextCalls := 0
			next := bot.HandlerFunc(func(ctx context.Context, b *bot.Bot, u *models.Update) {
				nextCalls++
			})

			mw := allowlistMiddleware(list, sender.send)
			mw(next)(context.Background(), nil, tt.update)

			if nextCalls != tt.wantNextCalls {
				t.Errorf("next calls = %d, want %d", nextCalls, tt.wantNextCalls)
			}
			if len(sender.sent) != tt.wantDenials {
				t.Errorf("denial messages = %d, want %d", len(sender.sent), tt.wantDenials)
			}
			if tt.wantDenials > 0 && sender.sent[0].text != telegram.MsgDenied {
				t.Errorf("denial text = %q, want %q", sender.sent[0].text, telegram.MsgDenied)
			}
		})
	}
}
