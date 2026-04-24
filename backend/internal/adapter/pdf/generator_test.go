package pdf_test

import (
	"context"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/pdf"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestMarotoPDFGenerator_Generate(t *testing.T) {
	g := pdf.NewMarotoPDFGenerator()

	tests := []struct {
		name    string
		input   usecase.PDFInput
		wantErr bool
	}{
		{
			name: "three sections",
			input: usecase.PDFInput{
				Meta: map[string]string{
					"client":   "Acme Corp",
					"project":  "Portal Redesign",
					"date":     "24.04.2026",
					"price":    "1 000 000 RUB",
					"timeline": "3 months",
				},
				Sections: []usecase.PDFSection{
					{Title: "About Us", Content: "We are a top engineering company specializing in web development."},
					{Title: "Solution", Content: "We propose a React + Go architecture with CI/CD pipeline."},
					{Title: "Pricing", Content: "Total: 1M RUB over 3 months."},
				},
				Summary: "Commercial proposal for Portal Redesign project.",
			},
		},
		{
			name: "empty sections",
			input: usecase.PDFInput{
				Meta:    map[string]string{"client": "Acme"},
				Summary: "Empty proposal",
			},
		},
		{
			name: "russian text",
			input: usecase.PDFInput{
				Meta: map[string]string{
					"client":  "ООО Рога и Копыта",
					"project": "Редизайн портала",
				},
				Sections: []usecase.PDFSection{
					{Title: "О компании", Content: "Мы — ведущая компания в сфере веб-разработки."},
					{Title: "Решение", Content: "Предлагаем архитектуру React + Go с CI/CD."},
				},
				Summary: "Коммерческое предложение для редизайна портала.",
			},
		},
		{
			name: "nil meta",
			input: usecase.PDFInput{
				Sections: []usecase.PDFSection{
					{Title: "Section", Content: "Content"},
				},
				Summary: "Test",
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
