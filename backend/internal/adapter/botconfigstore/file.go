// Package botconfigstore implements usecase/botconfig.Repository on top of
// a single JSON file. Unlike profilestore (one record per chat) the bot has
// one configuration record per deployment, so the on-disk shape is a flat
// object, not a map.
package botconfigstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// ErrCorruptStore wraps any failure that indicates the on-disk file is no
// longer a valid representation of a BotConfig (malformed JSON, or values
// that violate domain invariants — e.g. someone hand-edited the file).
// Callers should errors.Is against this when deciding whether to surface
// the problem to operators vs. transparently reset.
var ErrCorruptStore = errors.New("botconfigstore: corrupt store")

// FileStore is a sync.RWMutex-guarded cache of the single bot config,
// mirrored to disk. Writes rewrite the whole file atomically (tmp + rename).
type FileStore struct {
	path string

	mu  sync.RWMutex
	cur *domain.BotConfig // nil = no config saved yet
}

// botConfigDTO is the on-disk shape. Pointer fields would round-trip nil
// vs. zero-value cleanly, but Telegram tokens and log levels never have a
// meaningful zero value so plain strings suffice. allowlist_user_ids is
// omitted when empty (= open mode).
type botConfigDTO struct {
	Token            string  `json:"token"`
	AllowlistUserIDs []int64 `json:"allowlist_user_ids,omitempty"`
	LogLevel         string  `json:"log_level"`
}

// NewFileStore creates a FileStore backed by path. If the file is missing,
// the store starts empty (Load returns ok=false). Missing parent directories
// are created. A malformed file aborts construction with ErrCorruptStore.
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("botconfigstore: path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("botconfigstore: mkdir parent: %w", err)
	}
	s := &FileStore{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FileStore) load() error {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("botconfigstore: read %s: %w", s.path, err)
	}
	if len(raw) == 0 {
		return nil
	}
	var dto botConfigDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return fmt.Errorf("%w: unmarshal: %v", ErrCorruptStore, err)
	}
	cfg, err := domain.NewBotConfig(dto.Token, dto.AllowlistUserIDs, dto.LogLevel)
	if err != nil {
		return fmt.Errorf("%w: invalid record: %v", ErrCorruptStore, err)
	}
	s.cur = cfg
	return nil
}

// Load returns the cached config. (nil, false, nil) when the store has no
// record (first run, or file absent).
func (s *FileStore) Load(_ context.Context) (*domain.BotConfig, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cur == nil {
		return nil, false, nil
	}
	return s.cur, true, nil
}

// Save replaces the cached config and rewrites the file atomically.
// A nil config is rejected — callers wanting to clear the store should
// instead remove the file out-of-band.
func (s *FileStore) Save(_ context.Context, cfg *domain.BotConfig) error {
	if cfg == nil {
		return errors.New("botconfigstore: nil config")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cur = cfg
	return s.persistLocked()
}

// persistLocked rewrites the file atomically. Caller must hold s.mu.
func (s *FileStore) persistLocked() error {
	dto := botConfigDTO{
		Token:    s.cur.Token(),
		LogLevel: s.cur.LogLevel().String(),
	}
	if !s.cur.Allowlist().IsOpen() {
		dto.AllowlistUserIDs = s.cur.Allowlist().Members()
	}
	raw, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return fmt.Errorf("botconfigstore: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".bot-config-*.json")
	if err != nil {
		return fmt.Errorf("botconfigstore: tmp create: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("botconfigstore: tmp write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("botconfigstore: tmp close: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("botconfigstore: rename: %w", err)
	}
	return nil
}
