package profilestore_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/profilestore"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

func newProfileT(t *testing.T) *domain.CompanyProfile {
	t.Helper()
	p, err := domain.NewCompanyProfile(
		"Acme", "15", "7",
		[]string{"Go", "React"},
		[]string{"ISO 9001"},
		[]string{"backend"},
		"Sber",
		"Remote-first",
	)
	if err != nil {
		t.Fatalf("NewCompanyProfile: %v", err)
	}
	return p
}

func TestFileStore_Get_NotFound(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	got, ok, err := s.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Errorf("Get found = true for empty store, want false (got %+v)", got)
	}
}

func TestFileStore_SetThenGet_RoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	prof := newProfileT(t)
	if err := s.Set(context.Background(), 42, prof); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := s.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get found = false after Set")
	}
	if got.Render() != prof.Render() {
		t.Errorf("Render diff after round-trip:\n  got=%q\n  want=%q", got.Render(), prof.Render())
	}
}

func TestFileStore_PersistsAcrossInstances(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s1, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore #1: %v", err)
	}
	prof := newProfileT(t)
	if err := s1.Set(context.Background(), 42, prof); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// New instance pointing at the same file must see the persisted entry.
	s2, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore #2: %v", err)
	}
	got, ok, err := s2.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get found = false on reloaded store")
	}
	if got.Render() != prof.Render() {
		t.Errorf("Render diff after reload:\n  got=%q\n  want=%q", got.Render(), prof.Render())
	}
}

func TestFileStore_Clear_DeletesProfile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	prof := newProfileT(t)
	if err := s.Set(context.Background(), 42, prof); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s.Clear(context.Background(), 42); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	_, ok, err := s.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Error("Get found = true after Clear, want false")
	}
}

func TestFileStore_Clear_AbsentChat_NoError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := s.Clear(context.Background(), 999); err != nil {
		t.Errorf("Clear on absent chat = %v, want nil", err)
	}
}

func TestFileStore_Set_OverwritesExisting(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	first, err := domain.NewCompanyProfile("First", "", "", nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := domain.NewCompanyProfile("Second", "", "", nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if err := s.Set(context.Background(), 42, first); err != nil {
		t.Fatalf("Set first: %v", err)
	}
	if err := s.Set(context.Background(), 42, second); err != nil {
		t.Fatalf("Set second: %v", err)
	}
	got, _, _ := s.Get(context.Background(), 42)
	if got.Render() != second.Render() {
		t.Errorf("expected overwrite to keep Second, got %q", got.Render())
	}
}

func TestFileStore_ConcurrentDifferentChats_NoRace(t *testing.T) {
	p := filepath.Join(t.TempDir(), "profiles.json")
	s, err := profilestore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	prof := newProfileT(t)

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if err := s.Set(context.Background(), id, prof); err != nil {
				t.Errorf("Set %d: %v", id, err)
			}
			_, _, _ = s.Get(context.Background(), id)
		}(int64(i))
	}
	wg.Wait()
}
