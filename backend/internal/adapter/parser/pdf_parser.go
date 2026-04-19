package parser

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/ledongthuc/pdf"
)

// PDFParser extracts text content from PDF files.
type PDFParser struct{}

func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

func (p *PDFParser) Supports(ft domain.FileType) bool {
	return ft == domain.FileTypePDF
}

func (p *PDFParser) Parse(_ context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("parse %s: %w", filename, domain.ErrEmptyContent)
	}

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", filename, err)
	}

	var sb strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		text, _ := reader.Page(i).GetPlainText(nil)
		sb.WriteString(text)
	}

	return strings.TrimSpace(sb.String()), nil
}
