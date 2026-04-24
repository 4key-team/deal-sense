package parser

import (
	"context"
	"fmt"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// MDParser reads Markdown files as plain text.
type MDParser struct{}

func NewMDParser() *MDParser {
	return &MDParser{}
}

func (p *MDParser) Parse(_ context.Context, _ string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("md parser: %w", domain.ErrEmptyContent)
	}
	return string(data), nil
}

func (p *MDParser) Supports(ft domain.FileType) bool {
	return ft == domain.FileTypeMD
}
