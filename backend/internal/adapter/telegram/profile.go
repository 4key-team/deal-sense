package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/daniil/deal-sense/backend/internal/domain"
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

// Handler convention: bot-business errors (store read/write failure) surface
// as a stable user-facing Reply — internal err.Error() is not leaked. The
// transport error from the Reply itself is what bubbles up to the caller
// (so the bot runtime can log it once).

func (h *ProfileHandler) showProfile(ctx context.Context, chatID int64) error {
	p, ok, err := h.profiles.Get(ctx, chatID)
	if err != nil {
		return h.replier.Reply(ctx, chatID, msgProfileLoadError)
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
		return h.replier.Reply(ctx, chatID, msgProfileSaveError)
	}
	return h.replier.Reply(ctx, chatID, msgProfileCleared)
}

// HandleWizardInput consumes one user message while a wizard session is
// active. /cancel aborts the wizard; otherwise the message fills the
// current step's field, the session advances, and the next question (or
// the final confirmation) is sent.
func (h *ProfileHandler) HandleWizardInput(ctx context.Context, u *Update) error {
	state, ok := h.sessions.Get(u.ChatID)
	if !ok {
		// Defensive: dispatcher should not call this without a session.
		return nil
	}
	text := strings.TrimSpace(u.Text)
	if text == "/cancel" {
		h.sessions.Clear(u.ChatID)
		return h.replier.Reply(ctx, u.ChatID, msgWizardCancelled)
	}

	switch state.Step {
	case StepName:
		state.Draft.Name = parseSentinel(text)
		state.Step = StepTeamSize
		return h.replier.Reply(ctx, u.ChatID, msgWizardTeamSize)
	case StepTeamSize:
		state.Draft.TeamSize = parseSentinel(text)
		state.Step = StepExperience
		return h.replier.Reply(ctx, u.ChatID, msgWizardExperience)
	case StepExperience:
		state.Draft.Experience = parseSentinel(text)
		state.Step = StepTechStack
		return h.replier.Reply(ctx, u.ChatID, msgWizardTechStack)
	case StepTechStack:
		state.Draft.TechStack = parseList(text)
		state.Step = StepCertifications
		return h.replier.Reply(ctx, u.ChatID, msgWizardCerts)
	case StepCertifications:
		state.Draft.Certifications = parseList(text)
		state.Step = StepSpecializations
		return h.replier.Reply(ctx, u.ChatID, msgWizardSpecs)
	case StepSpecializations:
		state.Draft.Specializations = parseList(text)
		state.Step = StepKeyClients
		return h.replier.Reply(ctx, u.ChatID, msgWizardClients)
	case StepKeyClients:
		state.Draft.KeyClients = parseSentinel(text)
		state.Step = StepExtra
		return h.replier.Reply(ctx, u.ChatID, msgWizardExtra)
	case StepExtra:
		state.Draft.Extra = parseSentinel(text)
		return h.finalize(ctx, u.ChatID, state.Draft)
	default:
		// Unknown step — clear to recover from corrupted state.
		h.sessions.Clear(u.ChatID)
		return h.replier.Reply(ctx, u.ChatID, msgWizardCancelled)
	}
}

// finalize constructs the immutable profile, persists it, and clears the
// session. Any failure ends the wizard so the user is not stuck answering
// the same question repeatedly.
func (h *ProfileHandler) finalize(ctx context.Context, chatID int64, d *ProfileDraft) error {
	h.sessions.Clear(chatID)

	profile, err := domain.NewCompanyProfile(
		d.Name, d.TeamSize, d.Experience,
		d.TechStack, d.Certifications, d.Specializations,
		d.KeyClients, d.Extra,
	)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyCompany) {
			return h.replier.Reply(ctx, chatID, msgWizardEmptyProfile)
		}
		return h.replier.Reply(ctx, chatID, msgProfileSaveError)
	}
	if err := h.profiles.Set(ctx, chatID, profile); err != nil {
		return h.replier.Reply(ctx, chatID, msgProfileSaveError)
	}
	return h.replier.Reply(ctx, chatID, fmt.Sprintf(msgWizardConfirmFmt, profile.Render()))
}

// parseList splits a comma-separated answer into trimmed, non-empty items.
// skipSentinel ("-") is the agreed answer for "skip" and yields nil.
func parseList(text string) []string {
	s := strings.TrimSpace(text)
	if s == "" || s == skipSentinel {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseSentinel returns "" for the skipSentinel marker, otherwise the
// trimmed answer.
func parseSentinel(text string) string {
	s := strings.TrimSpace(text)
	if s == skipSentinel {
		return ""
	}
	return s
}
