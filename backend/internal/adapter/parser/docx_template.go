package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// DocxTemplate fills DOCX templates with placeholder values.
// Placeholders use {{key}} syntax in the document text.
type DocxTemplate struct{}

func NewDocxTemplate() *DocxTemplate {
	return &DocxTemplate{}
}

func (t *DocxTemplate) Fill(_ context.Context, template []byte, params map[string]string) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("fill template: %w", domain.ErrEmptyTemplate)
	}

	r, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, fmt.Errorf("fill template: %w", err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("fill template: open %s: %w", f.Name, err)
		}
		content, _ := io.ReadAll(rc)
		rc.Close()

		// Replace placeholders in document XML files.
		if isDocxXML(f.Name) {
			xml := mergePlaceholderRuns(string(content))
			for k, v := range params {
				xml = strings.ReplaceAll(xml, "{{"+k+"}}", escapeXML(v))
			}
			content = []byte(xml)
		}

		fw, _ := w.Create(f.Name)
		fw.Write(content)
	}

	w.Close()
	return buf.Bytes(), nil
}

func isDocxXML(name string) bool {
	return name == "word/document.xml" ||
		name == "word/header1.xml" ||
		name == "word/header2.xml" ||
		name == "word/header3.xml" ||
		name == "word/footer1.xml" ||
		name == "word/footer2.xml" ||
		name == "word/footer3.xml"
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
