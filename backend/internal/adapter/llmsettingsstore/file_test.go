package llmsettingsstore_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llmsettingsstore"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

func newCfgT(t *testing.T, suffix string) *domain.LLMSettings {
	t.Helper()
	cfg, err := domain.NewLLMSettings("openai", "https://openrouter.ai/api/v1", "sk-secret-"+suffix, "anthropic/claude-sonnet-4")
	if err != nil {
		t.Fatalf("NewLLMSettings: %v", err)
	}
	return cfg
}

func TestNewFileStore_EmptyPath_Error(t *testing.T) {
	_, err := llmsettingsstore.NewFileStore("")
	if err == nil {
		t.Fatal("NewFileStore(\"\") must return an error")
	}
}

func TestFileStore_Get_NotFound(t *testing.T) {
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
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
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	cfg := newCfgT(t, "round-trip")
	if err := s.Set(context.Background(), 42, cfg); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, ok, err := s.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("Get found = false after Set")
	}
	if got.Provider() != cfg.Provider() ||
		got.BaseURL() != cfg.BaseURL() ||
		got.APIKey() != cfg.APIKey() ||
		got.Model() != cfg.Model() {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, cfg)
	}
}

func TestFileStore_PersistsAcrossInstances(t *testing.T) {
	p := filepath.Join(t.TempDir(), "llm.json")
	s1, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore #1: %v", err)
	}
	cfg := newCfgT(t, "persist")
	if err := s1.Set(context.Background(), 42, cfg); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// New instance pointing at the same file must see the persisted entry.
	s2, err := llmsettingsstore.NewFileStore(p)
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
	if got.APIKey() != cfg.APIKey() {
		t.Errorf("APIKey diff after reload: got %q want %q", got.APIKey(), cfg.APIKey())
	}
}

func TestFileStore_Clear_DeletesSettings(t *testing.T) {
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	cfg := newCfgT(t, "to-clear")
	if err := s.Set(context.Background(), 42, cfg); err != nil {
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
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := s.Clear(context.Background(), 999); err != nil {
		t.Errorf("Clear on absent chat = %v, want nil", err)
	}
}

func TestFileStore_Set_OverwritesExisting(t *testing.T) {
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	first, err := domain.NewLLMSettings("openai", "", "sk-first", "gpt-4o")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := domain.NewLLMSettings("anthropic", "https://api.anthropic.com/v1", "sk-second", "claude-3-5-sonnet")
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
	if got.Provider() != "anthropic" || got.APIKey() != "sk-second" {
		t.Errorf("expected overwrite to keep Second, got %+v", got)
	}
}

func TestFileStore_Set_NilCfg_Error(t *testing.T) {
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := s.Set(context.Background(), 42, nil); err == nil {
		t.Error("Set(nil) must return an error")
	}
}

func TestFileStore_ConcurrentDifferentChats_NoRace(t *testing.T) {
	p := filepath.Join(t.TempDir(), "llm.json")
	s, err := llmsettingsstore.NewFileStore(p)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	cfg := newCfgT(t, "race")

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			if err := s.Set(context.Background(), id, cfg); err != nil {
				t.Errorf("Set %d: %v", id, err)
			}
			_, _, _ = s.Get(context.Background(), id)
		}(int64(i))
	}
	wg.Wait()
}
