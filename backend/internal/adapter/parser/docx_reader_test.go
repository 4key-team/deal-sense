package parser_test

import (
	"archive/zip"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestDocxReader_Supports(t *testing.T) {
	r := parser.NewDocxReader()

	tests := []struct {
		name string
		ft   domain.FileType
		want bool
	}{
		{name: "supports docx", ft: domain.FileTypeDOCX, want: true},
		{name: "rejects pdf", ft: domain.FileTypePDF, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Supports(tt.ft); got != tt.want {
				t.Errorf("Supports(%v) = %v, want %v", tt.ft, got, tt.want)
			}
		})
	}
}

func TestDocxReader_Parse(t *testing.T) {
	r := parser.NewDocxReader()

	t.Run("extracts text from valid DOCX", func(t *testing.T) {
		data, err := os.ReadFile("testdata/hello.docx")
		if err != nil {
			t.Fatalf("read testdata: %v", err)
		}

		text, err := r.Parse(t.Context(), "hello.docx", data)
		if err != nil {
			t.Fatalf("Parse() error: %v", err)
		}
		if !strings.Contains(text, "Test document content") {
			t.Errorf("Parse() = %q, want to contain 'Test document content'", text)
		}
		if !strings.Contains(text, "Second paragraph") {
			t.Errorf("Parse() = %q, want to contain 'Second paragraph'", text)
		}
	})

	t.Run("returns error for empty data", func(t *testing.T) {
		_, err := r.Parse(t.Context(), "empty.docx", nil)
		if err == nil {
			t.Error("expected error for empty data")
		}
	})

	t.Run("returns error for invalid DOCX", func(t *testing.T) {
		_, err := r.Parse(t.Context(), "bad.docx", []byte("not a docx"))
		if err == nil {
			t.Error("expected error for invalid DOCX")
		}
	})

	t.Run("returns error for ZIP without document.xml", func(t *testing.T) {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		fw, _ := w.Create("other.txt")
		fw.Write([]byte("hello"))
		w.Close()

		_, err := r.Parse(t.Context(), "nodoc.docx", buf.Bytes())
		if err == nil {
			t.Error("expected error for missing document.xml")
		}
	})

	t.Run("returns error for broken XML in document.xml", func(t *testing.T) {
		var buf bytes.Buffer
		w := zip.NewWriter(&buf)
		fw, _ := w.Create("word/document.xml")
		fw.Write([]byte("<broken xml"))
		w.Close()

		_, err := r.Parse(t.Context(), "broken.docx", buf.Bytes())
		if err == nil {
			t.Error("expected error for broken XML")
		}
	})
}
