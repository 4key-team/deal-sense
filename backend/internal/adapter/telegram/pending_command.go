package telegram

import (
	"sync"
	"time"
)

// PendingCommandKind enumerates the commands that can be left "pending" —
// i.e. the user typed the command without attaching a file, and the bot
// should now route the next document upload from that chat through the
// corresponding handler.
type PendingCommandKind string

const (
	PendingAnalyze  PendingCommandKind = "analyze"
	PendingGenerate PendingCommandKind = "generate"
)

// CollectedFile is a document accumulated during a pending command. The
// first file in PendingCommandState.Files is the primary input (template
// for /generate, tender for /analyze); subsequent entries are context
// (briefs, ZIPs, price lists) the user attaches before /go.
type CollectedFile struct {
	FileID   string
	Filename string
	Data     []byte
}

// PendingCommandState records a chat-scoped pending command and when it
// was issued. StartedAt drives TTL eviction so abandoned states (user
// typed /analyze then walked away) do not pile up in memory. Files
// accumulates documents the user uploaded while in this pending command;
// /go finalises them, /cancel discards them.
type PendingCommandState struct {
	ChatID    int64
	Kind      PendingCommandKind
	StartedAt time.Time
	Files     []CollectedFile
}

// defaultPendingTTL bounds how long a pending command sits in memory
// waiting for a file upload. 10 min is short enough that a stale state
// won't surprise the user on a different document upload tomorrow, long
// enough to cover slow uploads on flaky mobile networks.
const defaultPendingTTL = 10 * time.Minute

// PendingCommandSessionsOption tunes an InMemoryPendingCommandSessions.
type PendingCommandSessionsOption func(*InMemoryPendingCommandSessions)

// WithPendingTTL overrides the eviction threshold for stale pending
// commands.
func WithPendingTTL(d time.Duration) PendingCommandSessionsOption {
	return func(s *InMemoryPendingCommandSessions) { s.ttl = d }
}

// WithPendingClock overrides the clock used to decide expiry. Production
// passes time.Now; tests pass a deterministic fake.
func WithPendingClock(now func() time.Time) PendingCommandSessionsOption {
	return func(s *InMemoryPendingCommandSessions) { s.now = now }
}

// InMemoryPendingCommandSessions is the in-memory implementation backing
// the two-step file workflow. State is lost on process restart, which is
// acceptable — the user simply re-issues the command.
type InMemoryPendingCommandSessions struct {
	m   sync.Map // chatID (int64) -> *PendingCommandState
	ttl time.Duration
	now func() time.Time
}

// NewInMemoryPendingCommandSessions constructs an empty store. Without
// options the production defaults are used (TTL 10 min, real clock).
func NewInMemoryPendingCommandSessions(opts ...PendingCommandSessionsOption) *InMemoryPendingCommandSessions {
	s := &InMemoryPendingCommandSessions{ttl: defaultPendingTTL, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get returns the pending kind for chatID. ok=false when no command is
// pending for that chat.
func (s *InMemoryPendingCommandSessions) Get(chatID int64) (PendingCommandKind, bool) {
	v, ok := s.m.Load(chatID)
	if !ok {
		return "", false
	}
	state, ok := v.(*PendingCommandState)
	if !ok {
		return "", false
	}
	return state.Kind, true
}

// Set records that chatID issued kind and is now waiting for a file.
// Overwrites any previous pending kind for that chat (latest wins) and
// resets the collected files — restarting a flow always starts empty.
func (s *InMemoryPendingCommandSessions) Set(chatID int64, kind PendingCommandKind) {
	s.m.Store(chatID, &PendingCommandState{
		ChatID:    chatID,
		Kind:      kind,
		StartedAt: s.now(),
	})
}

// SetState is a low-level setter used by tests to inject a state with
// specific fields (e.g. zero StartedAt). Production code should use Set.
func (s *InMemoryPendingCommandSessions) SetState(chatID int64, state PendingCommandState) {
	s.m.Store(chatID, &state)
}

// Clear removes the pending state for chatID. Missing entries are a no-op.
func (s *InMemoryPendingCommandSessions) Clear(chatID int64) {
	s.m.Delete(chatID)
}

// AppendFile adds f to the collected files for chatID. If no pending
// command exists for chatID, the call is a no-op — a stray upload must
// not implicitly start a flow.
func (s *InMemoryPendingCommandSessions) AppendFile(chatID int64, f CollectedFile) {
	v, ok := s.m.Load(chatID)
	if !ok {
		return
	}
	state, ok := v.(*PendingCommandState)
	if !ok {
		return
	}
	state.Files = append(state.Files, f)
}

// Files returns a snapshot of the documents collected so far for chatID.
// Returns nil/empty when no pending command exists or no files yet.
func (s *InMemoryPendingCommandSessions) Files(chatID int64) []CollectedFile {
	v, ok := s.m.Load(chatID)
	if !ok {
		return nil
	}
	state, ok := v.(*PendingCommandState)
	if !ok {
		return nil
	}
	if len(state.Files) == 0 {
		return nil
	}
	out := make([]CollectedFile, len(state.Files))
	copy(out, state.Files)
	return out
}

// Sweep removes states whose StartedAt is older than ttl. Zero StartedAt
// is defensively kept (likely caller misuse, not an abandoned flow).
// Returns the number of entries evicted.
func (s *InMemoryPendingCommandSessions) Sweep() int {
	cutoff := s.now().Add(-s.ttl)
	removed := 0
	s.m.Range(func(key, value any) bool {
		state, ok := value.(*PendingCommandState)
		if !ok {
			return true
		}
		if state.StartedAt.IsZero() {
			return true
		}
		if state.StartedAt.Before(cutoff) {
			s.m.Delete(key)
			removed++
		}
		return true
	})
	return removed
}
