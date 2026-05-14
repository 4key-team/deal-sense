package botconfigstore_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/botconfigstore"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

const fixtureValidToken = "8829614348:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"

func mustBotConfig(t *testing.T, token string, ids []int64, level string) *domain.BotConfig {
	t.Helper()
	cfg, err := domain.NewBotConfig(token, ids, level)
	if err != nil {
		t.Fatalf("NewBotConfig: %v", err)
	}
	return cfg
}

func TestNewFileStore_EmptyPath_Errors(t *testing.T) {
	_, err := botconfigstore.NewFileStore("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestFileStore_Load_MissingFile_ReturnsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	cfg, ok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if ok {
		t.Error("Load must return ok=false when file is absent")
	}
	if cfg != nil {
		t.Error("Load must return nil config when file is absent")
	}
}

func TestFileStore_Load_EmptyFile_ReturnsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	cfg, ok, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load on empty file: %v", err)
	}
	if ok || cfg != nil {
		t.Error("Load must return absent for empty file")
	}
}

func TestFileStore_Load_CorruptJSON_Errors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := botconfigstore.NewFileStore(path)
	if err == nil {
		t.Fatal("expected error when file has malformed JSON")
	}
}

func TestFileStore_Save_then_Load_RestrictedRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	saved := mustBotConfig(t, fixtureValidToken, []int64{42, 100}, "warn")
	if err := store.Save(context.Background(), saved); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Re-open to prove persistence (not just in-memory cache).
	reopened, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen NewFileStore: %v", err)
	}
	loaded, ok, err := reopened.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("Load must return ok=true after Save")
	}
	if loaded.Token() != fixtureValidToken {
		t.Errorf("token = %q, want %q", loaded.Token(), fixtureValidToken)
	}
	if loaded.LogLevel() != domain.LogLevelWarn {
		t.Errorf("log level = %q, want warn", loaded.LogLevel())
	}
	if loaded.Allowlist().IsOpen() {
		t.Error("allowlist should be restricted after round-trip with IDs")
	}
	if !loaded.Allowlist().IsAllowed(42) || !loaded.Allowlist().IsAllowed(100) {
		t.Error("allowlist members did not survive round-trip")
	}
}

func TestFileStore_Save_then_Load_OpenAllowlistRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	open := mustBotConfig(t, fixtureValidToken, nil, "info")
	if err := store.Save(context.Background(), open); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("reopen NewFileStore: %v", err)
	}
	loaded, ok, err := reopened.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("Load must return ok=true after Save (open mode)")
	}
	if !loaded.Allowlist().IsOpen() {
		t.Error("open mode must survive round-trip")
	}
	if !loaded.Allowlist().IsAllowed(123456) {
		t.Error("open allowlist must still admit any user")
	}
}

func TestFileStore_Save_Overwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	first := mustBotConfig(t, fixtureValidToken, []int64{1}, "info")
	if err := store.Save(context.Background(), first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	second := mustBotConfig(t, fixtureValidToken, []int64{42, 100}, "error")
	if err := store.Save(context.Background(), second); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	loaded, ok, err := store.Load(context.Background())
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if loaded.LogLevel() != domain.LogLevelError {
		t.Errorf("log level = %q, want error (overwrite lost)", loaded.LogLevel())
	}
	if loaded.Allowlist().IsAllowed(1) {
		t.Error("first save's allowlist must not survive overwrite")
	}
}

func TestFileStore_Save_NilConfig_Errors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot-config.json")
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := store.Save(context.Background(), nil); err == nil {
		t.Error("Save(nil) must return error")
	}
}

func TestFileStore_Save_CreatesParentDir(t *testing.T) {
	deep := filepath.Join(t.TempDir(), "a", "b", "c", "bot-config.json")
	store, err := botconfigstore.NewFileStore(deep)
	if err != nil {
		t.Fatalf("NewFileStore should create parent dirs: %v", err)
	}
	cfg := mustBotConfig(t, fixtureValidToken, []int64{42}, "info")
	if err := store.Save(context.Background(), cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(deep); err != nil {
		t.Errorf("file not created at %s: %v", deep, err)
	}
}

func TestFileStore_ConcurrentSaveAndLoad_NoRace(t *testing.T) {
	// Smoke test for the mutex contract — run -race surfaces violations.
	path := filepath.Join(t.TempDir(), "bot-config.json")
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	cfg := mustBotConfig(t, fixtureValidToken, []int64{42}, "info")
	if err := store.Save(context.Background(), cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	const iters = 50
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			if err := store.Save(context.Background(), cfg); err != nil {
				t.Errorf("save: %v", err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			if _, _, err := store.Load(context.Background()); err != nil {
				t.Errorf("load: %v", err)
				return
			}
		}
	}()
	wg.Wait()
}

// Sentinel — ensure exported error doesn't accidentally regress to nil.
func TestFileStore_PackageExports_ErrCorruptIsAvailable(t *testing.T) {
	if botconfigstore.ErrCorruptStore == nil {
		t.Fatal("ErrCorruptStore must be exported and non-nil")
	}
	// Build a corrupt file and verify Load wraps ErrCorruptStore.
	path := filepath.Join(t.TempDir(), "bot-config.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := botconfigstore.NewFileStore(path)
	if !errors.Is(err, botconfigstore.ErrCorruptStore) {
		t.Errorf("err = %v, want wrapping ErrCorruptStore", err)
	}
}
