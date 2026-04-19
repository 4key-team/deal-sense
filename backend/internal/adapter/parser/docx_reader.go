package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// DocxReader extracts text content from DOCX files using stdlib zip+xml.
type DocxReader struct{}

func NewDocxReader() *DocxReader {
	return &DocxReader{}
}

func (r *DocxReader) Supports(ft domain.FileType) bool {
	return ft == domain.FileTypeDOCX
}

func (r *DocxReader) Parse(_ context.Context, filename string, data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("parse %s: %w", filename, domain.ErrEmptyContent)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", filename, err)
	}

	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			return extractDocxText(f)
		}
	}

	return "", fmt.Errorf("parse %s: word/document.xml not found in archive", filename)
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"body>p"`
}

type docxParagraph struct {
	Runs []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

func extractDocxText(f *zip.File) (string, error) {
	rc, _ := f.Open() // zip.File.Open on in-memory reader cannot fail
	defer rc.Close()

	var doc docxBody
	if err := xml.NewDecoder(rc).Decode(&doc); err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, p := range doc.Paragraphs {
		if i > 0 {
			sb.WriteByte('\n')
		}
		for _, r := range p.Runs {
			sb.WriteString(r.Text)
		}
	}

	return strings.TrimSpace(sb.String()), nil
}
