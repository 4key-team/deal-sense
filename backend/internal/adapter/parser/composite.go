package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// Composite routes parsing to the correct parser based on file extension.
type Composite struct {
	parsers []usecase.DocumentParser
}

func NewComposite(parsers ...usecase.DocumentParser) *Composite {
	return &Composite{parsers: parsers}
}

func (c *Composite) Supports(ft domain.FileType) bool {
	for _, p := range c.parsers {
		if p.Supports(ft) {
			return true
		}
	}
	return false
}

func (c *Composite) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	ext := strings.TrimPrefix(filepath.Ext(filename), ".")
	ft, err := domain.ParseFileType(ext)
	if err != nil {
		return "", fmt.Errorf("composite parser: %w", err)
	}

	for _, p := range c.parsers {
		if p.Supports(ft) {
			return p.Parse(ctx, filename, data)
		}
	}

	return "", fmt.Errorf("composite parser: no parser for %s", ext)
}
