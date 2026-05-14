// Package profilestore implements usecase/telegram.ProfileStore on top of a
// JSON file. It is the persistence adapter for per-chat Telegram company
// profiles: one file per deployment, single-bot scale.
package profilestore

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
	data map[int64]profileDTO
}

// profileDTO is the on-disk shape; it mirrors domain.CompanyProfile fields
// 1-to-1 so reloads return identical Render() output.
type profileDTO struct {
	Name            string   `json:"name"`
	TeamSize        string   `json:"team_size"`
	Experience      string   `json:"experience"`
	TechStack       []string `json:"tech_stack,omitempty"`
	Certifications  []string `json:"certifications,omitempty"`
	Specializations []string `json:"specializations,omitempty"`
	KeyClients      string   `json:"key_clients"`
	Extra           string   `json:"extra"`
}

// NewFileStore creates a FileStore backed by path. If the file is missing the
// store starts empty; missing parent directories are created.
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		return nil, errors.New("profilestore: path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("profilestore: mkdir parent: %w", err)
	}
	s := &FileStore{path: path, data: map[int64]profileDTO{}}
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
		return fmt.Errorf("profilestore: read %s: %w", s.path, err)
	}
	if len(raw) == 0 {
		return nil
	}
	var disk map[int64]profileDTO
	if err := json.Unmarshal(raw, &disk); err != nil {
		return fmt.Errorf("profilestore: unmarshal: %w", err)
	}
	if disk != nil {
		s.data = disk
	}
	return nil
}

// Get returns the profile for chatID. (nil, false, nil) when absent.
func (s *FileStore) Get(_ context.Context, chatID int64) (*domain.CompanyProfile, bool, error) {
	s.mu.RLock()
	dto, ok := s.data[chatID]
	s.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	p, err := domain.NewCompanyProfile(
		dto.Name, dto.TeamSize, dto.Experience,
		dto.TechStack, dto.Certifications, dto.Specializations,
		dto.KeyClients, dto.Extra,
	)
	if err != nil {
		return nil, false, fmt.Errorf("profilestore: rebuild profile: %w", err)
	}
	return p, true, nil
}

// Set writes the profile and flushes the whole file.
func (s *FileStore) Set(_ context.Context, chatID int64, p *domain.CompanyProfile) error {
	if p == nil {
		return errors.New("profilestore: nil profile")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[chatID] = profileDTO{
		Name:            p.Name(),
		TeamSize:        p.TeamSize(),
		Experience:      p.Experience(),
		TechStack:       p.TechStack(),
		Certifications:  p.Certifications(),
		Specializations: p.Specializations(),
		KeyClients:      p.KeyClients(),
		Extra:           p.Extra(),
	}
	return s.persistLocked()
}

// Clear removes the profile for chatID; missing entries are a no-op.
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
		return fmt.Errorf("profilestore: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".profiles-*.json")
	if err != nil {
		return fmt.Errorf("profilestore: tmp create: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("profilestore: tmp write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("profilestore: tmp close: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("profilestore: rename: %w", err)
	}
	return nil
}
