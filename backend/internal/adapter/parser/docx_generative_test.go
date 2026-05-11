package parser_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	docx "github.com/mmonterroca/docxgo/v2"
	"github.com/mmonterroca/docxgo/v2/domain"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func makeDocxFixture(paragraphs []struct{ text, style string }) []byte {
	doc := docx.NewDocument()
	for _, p := range paragraphs {
		para, _ := doc.AddParagraph()
		if p.style != "" {
			para.SetStyle(p.style)
		}
		run, _ := para.AddRun()
		run.SetText(p.text)
	}
	var buf bytes.Buffer
	doc.WriteTo(&buf)
	return buf.Bytes()
}

func docxParagraphTexts(data []byte) []string {
	doc, err := docx.OpenDocumentFromBytes(data)
	if err != nil {
		return nil
	}
	var texts []string
	for _, p := range doc.Paragraphs() {
		texts = append(texts, p.Text())
	}
	return texts
}

func TestDocxGenerative_GenerativeFill(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		paragraphs     []struct{ text, style string }
		sections       []usecase.ContentSection
		wantContains   []string
		wantNotContain []string
		wantErr        bool
	}{
		{
			name: "replaces content after matched heading",
			paragraphs: []struct{ text, style string }{
				{"About Us", domain.StyleIDHeading1},
				{"Old content", ""},
			},
			sections: []usecase.ContentSection{
				{Title: "About Us", Content: "We are the best company."},
			},
			wantContains:   []string{"We are the best company."},
			wantNotContain: []string{"Old content"},
		},
		{
			name: "case-insensitive heading match",
			paragraphs: []struct{ text, style string }{
				{"Our Services", domain.StyleIDHeading1},
				{"placeholder", ""},
			},
			sections: []usecase.ContentSection{
				{Title: "our services", Content: "We provide consulting."},
			},
			wantContains:   []string{"We provide consulting."},
			wantNotContain: []string{"placeholder"},
		},
		{
			name: "appends unmatched sections at end",
			paragraphs: []struct{ text, style string }{
				{"Cover Page", ""},
			},
			sections: []usecase.ContentSection{
				{Title: "New Section", Content: "Appended content."},
			},
			wantContains: []string{"Cover Page", "New Section", "Appended content."},
		},
		{
			name: "multiline content",
			paragraphs: []struct{ text, style string }{
				{"Scope", domain.StyleIDHeading1},
				{"Old scope", ""},
			},
			sections: []usecase.ContentSection{
				{Title: "Scope", Content: "Line one.\nLine two.\nLine three."},
			},
			wantContains:   []string{"Line one.", "Line two.", "Line three."},
			wantNotContain: []string{"Old scope"},
		},
		{
			name: "bullet list items",
			paragraphs: []struct{ text, style string }{
				{"Features", domain.StyleIDHeading1},
				{"Old features", ""},
			},
			sections: []usecase.ContentSection{
				{Title: "Features", Content: "Our features:\n- Fast delivery\n- Quality control"},
			},
			wantContains:   []string{"Our features:", "Fast delivery", "Quality control"},
			wantNotContain: []string{"Old features"},
		},
		{
			name: "strips markdown from content",
			paragraphs: []struct{ text, style string }{
				{"Details", domain.StyleIDHeading1},
				{"Old details", ""},
			},
			sections: []usecase.ContentSection{
				{Title: "Details", Content: "**1С УТ**: мастер-система.\n### Подзаголовок\n| Позиция | Цена |\n|---|---|\n| Bitrix | 13 990 ₽ |\n[ваш email](mailto:x)"},
			},
			wantContains:   []string{"1С УТ", "Подзаголовок", "Bitrix", "13 990 ₽", "ваш email"},
			wantNotContain: []string{"**", "###", "|---|", "[ваш email](mailto:x)"},
		},
		{
			name:         "empty sections — no modification",
			paragraphs:   []struct{ text, style string }{{"Original", ""}},
			sections:     nil,
			wantContains: []string{"Original"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template := makeDocxFixture(tt.paragraphs)
			g := parser.NewDocxGenerative()
			result, err := g.GenerativeFill(ctx, template, tt.sections)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			texts := docxParagraphTexts(result)
			allText := strings.Join(texts, "\n")

			for _, want := range tt.wantContains {
				if !strings.Contains(allText, want) {
					t.Errorf("output missing %q\nparagraphs: %v", want, texts)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(allText, notWant) {
					t.Errorf("output should NOT contain %q\nparagraphs: %v", notWant, texts)
				}
			}
		})
	}
}

func TestDocxGenerative_GenerativeFill_EmptyTemplate(t *testing.T) {
	g := parser.NewDocxGenerative()
	_, err := g.GenerativeFill(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestDocxGenerative_GenerativeFill_InvalidData(t *testing.T) {
	g := parser.NewDocxGenerative()
	_, err := g.GenerativeFill(context.Background(), []byte("not a zip"), nil)
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
}

func TestDocxGenerative_GenerateClean(t *testing.T) {
	ctx := context.Background()
	g := parser.NewDocxGenerative()

	input := usecase.ContentInput{
		Meta:    map[string]string{"client": "Acme Corp", "project": "Portal"},
		Summary: "Commercial proposal for Acme Corp portal development.",
		Sections: []usecase.ContentSection{
			{Title: "Introduction", Content: "We propose building a modern portal."},
			{Title: "Timeline", Content: "- Phase 1: Design\n- Phase 2: Development\n- Phase 3: Launch"},
		},
	}

	result, err := g.GenerateClean(ctx, input)
	if err != nil {
		t.Fatalf("GenerateClean() error: %v", err)
	}

	// Verify it's a valid DOCX
	doc, err := docx.OpenDocumentFromBytes(result)
	if err != nil {
		t.Fatalf("result is not valid DOCX: %v", err)
	}

	texts := make([]string, 0)
	for _, p := range doc.Paragraphs() {
		texts = append(texts, p.Text())
	}
	allText := strings.Join(texts, "\n")

	for _, want := range []string{
		"Commercial proposal for Acme Corp portal development.",
		"Introduction",
		"We propose building a modern portal.",
		"Timeline",
		"Phase 1: Design",
		"Phase 2: Development",
		"Phase 3: Launch",
	} {
		if !strings.Contains(allText, want) {
			t.Errorf("output missing %q\nparagraphs: %v", want, texts)
		}
	}
}

// makeDocxWithBody creates a raw DOCX via zip that docxgo cannot open
// (missing [Content_Types].xml), triggering the zip-based fallback path.
func makeDocxWithBody(bodyXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("word/document.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + bodyXML + `</w:body></w:document>`))
	w.Close()
	return buf.Bytes()
}

func makeDocxWithStyles(bodyXML, stylesXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("word/document.xml")
	f.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:body>` + bodyXML + `</w:body></w:document>`))
	s, _ := w.Create("word/styles.xml")
	s.Write([]byte(stylesXML))
	w.Close()
	return buf.Bytes()
}

func readDocxEntry(data []byte, entryName string) string {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}
	for _, f := range r.File {
		if f.Name == entryName {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			return string(b)
		}
	}
	return ""
}

func TestDocxGenerative_ZipFallback_ReplacesContent(t *testing.T) {
	ctx := context.Background()
	body := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>About Us</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Old text here</w:t></w:r></w:p>`

	template := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "About Us", Content: "We are the best company."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxEntry(result, "word/document.xml")
	if !strings.Contains(xml, "We are the best company.") {
		t.Errorf("output missing replaced content\ngot: %s", xml)
	}
	if strings.Contains(xml, "Old text here") {
		t.Errorf("output still contains old content\ngot: %s", xml)
	}
}

func TestDocxGenerative_ZipFallback_AppendsUnmatched(t *testing.T) {
	ctx := context.Background()
	body := `<w:p><w:r><w:t>Cover page</w:t></w:r></w:p>`

	template := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "New Section", Content: "Appended content."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxEntry(result, "word/document.xml")
	if !strings.Contains(xml, "Cover page") {
		t.Errorf("original content lost")
	}
	if !strings.Contains(xml, "New Section") {
		t.Errorf("appended heading missing")
	}
	if !strings.Contains(xml, "Appended content.") {
		t.Errorf("appended content missing")
	}
}

func TestDocxGenerative_ZipFallback_BulletList(t *testing.T) {
	ctx := context.Background()
	body := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Features</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Old</w:t></w:r></w:p>`

	template := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "Features", Content: "Items:\n- First\n- Second"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxEntry(result, "word/document.xml")
	if !strings.Contains(xml, `w:val="ListBullet"`) {
		t.Errorf("ListBullet style missing\ngot: %s", xml)
	}
	if !strings.Contains(xml, "First") || !strings.Contains(xml, "Second") {
		t.Errorf("bullet items missing\ngot: %s", xml)
	}
}

func TestDocxGenerative_ZipFallback_AppendBeforeSectPr(t *testing.T) {
	ctx := context.Background()
	body := `<w:p><w:r><w:t>Cover</w:t></w:r></w:p>` +
		`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/></w:sectPr>`

	template := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "Intro", Content: "Text."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxEntry(result, "word/document.xml")
	sectPrIdx := strings.Index(xml, "<w:sectPr")
	contentIdx := strings.Index(xml, "Text.")
	if contentIdx < 0 || (sectPrIdx >= 0 && contentIdx > sectPrIdx) {
		t.Errorf("content must be before <w:sectPr>")
	}
}

func TestDocxGenerative_ZipFallback_InjectsListBulletStyle(t *testing.T) {
	ctx := context.Background()
	stylesXML := `<?xml version="1.0"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
		`<w:style w:type="paragraph" w:styleId="Normal"><w:name w:val="normal"/></w:style></w:styles>`
	body := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Items</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Old</w:t></w:r></w:p>`

	template := makeDocxWithStyles(body, stylesXML)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "Items", Content: "- One\n- Two"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	styles := readDocxEntry(result, "word/styles.xml")
	if !strings.Contains(styles, `w:styleId="ListBullet"`) {
		t.Errorf("ListBullet style not injected\ngot: %s", styles)
	}
}

func TestDocxGenerative_ZipFallback_InsertsBeforeFooter(t *testing.T) {
	ctx := context.Background()
	body := `<w:p><w:r><w:t>Уважаемые коллеги!</w:t></w:r></w:p>` +
		`<w:p></w:p><w:p></w:p>` +
		`<w:p><w:r><w:t>С уважением, Директор</w:t></w:r></w:p>` +
		`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/></w:sectPr>`

	template := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "Введение", Content: "Текст введения."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxEntry(result, "word/document.xml")
	contentIdx := strings.Index(xml, "Текст введения.")
	footerIdx := strings.Index(xml, "С уважением")
	if contentIdx < 0 {
		t.Fatal("content not found")
	}
	if footerIdx < 0 {
		t.Fatal("footer not found")
	}
	if contentIdx > footerIdx {
		t.Errorf("content (at %d) must be BEFORE footer (at %d)", contentIdx, footerIdx)
	}
}

func TestDocxGenerative_ZipFallback_StripsMarkdown(t *testing.T) {
	ctx := context.Background()
	body := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Details</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>Old</w:t></w:r></w:p>`

	template := makeDocxWithBody(body)
	g := parser.NewDocxGenerative()
	result, err := g.GenerativeFill(ctx, template, []usecase.ContentSection{
		{Title: "Details", Content: "**Bold text** here.\n### Heading\n| A | B |\n|---|---|\n| C | D |"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	xml := readDocxEntry(result, "word/document.xml")
	if strings.Contains(xml, "**") {
		t.Errorf("output contains ** markdown markers\ngot: %s", xml)
	}
	if strings.Contains(xml, "###") {
		t.Errorf("output contains ### markdown markers\ngot: %s", xml)
	}
	if !strings.Contains(xml, "Bold text") {
		t.Errorf("output missing cleaned text 'Bold text'\ngot: %s", xml)
	}
	if !strings.Contains(xml, "Heading") {
		t.Errorf("output missing cleaned heading text\ngot: %s", xml)
	}
}

func TestDocxGenerative_GenerateClean_StripsMarkdown(t *testing.T) {
	ctx := context.Background()
	g := parser.NewDocxGenerative()

	input := usecase.ContentInput{
		Summary: "Test",
		Sections: []usecase.ContentSection{
			{Title: "Info", Content: "**Bold** and *italic* and [link](http://x)"},
		},
	}

	result, err := g.GenerateClean(ctx, input)
	if err != nil {
		t.Fatalf("GenerateClean() error: %v", err)
	}

	texts := docxParagraphTexts(result)
	allText := strings.Join(texts, "\n")

	if strings.Contains(allText, "**") {
		t.Errorf("output contains ** markdown markers\nparagraphs: %v", texts)
	}
	if strings.Contains(allText, "[link](http://x)") {
		t.Errorf("output contains raw markdown link\nparagraphs: %v", texts)
	}
	if !strings.Contains(allText, "Bold") {
		t.Errorf("output missing cleaned text\nparagraphs: %v", texts)
	}
}

func TestDocxGenerative_GenerateClean_EmptyInput(t *testing.T) {
	ctx := context.Background()
	g := parser.NewDocxGenerative()

	result, err := g.GenerateClean(ctx, usecase.ContentInput{})
	if err != nil {
		t.Fatalf("GenerateClean() error: %v", err)
	}

	// Should still be a valid DOCX
	_, err = docx.OpenDocumentFromBytes(result)
	if err != nil {
		t.Fatalf("result is not valid DOCX: %v", err)
	}
}
