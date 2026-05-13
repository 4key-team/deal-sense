package telegram

import (
	"context"
	"sync"
	"time"
)

// defaultSessionTTL bounds how long a half-finished wizard sits in memory
// before the sweeper discards it. Picked at 30 min — long enough to walk
// the wizard at human pace, short enough to keep RAM bounded even under
// abandoned flows.
const defaultSessionTTL = 30 * time.Minute

// SessionsOption tunes an InMemoryWizardSessions instance. Use the
// WithSession*-prefixed helpers to construct them.
type SessionsOption func(*InMemoryWizardSessions)

// WithSessionTTL overrides the eviction threshold for stale sessions.
func WithSessionTTL(d time.Duration) SessionsOption {
	return func(s *InMemoryWizardSessions) { s.ttl = d }
}

// WithSessionClock overrides the clock used to decide expiry. Production
// passes time.Now; tests pass a deterministic fake.
func WithSessionClock(now func() time.Time) SessionsOption {
	return func(s *InMemoryWizardSessions) { s.now = now }
}

// InMemoryWizardSessions is the production WizardSessions implementation:
// a sync.Map keyed by chatID with a periodic sweeper that evicts wizard
// drafts older than ttl. State is lost on process restart — acceptable
// because the wizard only takes seconds and a fresh /profile edit restarts it.
type InMemoryWizardSessions struct {
	m   sync.Map // chatID (int64) -> *WizardState
	ttl time.Duration
	now func() time.Time
}

// NewInMemoryWizardSessions constructs an empty session store. Without
// options the production defaults are used (TTL 30 min, real clock).
func NewInMemoryWizardSessions(opts ...SessionsOption) *InMemoryWizardSessions {
	s := &InMemoryWizardSessions{ttl: defaultSessionTTL, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *InMemoryWizardSessions) Get(chatID int64) (*WizardState, bool) {
	v, ok := s.m.Load(chatID)
	if !ok {
		return nil, false
	}
	state, ok := v.(*WizardState)
	return state, ok
}

func (s *InMemoryWizardSessions) Set(chatID int64, state *WizardState) {
	s.m.Store(chatID, state)
}

func (s *InMemoryWizardSessions) Clear(chatID int64) {
	s.m.Delete(chatID)
}

// Sweep removes sessions whose StartedAt is older than ttl. Sessions with a
// zero StartedAt are kept (defensive — they look "very old" to a naive
// check but are likely caller misuse, not abandoned flows). Returns the
// number of entries evicted.
func (s *InMemoryWizardSessions) Sweep() int {
	cutoff := s.now().Add(-s.ttl)
	removed := 0
	s.m.Range(func(key, value any) bool {
		state, ok := value.(*WizardState)
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

// Run sweeps the store every tick until ctx is canceled. Caller invokes it
// in a goroutine — the function blocks for the lifetime of the bot. Errors
// are impossible (Sweep never fails); count is left for callers who want
// to log it via a wrapper.
func (s *InMemoryWizardSessions) Run(ctx context.Context, tick time.Duration) {
	if err := ctx.Err(); err != nil {
		return
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Sweep()
		}
	}
}
