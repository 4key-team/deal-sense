package parser

import (
	"context"
	"fmt"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// DocParser handles legacy Word 97-2003 (.doc) binaries by converting
// them to .docx via the injected DocConverter and then reusing the
// existing DOCX text extractor.
type DocParser struct {
	converter usecase.DocConverter
	reader    *DocxReader
}

func NewDocParser(converter usecase.DocConverter) *DocParser {
	return &DocParser{converter: converter, reader: NewDocxReader()}
}

func (p *DocParser) Supports(ft domain.FileType) bool {
	return ft == domain.FileTypeDOC
}

func (p *DocParser) Parse(ctx context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("parse %s: %w", filename, domain.ErrEmptyContent)
	}
	docx, err := p.converter.ConvertToDOCX(ctx, data)
	if err != nil {
		return "", fmt.Errorf("parse %s: convert .doc: %w", filename, err)
	}
	return p.reader.Parse(ctx, filename, docx)
}

var _ usecase.DocumentParser = (*DocParser)(nil)
