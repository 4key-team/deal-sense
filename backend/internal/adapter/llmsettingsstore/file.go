// Package llmsettingsstore implements usecase/llmsettings.Repository on top
// of a JSON file. It is the persistence adapter for per-chat Telegram LLM
// provider settings: one file per deployment, single-bot scale.
package llmsettingsstore

import (
	"context"
	"errors"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// FileStore is a sync.RWMutex-guarded map mirrored to a JSON file on disk.
// Writes rewrite the whole file atomically (tmp file + rename).
type FileStore struct{}

// NewFileStore creates a FileStore backed by path. RED stub returns an
// error so the package compiles but the FileStore tests fail until the
// GREEN implementation lands.
func NewFileStore(path string) (*FileStore, error) {
	return nil, errors.New("llmsettingsstore: not implemented")
}

// Get returns the settings for chatID. (nil, false, nil) when absent.
func (s *FileStore) Get(_ context.Context, _ int64) (*domain.LLMSettings, bool, error) {
	return nil, false, errors.New("llmsettingsstore: not implemented")
}

// Set writes the settings and flushes the whole file atomically.
func (s *FileStore) Set(_ context.Context, _ int64, _ *domain.LLMSettings) error {
	return errors.New("llmsettingsstore: not implemented")
}

// Clear removes the settings for chatID; missing entries are a no-op.
func (s *FileStore) Clear(_ context.Context, _ int64) error {
	return errors.New("llmsettingsstore: not implemented")
}
