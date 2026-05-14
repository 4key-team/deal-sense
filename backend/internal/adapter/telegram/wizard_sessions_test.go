package telegram_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

func TestInMemoryWizardSessions_GetMissing_ReturnsFalse(t *testing.T) {
	ws := telegram.NewInMemoryWizardSessions()
	got, ok := ws.Get(42)
	if ok {
		t.Errorf("Get on empty store ok = true, want false (got %+v)", got)
	}
}

func TestInMemoryWizardSessions_SetThenGet_ReturnsSamePointer(t *testing.T) {
	ws := telegram.NewInMemoryWizardSessions()
	state := &telegram.WizardState{ChatID: 42, Step: telegram.StepName}
	ws.Set(42, state)

	got, ok := ws.Get(42)
	if !ok {
		t.Fatal("Get after Set ok = false")
	}
	if got != state {
		t.Errorf("Get returned %p, want %p", got, state)
	}
}

func TestInMemoryWizardSessions_Clear_RemovesEntry(t *testing.T) {
	ws := telegram.NewInMemoryWizardSessions()
	ws.Set(42, &telegram.WizardState{ChatID: 42})
	ws.Clear(42)
	if _, ok := ws.Get(42); ok {
		t.Error("Get after Clear ok = true, want false")
	}
}

func TestInMemoryWizardSessions_Clear_AbsentNoOp(t *testing.T) {
	ws := telegram.NewInMemoryWizardSessions()
	// Should not panic.
	ws.Clear(999)
}

func TestInMemoryWizardSessions_ConcurrentSafe(t *testing.T) {
	ws := telegram.NewInMemoryWizardSessions()
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			ws.Set(id, &telegram.WizardState{ChatID: id, Step: telegram.StepName})
			_, _ = ws.Get(id)
			if id%2 == 0 {
				ws.Clear(id)
			}
		}(int64(i))
	}
	wg.Wait()
}

// --- TTL / Sweep / Run ---------------------------------------------------

// fakeClock is a deterministic clock for sweeper tests. Advance moves the
// "now" forward; Now is goroutine-safe under the mutex.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }
func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}
func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
}

func TestInMemoryWizardSessions_Sweep_RemovesExpired(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base)
	ws := telegram.NewInMemoryWizardSessions(
		telegram.WithSessionTTL(5*time.Minute),
		telegram.WithSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.WizardState{ChatID: 42, Step: telegram.StepName, StartedAt: base.Add(-10 * time.Minute)})
	ws.Set(7, &telegram.WizardState{ChatID: 7, Step: telegram.StepName, StartedAt: base.Add(-1 * time.Minute)})

	removed := ws.Sweep()
	if removed != 1 {
		t.Errorf("Sweep removed %d, want 1", removed)
	}
	if _, ok := ws.Get(42); ok {
		t.Error("chat 42 (stale) should be evicted")
	}
	if _, ok := ws.Get(7); !ok {
		t.Error("chat 7 (fresh) should survive")
	}
}

func TestInMemoryWizardSessions_Sweep_NoExpired_NoOp(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base)
	ws := telegram.NewInMemoryWizardSessions(
		telegram.WithSessionTTL(5*time.Minute),
		telegram.WithSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.WizardState{ChatID: 42, StartedAt: base.Add(-1 * time.Minute)})
	if removed := ws.Sweep(); removed != 0 {
		t.Errorf("Sweep removed %d, want 0", removed)
	}
}

func TestInMemoryWizardSessions_Sweep_SkipsZeroStartedAt(t *testing.T) {
	// A WizardState without StartedAt set must not be evicted on the first
	// sweep — that would be a hostile UX (any caller forgetting to set the
	// time loses their wizard immediately).
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base)
	ws := telegram.NewInMemoryWizardSessions(
		telegram.WithSessionTTL(5*time.Minute),
		telegram.WithSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.WizardState{ChatID: 42}) // zero StartedAt
	if removed := ws.Sweep(); removed != 0 {
		t.Errorf("Sweep removed %d for zero-StartedAt, want 0", removed)
	}
	if _, ok := ws.Get(42); !ok {
		t.Error("zero-StartedAt session must not be evicted")
	}
}

func TestInMemoryWizardSessions_DefaultTTL_30Minutes(t *testing.T) {
	// Sanity-check the production default: a session 1 minute old must
	// not be swept; one 31 minutes old must be.
	now := time.Now()
	ws := telegram.NewInMemoryWizardSessions()
	ws.Set(1, &telegram.WizardState{ChatID: 1, StartedAt: now.Add(-1 * time.Minute)})
	ws.Set(2, &telegram.WizardState{ChatID: 2, StartedAt: now.Add(-31 * time.Minute)})

	ws.Sweep()
	if _, ok := ws.Get(1); !ok {
		t.Error("1-min-old session should survive default 30-min TTL")
	}
	if _, ok := ws.Get(2); ok {
		t.Error("31-min-old session should be swept by default 30-min TTL")
	}
}

func TestInMemoryWizardSessions_Run_SweepsPeriodically_AndReturnsOnCtxCancel(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base)
	ws := telegram.NewInMemoryWizardSessions(
		telegram.WithSessionTTL(1*time.Millisecond),
		telegram.WithSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.WizardState{ChatID: 42, StartedAt: base.Add(-1 * time.Hour)})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, 5*time.Millisecond)
		close(done)
	}()

	// Poll for the sweeper to remove the stale session.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, ok := ws.Get(42); !ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if _, ok := ws.Get(42); ok {
		t.Error("Run did not evict stale session within 500ms")
	}

	cancel()
	select {
	case <-done:
		// graceful exit
	case <-time.After(time.Second):
		t.Error("Run did not return after ctx cancel")
	}
}

func TestInMemoryWizardSessions_Run_ReturnsImmediately_IfCtxAlreadyDone(t *testing.T) {
	ws := telegram.NewInMemoryWizardSessions()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan struct{})
	go func() {
		ws.Run(ctx, 5*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Run did not return on already-canceled ctx")
	}
}
