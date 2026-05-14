// Package llmsettingsstore implements usecase/llmsettings.Repository on top
// of a JSON file. It is the persistence adapter for per-chat Telegram LLM
// provider settings: one file per deployment, single-bot scale.
package llmsettingsstore

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

// FileStore is a sync.RWMutex-guarded map mirrored to a JSON file on disk.
// Writes rewrite the whole file atomically (tmp file + rename).
type FileStore struct {
	path string
	mu   sync.RWMutex
	data map[int64]llmDTO
}

// llmDTO is the on-disk shape; it mirrors domain.LLMSettings fields 1-to-1
// so reloads return cfgs that round-trip every accessor.
type llmDTO struct {
	Provider string `json:"provider"`
	BaseURL  string `json:"base_url,omitempty"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// NewFileStore creates a FileStore backed by path. If the file is missing
// the store starts empty; missing parent directories are created.
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("llmsettingsstore: path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("llmsettingsstore: mkdir parent: %w", err)
	}
	s := &FileStore{path: path, data: map[int64]llmDTO{}}
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
		return fmt.Errorf("llmsettingsstore: read %s: %w", s.path, err)
	}
	if len(raw) == 0 {
		return nil
	}
	var disk map[int64]llmDTO
	if err := json.Unmarshal(raw, &disk); err != nil {
		return fmt.Errorf("llmsettingsstore: unmarshal: %w", err)
	}
	if disk != nil {
		s.data = disk
	}
	return nil
}

// Get returns the settings for chatID. (nil, false, nil) when absent.
func (s *FileStore) Get(_ context.Context, chatID int64) (*domain.LLMSettings, bool, error) {
	s.mu.RLock()
	dto, ok := s.data[chatID]
	s.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	cfg, err := domain.NewLLMSettings(dto.Provider, dto.BaseURL, dto.APIKey, dto.Model)
	if err != nil {
		return nil, false, fmt.Errorf("llmsettingsstore: rebuild settings: %w", err)
	}
	return cfg, true, nil
}

// Set writes the settings and flushes the whole file atomically.
func (s *FileStore) Set(_ context.Context, chatID int64, cfg *domain.LLMSettings) error {
	if cfg == nil {
		return errors.New("llmsettingsstore: nil cfg")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[chatID] = llmDTO{
		Provider: cfg.Provider(),
		BaseURL:  cfg.BaseURL(),
		APIKey:   cfg.APIKey(),
		Model:    cfg.Model(),
	}
	return s.persistLocked()
}

// Clear removes the settings for chatID; missing entries are a no-op.
func (s *FileStore) Clear(_ context.Context, chatID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.data[chatID]; !ok {
		return nil
	}
	delete(s.data, chatID)
	return s.persistLocked()
}

// persistLocked rewrites the file atomically. Caller must hold s.mu.
func (s *FileStore) persistLocked() error {
	raw, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("llmsettingsstore: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".llm-settings-*.json")
	if err != nil {
		return fmt.Errorf("llmsettingsstore: tmp create: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("llmsettingsstore: tmp write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("llmsettingsstore: tmp close: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("llmsettingsstore: rename: %w", err)
	}
	return nil
}
