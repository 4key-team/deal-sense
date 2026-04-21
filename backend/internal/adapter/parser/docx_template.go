package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
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

	// Normalize split placeholder runs before go-docx processes the file.
	template = normalizeDocxRuns(template)

	// go-docx requires a file path — write to temp file.
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

// normalizeDocxRuns opens a DOCX (ZIP), merges split placeholder runs
// in word/document.xml, and returns the modified DOCX bytes.
func normalizeDocxRuns(data []byte) []byte {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return data
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return data
		}
		content, _ := io.ReadAll(rc)
		rc.Close()

		if f.Name == "word/document.xml" {
			content = []byte(mergePlaceholderRuns(string(content)))
		}

		fw, _ := w.Create(f.Name)
		fw.Write(content)
	}

	w.Close()
	return buf.Bytes()
}
