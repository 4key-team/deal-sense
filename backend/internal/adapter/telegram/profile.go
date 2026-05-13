package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// ProfileHandler implements the /profile command and the wizard that fills
// it. Persistence lives in ProfileStore; per-chat wizard state in WizardSessions.
type ProfileHandler struct {
	profiles usecase.ProfileStore
	sessions WizardSessions
	replier  Replier
}

// NewProfileHandler wires the dependencies for /profile.
func NewProfileHandler(profiles usecase.ProfileStore, sessions WizardSessions, replier Replier) *ProfileHandler {
	return &ProfileHandler{profiles: profiles, sessions: sessions, replier: replier}
}

// HandleCommand dispatches on the subcommand following "/profile".
//
//	/profile         → show the saved profile or a hint to create one
//	/profile edit    → reset the wizard at StepName and ask the first question
//	/profile clear   → delete the saved profile
func (h *ProfileHandler) HandleCommand(ctx context.Context, u *Update) error {
	arg := strings.TrimSpace(strings.TrimPrefix(u.Text, "/profile"))
	switch arg {
	case "":
		return h.showProfile(ctx, u.ChatID)
	case "edit":
		return h.startWizard(ctx, u.ChatID)
	case "clear":
		return h.clearProfile(ctx, u.ChatID)
	default:
		return h.replier.Reply(ctx, u.ChatID, msgProfileUnknownCmd)
	}
}

func (h *ProfileHandler) showProfile(ctx context.Context, chatID int64) error {
	p, ok, err := h.profiles.Get(ctx, chatID)
	if err != nil {
		return h.replier.Reply(ctx, chatID, fmt.Sprintf("%s %s", msgProfileLoadError, err.Error()))
	}
	if !ok {
		return h.replier.Reply(ctx, chatID, msgProfileEmpty)
	}
	return h.replier.Reply(ctx, chatID, fmt.Sprintf(msgProfileShowFmt, p.Render()))
}

func (h *ProfileHandler) startWizard(ctx context.Context, chatID int64) error {
	h.sessions.Set(chatID, &WizardState{
		ChatID:    chatID,
		Step:      StepName,
		Draft:     &ProfileDraft{},
		StartedAt: time.Now(),
	})
	return h.replier.Reply(ctx, chatID, msgWizardStart)
}

func (h *ProfileHandler) clearProfile(ctx context.Context, chatID int64) error {
	if err := h.profiles.Clear(ctx, chatID); err != nil {
		return h.replier.Reply(ctx, chatID, fmt.Sprintf("%s %s", msgProfileSaveError, err.Error()))
	}
	return h.replier.Reply(ctx, chatID, msgProfileCleared)
}
