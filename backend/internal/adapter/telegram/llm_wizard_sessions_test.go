package telegram_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

func TestInMemoryLLMWizardSessions_GetMissing_ReturnsFalse(t *testing.T) {
	ws := telegram.NewInMemoryLLMWizardSessions()
	got, ok := ws.Get(42)
	if ok {
		t.Errorf("Get on empty store ok = true, want false (got %+v)", got)
	}
}

func TestInMemoryLLMWizardSessions_SetThenGet_ReturnsSamePointer(t *testing.T) {
	ws := telegram.NewInMemoryLLMWizardSessions()
	state := &telegram.LLMWizardState{
		ChatID: 42,
		Step:   telegram.StepLLMProvider,
		Draft:  &telegram.LLMSettingsDraft{},
	}
	ws.Set(42, state)

	got, ok := ws.Get(42)
	if !ok {
		t.Fatal("Get after Set ok = false")
	}
	if got != state {
		t.Errorf("Get returned %p, want %p", got, state)
	}
}

func TestInMemoryLLMWizardSessions_Clear_RemovesEntry(t *testing.T) {
	ws := telegram.NewInMemoryLLMWizardSessions()
	ws.Set(42, &telegram.LLMWizardState{ChatID: 42})
	ws.Clear(42)
	if _, ok := ws.Get(42); ok {
		t.Error("Get after Clear ok = true, want false")
	}
}

func TestInMemoryLLMWizardSessions_Clear_AbsentNoOp(t *testing.T) {
	ws := telegram.NewInMemoryLLMWizardSessions()
	ws.Clear(999) // must not panic
}

func TestInMemoryLLMWizardSessions_ConcurrentSafe(t *testing.T) {
	ws := telegram.NewInMemoryLLMWizardSessions()
	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			ws.Set(id, &telegram.LLMWizardState{
				ChatID: id, Step: telegram.StepLLMProvider,
				Draft: &telegram.LLMSettingsDraft{},
			})
			_, _ = ws.Get(id)
			if id%2 == 0 {
				ws.Clear(id)
			}
		}(int64(i))
	}
	wg.Wait()
}

// --- TTL / Sweep / Run ---------------------------------------------------

func TestInMemoryLLMWizardSessions_Sweep_RemovesExpired(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base) // reuse fakeClock from wizard_sessions_test.go
	ws := telegram.NewInMemoryLLMWizardSessions(
		telegram.WithLLMSessionTTL(5*time.Minute),
		telegram.WithLLMSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.LLMWizardState{
		ChatID: 42, Step: telegram.StepLLMProvider,
		StartedAt: base.Add(-10 * time.Minute),
	})
	ws.Set(7, &telegram.LLMWizardState{
		ChatID: 7, Step: telegram.StepLLMProvider,
		StartedAt: base.Add(-1 * time.Minute),
	})

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

func TestInMemoryLLMWizardSessions_Sweep_SkipsZeroStartedAt(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base)
	ws := telegram.NewInMemoryLLMWizardSessions(
		telegram.WithLLMSessionTTL(5*time.Minute),
		telegram.WithLLMSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.LLMWizardState{ChatID: 42}) // zero StartedAt
	if removed := ws.Sweep(); removed != 0 {
		t.Errorf("Sweep removed %d for zero-StartedAt, want 0", removed)
	}
	if _, ok := ws.Get(42); !ok {
		t.Error("zero-StartedAt session must not be evicted")
	}
}

func TestInMemoryLLMWizardSessions_Run_SweepsPeriodically_AndReturnsOnCtxCancel(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(base)
	ws := telegram.NewInMemoryLLMWizardSessions(
		telegram.WithLLMSessionTTL(1*time.Millisecond),
		telegram.WithLLMSessionClock(clock.Now),
	)
	ws.Set(42, &telegram.LLMWizardState{ChatID: 42, StartedAt: base.Add(-1 * time.Hour)})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		ws.Run(ctx, 5*time.Millisecond)
		close(done)
	}()

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
	case <-time.After(time.Second):
		t.Error("Run did not return after ctx cancel")
	}
}

func TestInMemoryLLMWizardSessions_Run_ReturnsImmediately_IfCtxAlreadyDone(t *testing.T) {
	ws := telegram.NewInMemoryLLMWizardSessions()
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
