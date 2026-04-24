package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// DocxGenerative fills DOCX templates in generative mode — replacing paragraph content
// by matching section titles to headings, and appending unmatched sections at the end.
type DocxGenerative struct{}

func NewDocxGenerative() *DocxGenerative {
	return &DocxGenerative{}
}

func (g *DocxGenerative) GenerativeFill(_ context.Context, template []byte, sections []usecase.ContentSection) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("generative fill: %w", domain.ErrEmptyTemplate)
	}

	r, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, fmt.Errorf("generative fill: %w", err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		content := mustReadZipEntry(f)

		if f.Name == "word/document.xml" {
			content = g.injectSections(string(content), sections)
		}

		header := f.FileHeader
		header.UncompressedSize64 = uint64(len(content))
		fw, _ := w.CreateHeader(&header)
		fw.Write(content)
	}

	w.Close()
	return buf.Bytes(), nil
}

// injectSections finds headings in the document XML and replaces following paragraphs
// with section content. Unmatched sections are appended before </w:body>.
func (g *DocxGenerative) injectSections(xml string, sections []usecase.ContentSection) []byte {
	if len(sections) == 0 {
		return []byte(xml)
	}

	// Build a map for quick lookup (case-insensitive title match).
	remaining := make(map[string]usecase.ContentSection, len(sections))
	for _, s := range sections {
		remaining[strings.ToLower(s.Title)] = s
	}

	// Simple approach: scan for heading paragraphs, replace the next body paragraph.
	result := xml
	for _, sec := range sections {
		titleLower := strings.ToLower(sec.Title)
		// Find a paragraph containing the title text.
		titleIdx := caseInsensitiveIndex(result, sec.Title)
		if titleIdx < 0 {
			continue
		}
		delete(remaining, titleLower)

		// Find the next <w:p> after the title's </w:p>.
		endOfTitleP := strings.Index(result[titleIdx:], "</w:p>")
		if endOfTitleP < 0 {
			continue
		}
		afterTitle := titleIdx + endOfTitleP + len("</w:p>")

		// Find the next paragraph to replace.
		nextPStart := strings.Index(result[afterTitle:], "<w:p")
		if nextPStart < 0 {
			continue
		}
		nextPStart += afterTitle
		nextPEnd := strings.Index(result[nextPStart:], "</w:p>")
		if nextPEnd < 0 {
			continue
		}
		nextPEnd += nextPStart + len("</w:p>")

		// Build replacement paragraph with generated content.
		replacement := buildParagraph(sec.Content)
		result = result[:nextPStart] + replacement + result[nextPEnd:]
	}

	// Append unmatched sections before </w:body>.
	if len(remaining) > 0 {
		var appended strings.Builder
		for _, sec := range sections {
			if _, ok := remaining[strings.ToLower(sec.Title)]; !ok {
				continue
			}
			appended.WriteString(buildHeadingParagraph(sec.Title))
			appended.WriteString(buildParagraph(sec.Content))
		}

		bodyEnd := strings.LastIndex(result, "</w:body>")
		if bodyEnd >= 0 {
			result = result[:bodyEnd] + appended.String() + result[bodyEnd:]
		}
	}

	return []byte(result)
}

func caseInsensitiveIndex(s, substr string) int {
	return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}

func buildParagraph(content string) string {
	return `<w:p><w:r><w:t xml:space="preserve">` + escapeXML(content) + `</w:t></w:r></w:p>`
}

func buildHeadingParagraph(title string) string {
	return `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>` +
		escapeXML(title) + `</w:t></w:r></w:p>`
}

// mustReadZipEntry is reused from docx_template.go (same package).
// escapeXML is reused from docx_template.go (same package).

// Ensure DocxGenerative implements GenerativeEngine at compile time.
var _ usecase.GenerativeEngine = (*DocxGenerative)(nil)

