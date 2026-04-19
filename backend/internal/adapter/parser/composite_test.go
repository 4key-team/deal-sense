package parser_test

import (
	"os"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestCompositeParser_Supports(t *testing.T) {
	p := parser.NewComposite(parser.NewPDFParser(), parser.NewDocxReader())

	tests := []struct {
		ft   domain.FileType
		want bool
	}{
		{domain.FileTypePDF, true},
		{domain.FileTypeDOCX, true},
	}
	for _, tt := range tests {
		if got := p.Supports(tt.ft); got != tt.want {
			t.Errorf("Supports(%v) = %v, want %v", tt.ft, got, tt.want)
		}
	}
}

func TestCompositeParser_Supports_Empty(t *testing.T) {
	p := parser.NewComposite() // no parsers
	if p.Supports(domain.FileTypePDF) {
		t.Error("empty composite should not support anything")
	}
}

func TestCompositeParser_Parse_NoParsers(t *testing.T) {
	p := parser.NewComposite() // no parsers
	_, err := p.Parse(t.Context(), "file.pdf", []byte("data"))
	if err == nil {
		t.Error("expected error when no parser available")
	}
}

func TestCompositeParser_Parse(t *testing.T) {
	p := parser.NewComposite(parser.NewPDFParser(), parser.NewDocxReader())

	t.Run("routes PDF to PDFParser", func(t *testing.T) {
		data, _ := os.ReadFile("testdata/hello.pdf")
		text, err := p.Parse(t.Context(), "test.pdf", data)
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		if text == "" {
			t.Error("expected non-empty text from PDF")
		}
	})

	t.Run("routes DOCX to DocxReader", func(t *testing.T) {
		data, _ := os.ReadFile("testdata/hello.docx")
		text, err := p.Parse(t.Context(), "test.docx", data)
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		if text == "" {
			t.Error("expected non-empty text from DOCX")
		}
	})

	t.Run("rejects unsupported extension", func(t *testing.T) {
		_, err := p.Parse(t.Context(), "file.txt", []byte("data"))
		if err == nil {
			t.Error("expected error for unsupported file type")
		}
	})
}
