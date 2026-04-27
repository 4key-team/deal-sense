package parser_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func makeDocxWithBody(bodyXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("word/document.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + bodyXML + `</w:body></w:document>`))
	// Add minimal content types
	ct, _ := w.Create("[Content_Types].xml")
	ct.Write([]byte(`<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`</Types>`))
	w.Close()
	return buf.Bytes()
}

func readDocxXML(data []byte) string {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			return string(b)
		}
	}
	return ""
}

func TestDocxGenerative_GenerativeFill(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		body     string
		sections []usecase.ContentSection
		contains []string
		wantErr  bool
	}{
		{
			name: "replaces paragraph content with section text",
			body: `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>About Us</w:t></w:r></w:p>` +
				`<w:p><w:r><w:t>Old text here</w:t></w:r></w:p>`,
			sections: []usecase.ContentSection{
				{Title: "About Us", Content: "We are the best company in the world."},
			},
			contains: []string{"We are the best company in the world."},
		},
		{
			name: "preserves non-matching paragraphs",
			body: `<w:p><w:r><w:t>Keep this text</w:t></w:r></w:p>`,
			sections: []usecase.ContentSection{
				{Title: "Missing Section", Content: "This should be appended"},
			},
			contains: []string{"Keep this text", "This should be appended"},
		},
		{
			name:     "empty sections — no error",
			body:     `<w:p><w:r><w:t>Original</w:t></w:r></w:p>`,
			sections: nil,
			contains: []string{"Original"},
		},
		{
			name: "multiline content splits into separate paragraphs",
			body: `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Scope</w:t></w:r></w:p>` +
				`<w:p><w:r><w:t>Old scope</w:t></w:r></w:p>`,
			sections: []usecase.ContentSection{
				{Title: "Scope", Content: "Line one.\nLine two.\nLine three."},
			},
			contains: []string{
				"Line one.</w:t></w:r></w:p>",
				"Line two.</w:t></w:r></w:p>",
				"Line three.</w:t></w:r></w:p>",
			},
		},
		{
			name: "bullet list items get ListBullet style",
			body: `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Features</w:t></w:r></w:p>` +
				`<w:p><w:r><w:t>Old features</w:t></w:r></w:p>`,
			sections: []usecase.ContentSection{
				{Title: "Features", Content: "Our features:\n- Fast delivery\n- Quality control\n- 24/7 support"},
			},
			contains: []string{
				"Our features:",
				"Fast delivery", "Quality control", "24/7 support",
				`w:val="ListBullet"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docx := makeDocxWithBody(tt.body)
			g := parser.NewDocxGenerative()
			result, err := g.GenerativeFill(ctx, docx, tt.sections)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			xml := readDocxXML(result)
			for _, s := range tt.contains {
				if !strings.Contains(xml, s) {
					t.Errorf("output XML missing %q\ngot: %s", s, xml)
				}
			}
		})
	}
}

func TestDocxGenerative_GenerativeFill_SplitRuns(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		body        string
		sections    []usecase.ContentSection
		contains    []string
		notContains []string
	}{
		{
			name: "heading split across runs still matches section",
			body: `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` +
				`<w:r><w:rPr><w:b/></w:rPr><w:t>О </w:t></w:r>` +
				`<w:r><w:rPr><w:b/></w:rPr><w:t>компании</w:t></w:r></w:p>` +
				`<w:p><w:r><w:t>Old description</w:t></w:r></w:p>`,
			sections: []usecase.ContentSection{
				{Title: "О компании", Content: "Мы лучшая компания."},
			},
			contains: []string{"Мы лучшая компания."},
			notContains: []string{
				`w:val="Heading1"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>О компании`,
			},
		},
		{
			name: "heading split into three runs matches",
			body: `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr>` +
				`<w:r><w:t>Наши </w:t></w:r>` +
				`<w:r><w:t>услу</w:t></w:r>` +
				`<w:r><w:t>ги</w:t></w:r></w:p>` +
				`<w:p><w:r><w:t>Placeholder</w:t></w:r></w:p>`,
			sections: []usecase.ContentSection{
				{Title: "Наши услуги", Content: "Список услуг."},
			},
			contains:    []string{"Список услуг."},
			notContains: []string{"Placeholder"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docx := makeDocxWithBody(tt.body)
			g := parser.NewDocxGenerative()
			result, err := g.GenerativeFill(ctx, docx, tt.sections)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			xml := readDocxXML(result)
			for _, s := range tt.contains {
				if !strings.Contains(xml, s) {
					t.Errorf("output XML missing %q\ngot: %s", s, xml)
				}
			}
			for _, s := range tt.notContains {
				if strings.Contains(xml, s) {
					t.Errorf("output XML should NOT contain %q\ngot: %s", s, xml)
				}
			}
		})
	}
}

func TestDocxGenerative_GenerativeFill_AppendBeforeSectPr(t *testing.T) {
	ctx := context.Background()

	body := `<w:p><w:r><w:t>Cover page</w:t></w:r></w:p>` +
		`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/></w:sectPr>`

	docx := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, docx, []usecase.ContentSection{
		{Title: "Введение", Content: "Текст введения."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxXML(result)

	// Content must appear BEFORE <w:sectPr>, not after it.
	sectPrIdx := strings.Index(xml, "<w:sectPr")
	contentIdx := strings.Index(xml, "Текст введения.")
	if contentIdx < 0 {
		t.Fatal("generated content not found in output")
	}
	if sectPrIdx >= 0 && contentIdx > sectPrIdx {
		t.Errorf("content inserted AFTER <w:sectPr> — must be before it\ncontent at %d, sectPr at %d\nxml: %s",
			contentIdx, sectPrIdx, xml)
	}
}

func TestDocxGenerative_GenerativeFill_EmptyTemplate(t *testing.T) {
	g := parser.NewDocxGenerative()
	_, err := g.GenerativeFill(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestDocxGenerative_GenerativeFill_InvalidZip(t *testing.T) {
	g := parser.NewDocxGenerative()
	_, err := g.GenerativeFill(context.Background(), []byte("not a zip"), nil)
	if err == nil {
		t.Fatal("expected error for invalid zip")
	}
}
