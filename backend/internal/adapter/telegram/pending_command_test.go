package telegram_test

import (
	"testing"
	"time"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

func TestPendingCommandKind_AcceptsKnownValues(t *testing.T) {
	cases := []struct {
		kind telegram.PendingCommandKind
		want string
	}{
		{telegram.PendingAnalyze, "analyze"},
		{telegram.PendingGenerate, "generate"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := string(tc.kind); got != tc.want {
				t.Errorf("kind = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPendingCommandSessions_Set_then_Get(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingAnalyze)

	got, ok := s.Get(42)
	if !ok {
		t.Fatal("Get(42) ok=false, want true after Set")
	}
	if got != telegram.PendingAnalyze {
		t.Errorf("Get(42) = %q, want analyze", got)
	}
}

func TestPendingCommandSessions_Get_Absent_ReturnsFalse(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	if _, ok := s.Get(999); ok {
		t.Error("Get on absent chat must return ok=false")
	}
}

func TestPendingCommandSessions_Clear(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingAnalyze)
	s.Clear(42)
	if _, ok := s.Get(42); ok {
		t.Error("Clear should remove the pending command")
	}
}

func TestPendingCommandSessions_Set_OverwritesPreviousKind(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingAnalyze)
	s.Set(42, telegram.PendingGenerate)
	got, ok := s.Get(42)
	if !ok {
		t.Fatal("Get after Set ok=false")
	}
	if got != telegram.PendingGenerate {
		t.Errorf("kind = %q, want generate (latest Set wins)", got)
	}
}

func TestPendingCommandSessions_Sweep_EvictsStale(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	clock := base
	s := telegram.NewInMemoryPendingCommandSessions(
		telegram.WithPendingTTL(5*time.Minute),
		telegram.WithPendingClock(func() time.Time { return clock }),
	)

	// Stage one pending command at t=0.
	s.Set(1, telegram.PendingAnalyze)

	// Advance the clock past TTL.
	clock = base.Add(10 * time.Minute)

	removed := s.Sweep()
	if removed != 1 {
		t.Errorf("Sweep removed = %d, want 1", removed)
	}
	if _, ok := s.Get(1); ok {
		t.Error("expired pending command must be gone after Sweep")
	}
}

func TestPendingCommandSessions_Sweep_KeepsFresh(t *testing.T) {
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	s := telegram.NewInMemoryPendingCommandSessions(
		telegram.WithPendingTTL(5*time.Minute),
		telegram.WithPendingClock(func() time.Time { return base }),
	)
	s.Set(1, telegram.PendingAnalyze)

	removed := s.Sweep()
	if removed != 0 {
		t.Errorf("Sweep removed = %d, want 0 (fresh session)", removed)
	}
	if _, ok := s.Get(1); !ok {
		t.Error("fresh pending command must survive Sweep")
	}
}

func TestPendingCommandSessions_Sweep_ZeroStartedAtKept(t *testing.T) {
	// Defensive: a malformed session with no StartedAt must not be evicted
	// just because it looks "infinitely old" to a naive cutoff check.
	base := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	s := telegram.NewInMemoryPendingCommandSessions(
		telegram.WithPendingTTL(5*time.Minute),
		telegram.WithPendingClock(func() time.Time { return base }),
	)
	// SetState lets us inject a state with zero StartedAt for the test.
	s.SetState(1, telegram.PendingCommandState{ChatID: 1, Kind: telegram.PendingAnalyze}) // zero time

	if removed := s.Sweep(); removed != 0 {
		t.Errorf("Sweep removed = %d, want 0 (zero StartedAt is kept)", removed)
	}
}

// --- multi-file collection ------------------------------------------------

func TestPendingCommandSessions_AppendFile_StartsEmpty(t *testing.T) {
	// A fresh pending state starts with no collected files; that's what
	// drives the wizard prompt "Пришлите шаблон".
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingGenerate)
	files := s.Files(42)
	if len(files) != 0 {
		t.Errorf("Files = %d, want 0 for fresh state", len(files))
	}
}

func TestPendingCommandSessions_AppendFile_Accumulates(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingGenerate)

	s.AppendFile(42, telegram.CollectedFile{Filename: "template.docx", Data: []byte("T")})
	s.AppendFile(42, telegram.CollectedFile{Filename: "brief.zip", Data: []byte("B")})

	files := s.Files(42)
	if len(files) != 2 {
		t.Fatalf("Files = %d, want 2", len(files))
	}
	if files[0].Filename != "template.docx" {
		t.Errorf("Files[0].Filename = %q, want template.docx", files[0].Filename)
	}
	if files[1].Filename != "brief.zip" {
		t.Errorf("Files[1].Filename = %q, want brief.zip", files[1].Filename)
	}
}

func TestPendingCommandSessions_AppendFile_NoActiveSession_NoOp(t *testing.T) {
	// AppendFile on a chat without a pending command must not silently
	// create one — that would be a footgun where a stray upload starts
	// a phantom flow.
	s := telegram.NewInMemoryPendingCommandSessions()
	s.AppendFile(42, telegram.CollectedFile{Filename: "x.docx"})
	if files := s.Files(42); len(files) != 0 {
		t.Errorf("Files = %d, want 0 (no session)", len(files))
	}
	if _, ok := s.Get(42); ok {
		t.Error("AppendFile must not create a session implicitly")
	}
}

func TestPendingCommandSessions_Files_AbsentChat_ReturnsEmpty(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	if files := s.Files(999); len(files) != 0 {
		t.Errorf("Files on absent chat = %d, want 0", len(files))
	}
}

func TestPendingCommandSessions_Clear_AlsoClearsFiles(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingGenerate)
	s.AppendFile(42, telegram.CollectedFile{Filename: "t.docx", Data: []byte("T")})
	s.Clear(42)
	if files := s.Files(42); len(files) != 0 {
		t.Errorf("Files after Clear = %d, want 0", len(files))
	}
}

func TestPendingCommandSessions_Set_ResetsFiles(t *testing.T) {
	// Set replaces the kind; per "latest Set wins" semantics it must also
	// reset the file collection — accumulated /generate files should not
	// leak into a fresh /analyze that the user kicks off afterwards.
	s := telegram.NewInMemoryPendingCommandSessions()
	s.Set(42, telegram.PendingGenerate)
	s.AppendFile(42, telegram.CollectedFile{Filename: "t.docx", Data: []byte("T")})

	s.Set(42, telegram.PendingAnalyze)
	if files := s.Files(42); len(files) != 0 {
		t.Errorf("Files after Set to a new kind = %d, want 0 (reset)", len(files))
	}
}

func TestPendingCommandSessions_ConcurrentSetGet_NoRace(t *testing.T) {
	s := telegram.NewInMemoryPendingCommandSessions()
	done := make(chan struct{}, 2)
	go func() {
		for range 200 {
			s.Set(1, telegram.PendingAnalyze)
		}
		done <- struct{}{}
	}()
	go func() {
		for range 200 {
			_, _ = s.Get(1)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}
