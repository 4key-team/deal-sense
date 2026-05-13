package telegram_test

import (
	"sync"
	"testing"

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
