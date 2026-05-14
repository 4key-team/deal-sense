package telegram

import (
	"context"
	"sync"
	"time"
)

// LLMSessionsOption tunes an InMemoryLLMWizardSessions instance.
type LLMSessionsOption func(*InMemoryLLMWizardSessions)

// WithLLMSessionTTL overrides the eviction threshold for stale /llm
// wizard sessions. Default matches the profile wizard (30 min).
func WithLLMSessionTTL(d time.Duration) LLMSessionsOption {
	return func(s *InMemoryLLMWizardSessions) { s.ttl = d }
}

// WithLLMSessionClock overrides the clock used to decide expiry.
// Production passes time.Now; tests pass a deterministic fake.
func WithLLMSessionClock(now func() time.Time) LLMSessionsOption {
	return func(s *InMemoryLLMWizardSessions) { s.now = now }
}

// InMemoryLLMWizardSessions is the production LLMWizardSessions
// implementation: a sync.Map keyed by chatID with a periodic sweeper
// that evicts drafts older than ttl. State is lost on process restart —
// acceptable because the wizard takes seconds and /llm edit restarts it.
type InMemoryLLMWizardSessions struct {
	m   sync.Map // chatID (int64) -> *LLMWizardState
	ttl time.Duration
	now func() time.Time
}

// NewInMemoryLLMWizardSessions constructs an empty session store. Without
// options the production defaults (TTL 30 min, real clock) are used.
func NewInMemoryLLMWizardSessions(opts ...LLMSessionsOption) *InMemoryLLMWizardSessions {
	s := &InMemoryLLMWizardSessions{ttl: defaultSessionTTL, now: time.Now}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *InMemoryLLMWizardSessions) Get(chatID int64) (*LLMWizardState, bool) {
	v, ok := s.m.Load(chatID)
	if !ok {
		return nil, false
	}
	state, ok := v.(*LLMWizardState)
	return state, ok
}

func (s *InMemoryLLMWizardSessions) Set(chatID int64, state *LLMWizardState) {
	s.m.Store(chatID, state)
}

func (s *InMemoryLLMWizardSessions) Clear(chatID int64) {
	s.m.Delete(chatID)
}

// Sweep removes sessions whose StartedAt is older than ttl. Sessions with
// a zero StartedAt are kept (defensive — they look "very old" to a naive
// check but are likely caller misuse, not abandoned flows). Returns the
// number of entries evicted.
func (s *InMemoryLLMWizardSessions) Sweep() int {
	cutoff := s.now().Add(-s.ttl)
	removed := 0
	s.m.Range(func(key, value any) bool {
		state, ok := value.(*LLMWizardState)
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

// Run sweeps the store every tick until ctx is canceled. Caller invokes
// it in a goroutine — the function blocks for the lifetime of the bot.
func (s *InMemoryLLMWizardSessions) Run(ctx context.Context, tick time.Duration) {
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
