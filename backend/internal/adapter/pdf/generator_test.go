package pdf_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/pdf"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestMarotoPDFGenerator_Generate(t *testing.T) {
	g := pdf.NewMarotoPDFGenerator()

	tests := []struct {
		name    string
		input   usecase.ContentInput
		wantErr bool
	}{
		{
			name: "three sections",
			input: usecase.ContentInput{
				Meta: map[string]string{
					"client":   "Acme Corp",
					"project":  "Portal Redesign",
					"date":     "24.04.2026",
					"price":    "1 000 000 RUB",
					"timeline": "3 months",
				},
				Sections: []usecase.ContentSection{
					{Title: "About Us", Content: "We are a top engineering company specializing in web development."},
					{Title: "Solution", Content: "We propose a React + Go architecture with CI/CD pipeline."},
					{Title: "Pricing", Content: "Total: 1M RUB over 3 months."},
				},
				Summary: "Commercial proposal for Portal Redesign project.",
			},
		},
		{
			name: "empty sections",
			input: usecase.ContentInput{
				Meta:    map[string]string{"client": "Acme"},
				Summary: "Empty proposal",
			},
		},
		{
			name: "russian text",
			input: usecase.ContentInput{
				Meta: map[string]string{
					"client":  "ООО Рога и Копыта",
					"project": "Редизайн портала",
				},
				Sections: []usecase.ContentSection{
					{Title: "О компании", Content: "Мы — ведущая компания в сфере веб-разработки."},
					{Title: "Решение", Content: "Предлагаем архитектуру React + Go с CI/CD."},
				},
				Summary: "Коммерческое предложение для редизайна портала.",
			},
		},
		{
			name: "nil meta",
			input: usecase.ContentInput{
				Sections: []usecase.ContentSection{
					{Title: "Section", Content: "Content"},
				},
				Summary: "Test",
			},
		},
		{
			name: "multiline content with bullet lists",
			input: usecase.ContentInput{
				Meta: map[string]string{"client": "Test", "project": "Bullets"},
				Sections: []usecase.ContentSection{
					{Title: "Features", Content: "Our features:\n- Fast delivery\n- Quality control\n- 24/7 support\n- Custom integrations"},
					{Title: "Details", Content: "Line one.\nLine two.\nLine three.\nLine four.\nLine five."},
				},
				Summary: "Test multiline",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := g.Generate(context.Background(), tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) == 0 {
				t.Fatal("expected non-empty PDF bytes")
			}
			// PDF files start with %PDF
			if len(result) < 4 || string(result[:4]) != "%PDF" {
				t.Errorf("output does not start with %%PDF, got: %q", string(result[:min(20, len(result))]))
			}
		})
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"### 2.1. Роли систем", "2.1. Роли систем"},
		{"## Заголовок", "Заголовок"},
		{"# Главный", "Главный"},
		{"**bold text**", "bold text"},
		{"Текст **с bold** внутри", "Текст с bold внутри"},
		{"*italic text*", "italic text"},
		{"Обычный текст", "Обычный текст"},
		{"|---------|----------|", ""},
		{"| Позиция | Стоимость |", "Позиция — Стоимость"},
		{"| Bitrix24 | 13 990 ₽ |", "Bitrix24 — 13 990 ₽"},
		{"- Пункт списка", "- Пункт списка"},
		{"* Пункт списка", "* Пункт списка"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pdf.StripMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("StripMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMarotoPDFGenerator_MarkdownContent(t *testing.T) {
	g := pdf.NewMarotoPDFGenerator()

	input := usecase.ContentInput{
		Meta: map[string]string{"client": "Test"},
		Sections: []usecase.ContentSection{
			{
				Title: "Features",
				Content: "### 2.1. Роли систем\n" +
					"**1С УТ** – источник истины.\n" +
					"Обычный текст без разметки.\n" +
					"| Позиция | Стоимость |\n" +
					"|---------|----------|\n" +
					"| Bitrix24 | 13 990 ₽ |\n" +
					"- Первый пункт\n" +
					"* Второй пункт",
			},
		},
		Summary: "Test markdown rendering",
	}

	result, err := g.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) < 4 || string(result[:4]) != "%PDF" {
		t.Error("output is not a valid PDF")
	}
}

func TestMarotoPDFGenerator_MultilineUsesMultipleRows(t *testing.T) {
	g := pdf.NewMarotoPDFGenerator()

	var lines strings.Builder
	for i := range 60 {
		fmt.Fprintf(&lines, "Line number %d with enough text to take space.\n", i+1)
	}
	lines.WriteString("- Bullet item one\n- Bullet item two\n- Bullet item three\n")

	input := usecase.ContentInput{
		Meta:     map[string]string{"client": "Test"},
		Sections: []usecase.ContentSection{{Title: "Long Section", Content: lines.String()}},
	}

	result, err := g.Generate(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pageCount := strings.Count(string(result), "/Type /Page")
	// /Type /Pages is also counted, subtract 1
	if pageCount > 1 {
		pageCount--
	}
	if pageCount < 2 {
		t.Errorf("60+ lines should produce at least 2 pages, got %d", pageCount)
	}
}
