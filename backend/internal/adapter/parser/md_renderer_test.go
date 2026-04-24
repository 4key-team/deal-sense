package parser_test

import (
	"context"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestMarkdownRenderer_Render(t *testing.T) {
	r := parser.NewMarkdownRenderer()

	tests := []struct {
		name     string
		input    usecase.MDInput
		contains []string
	}{
		{
			name: "full proposal",
			input: usecase.MDInput{
				Meta: map[string]string{
					"client":  "Acme Corp",
					"project": "Portal",
					"price":   "1M RUB",
					"date":    "24.04.2026",
				},
				Sections: []usecase.MDSection{
					{Title: "О компании", Content: "Мы лучшие."},
					{Title: "Решение", Content: "React + Go."},
				},
				Summary: "Коммерческое предложение.",
			},
			contains: []string{
				"# Коммерческое предложение",
				"Acme Corp",
				"Portal",
				"## О компании",
				"Мы лучшие.",
				"## Решение",
				"React + Go.",
			},
		},
		{
			name: "empty sections",
			input: usecase.MDInput{
				Meta:    map[string]string{"client": "Test"},
				Summary: "Empty",
			},
			contains: []string{"# Empty", "Test"},
		},
		{
			name: "nil meta",
			input: usecase.MDInput{
				Sections: []usecase.MDSection{{Title: "Sec", Content: "Text"}},
				Summary:  "Proposal",
			},
			contains: []string{"# Proposal", "## Sec", "Text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Render(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) == 0 {
				t.Fatal("expected non-empty result")
			}
			md := string(result)
			for _, s := range tt.contains {
				if !strings.Contains(md, s) {
					t.Errorf("output missing %q\ngot:\n%s", s, md)
				}
			}
		})
	}
}
