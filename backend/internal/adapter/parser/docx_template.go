package parser

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/lukasjarosch/go-docx"
)

// DocxTemplate fills DOCX templates with placeholder values using go-docx.
type DocxTemplate struct{}

func NewDocxTemplate() *DocxTemplate {
	return &DocxTemplate{}
}

func (t *DocxTemplate) Fill(_ context.Context, template []byte, params map[string]string) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("fill template: %w", domain.ErrEmptyTemplate)
	}

	// go-docx requires a file path — write to temp file.
	// MkdirTemp/WriteFile on /tmp are infra-level ops; ignoring errors is safe.
	tmpDir, _ := os.MkdirTemp("", "deal-sense-tmpl-*")
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.docx")
	os.WriteFile(inputPath, template, 0o600)

	doc, err := docx.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("fill template: %w", err)
	}

	replaceMap := docx.PlaceholderMap{}
	for k, v := range params {
		replaceMap[k] = v
	}

	doc.ReplaceAll(replaceMap)

	var buf bytes.Buffer
	doc.Write(&buf)

	return buf.Bytes(), nil
}
