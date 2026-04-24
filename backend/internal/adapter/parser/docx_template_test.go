package parser_test

import (
	"archive/zip"
	"bytes"
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

	t.Run("preserves zip entry metadata", func(t *testing.T) {
		data, err := os.ReadFile("testdata/template.docx")
		if err != nil {
			t.Fatalf("read testdata: %v", err)
		}

		result, err := tmpl.Fill(t.Context(), data, map[string]string{})
		if err != nil {
			t.Fatalf("Fill() error: %v", err)
		}

		r, err := zip.NewReader(bytes.NewReader(result), int64(len(result)))
		if err != nil {
			t.Fatalf("output is not a valid ZIP: %v", err)
		}

		origR, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		origByName := make(map[string]*zip.File)
		for _, f := range origR.File {
			origByName[f.Name] = f
		}

		if len(r.File) != len(origR.File) {
			t.Errorf("output has %d entries, original has %d", len(r.File), len(origR.File))
		}

		for _, f := range r.File {
			orig, ok := origByName[f.Name]
			if !ok {
				t.Errorf("output contains unexpected entry: %s", f.Name)
				continue
			}
			if f.Method != orig.Method {
				t.Errorf("entry %s: compression method = %d, want %d", f.Name, f.Method, orig.Method)
			}
		}
	})
}
