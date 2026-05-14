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
// that evicts drafts older than ttl. RED stub — Get/Set/Clear/Sweep/Run
// all no-op so the package compiles but the tests fail.
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
	// RED stub
	_ = chatID
	return nil, false
}

func (s *InMemoryLLMWizardSessions) Set(chatID int64, state *LLMWizardState) {
	// RED stub
	_ = chatID
	_ = state
}

func (s *InMemoryLLMWizardSessions) Clear(chatID int64) {
	// RED stub
	_ = chatID
}

// Sweep removes sessions whose StartedAt is older than ttl. RED stub
// returns 0 evictions.
func (s *InMemoryLLMWizardSessions) Sweep() int {
	return 0
}

// Run sweeps the store every tick until ctx is canceled. RED stub
// returns immediately.
func (s *InMemoryLLMWizardSessions) Run(ctx context.Context, tick time.Duration) {
	_ = ctx
	_ = tick
}
