package parser_test

import (
	"bytes"
	"context"
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
			name:       "empty sections — no modification",
			paragraphs: []struct{ text, style string }{{"Original", ""}},
			sections:   nil,
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
