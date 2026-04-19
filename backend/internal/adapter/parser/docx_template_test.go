package parser_test

import (
	"os"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
)

func TestDocxTemplate_Fill(t *testing.T) {
	tmpl := parser.NewDocxTemplate()

	t.Run("fills placeholders in template", func(t *testing.T) {
		data, err := os.ReadFile("testdata/template.docx")
		if err != nil {
			t.Fatalf("read testdata: %v", err)
		}

		params := map[string]string{
			"company_name": "Acme Corp",
			"project_name": "Portal",
		}

		result, err := tmpl.Fill(t.Context(), data, params)
		if err != nil {
			t.Fatalf("Fill() error: %v", err)
		}
		if len(result) == 0 {
			t.Error("Fill() returned empty result")
		}
	})

	t.Run("returns error for empty template", func(t *testing.T) {
		_, err := tmpl.Fill(t.Context(), nil, map[string]string{"a": "b"})
		if err == nil {
			t.Error("expected error for empty template")
		}
	})

	t.Run("returns error for invalid DOCX", func(t *testing.T) {
		_, err := tmpl.Fill(t.Context(), []byte("not a docx"), map[string]string{"a": "b"})
		if err == nil {
			t.Error("expected error for invalid DOCX")
		}
	})
}
