package telegram

import (
	"context"
	"fmt"
	"strings"

	usecase "github.com/daniil/deal-sense/backend/internal/usecase/telegram"
)

// PendingSessions is the narrow contract the pending router needs over
// the sessions store — keeps the matcher + dispatcher testable without
// the full InMemoryPendingCommandSessions surface.
type PendingSessions interface {
	Get(chatID int64) (PendingCommandKind, bool)
	AppendFile(chatID int64, f CollectedFile)
	Files(chatID int64) []CollectedFile
	Clear(chatID int64)
}

// ShouldRoutePending reports whether the incoming message belongs to the
// pending-command file collection flow. It is the matcher predicate that
// the runtime registers as a bot.MatchFunc.
//
// Returns true when:
//   - a pending command session exists for chatID, AND
//   - the message either has a document attached, OR
//   - the message text is /go or /cancel, OR
//   - the message text is free-form (not a known slash command — those
//     are routed by their own prefix handlers).
//
// Slash commands other than /go and /cancel are explicitly NOT claimed
// so /profile, /llm, /analyze, /generate, /start, /help keep working
// even when a pending session is active.
func ShouldRoutePending(text string, hasDoc bool, chatID int64, sessions PendingSessions) bool {
	if _, ok := sessions.Get(chatID); !ok {
		return false
	}
	if hasDoc {
		return true
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed == "/go" || trimmed == "/cancel" ||
			strings.HasPrefix(trimmed, "/go ") || strings.HasPrefix(trimmed, "/cancel ")
	}
	return true
}

// RoutePending consumes a message during an active pending-command
// session. Returns handled=true when the router took ownership; the
// transport error (if any) bubbles up so the caller can log it once.
//
// Branches:
//   - document → append to collected files and reply with progress.
//   - /go      → dispatch to the matching handler with the accumulated
//     files; first file is the primary input, rest is context.
//   - /cancel  → clear the session and reply.
//   - free text → reply with msgPendingTextHint (no fallback noise).
func RoutePending(
	ctx context.Context,
	u *Update,
	sessions PendingSessions,
	ah *AnalyzeHandler,
	gh *GenerateHandler,
	replier Replier,
) (bool, error) {
	kind, ok := sessions.Get(u.ChatID)
	if !ok {
		return false, nil
	}

	if u.Document != nil {
		sessions.AppendFile(u.ChatID, CollectedFile{
			FileID:   u.Document.FileID,
			Filename: u.Document.Filename,
			Data:     u.Document.Data,
		})
		count := len(sessions.Files(u.ChatID))
		return true, replier.Reply(ctx, u.ChatID,
			fmt.Sprintf(msgPendingFileAddedFmt, u.Document.Filename, count))
	}

	text := strings.TrimSpace(u.Text)
	switch text {
	case "/cancel":
		sessions.Clear(u.ChatID)
		return true, replier.Reply(ctx, u.ChatID, msgPendingCancelled)
	case "/go":
		files := sessions.Files(u.ChatID)
		if len(files) == 0 {
			return true, replier.Reply(ctx, u.ChatID, msgPendingGoNoFiles)
		}
		sessions.Clear(u.ChatID)
		return true, dispatchPending(ctx, u.ChatID, kind, files, ah, gh)
	}
	// Free-form text — gentle hint, no fallback noise.
	return true, replier.Reply(ctx, u.ChatID, msgPendingTextHint)
}

// dispatchPending forwards the collected files to the right backend
// handler. /analyze uses only the first file (single tender input);
// /generate treats the first file as the template and the remaining
// files as context attachments.
func dispatchPending(
	ctx context.Context,
	chatID int64,
	kind PendingCommandKind,
	files []CollectedFile,
	ah *AnalyzeHandler,
	gh *GenerateHandler,
) error {
	primary := files[0]
	switch kind {
	case PendingAnalyze:
		return ah.Handle(ctx, &Update{
			ChatID: chatID,
			Text:   "/analyze",
			Document: &Document{
				FileID:   primary.FileID,
				Filename: primary.Filename,
				Data:     primary.Data,
			},
		})
	case PendingGenerate:
		var ctxFiles []usecase.ContextFile
		for _, f := range files[1:] {
			ctxFiles = append(ctxFiles, usecase.ContextFile{
				Filename: f.Filename,
				Data:     f.Data,
			})
		}
		return gh.HandleCollected(ctx, chatID, primary, ctxFiles)
	}
	return nil
}
