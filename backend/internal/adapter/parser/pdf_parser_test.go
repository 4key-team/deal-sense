package parser_test

import (
	"os"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestPDFParser_Supports(t *testing.T) {
	p := parser.NewPDFParser()

	tests := []struct {
		name string
		ft   domain.FileType
		want bool
	}{
		{name: "supports pdf", ft: domain.FileTypePDF, want: true},
		{name: "rejects docx", ft: domain.FileTypeDOCX, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.Supports(tt.ft); got != tt.want {
				t.Errorf("Supports(%v) = %v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestPDFParser_Parse(t *testing.T) {
	p := parser.NewPDFParser()

	t.Run("extracts text from valid PDF", func(t *testing.T) {
		data, err := os.ReadFile("testdata/hello.pdf")
		if err != nil {
			t.Fatalf("read testdata: %v", err)
		}

		text, err := p.Parse(t.Context(), "hello.pdf", data)
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		if text == "" {
			t.Error("Parse() returned empty text")
		}
	})

	t.Run("returns error for empty data", func(t *testing.T) {
		_, err := p.Parse(t.Context(), "empty.pdf", nil)
		if err == nil {
			t.Error("expected error for empty data")
		}
	})

	t.Run("returns error for invalid PDF", func(t *testing.T) {
		_, err := p.Parse(t.Context(), "bad.pdf", []byte("not a pdf"))
		if err == nil {
			t.Error("expected error for invalid PDF")
		}
	})

	t.Run("handles PDF with empty page gracefully", func(t *testing.T) {
		data, err := os.ReadFile("testdata/multipage.pdf")
		if err != nil {
			t.Fatalf("read testdata: %v", err)
		}

		text, err := p.Parse(t.Context(), "multipage.pdf", data)
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		// Should extract text from page 1, skip empty page 2
		if text == "" {
			t.Error("expected some text from multipage PDF")
		}
	})
}
