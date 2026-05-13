package parser_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

type stubDocConverter struct {
	docx []byte
	err  error
	got  []byte
}

func (s *stubDocConverter) ConvertToDOCX(_ context.Context, doc []byte) ([]byte, error) {
	s.got = doc
	return s.docx, s.err
}

// buildMinimalDocx produces a valid in-memory DOCX archive containing
// a single paragraph with the given text — enough for DocxReader to
// extract that text without any external tools.
func buildMinimalDocx(t *testing.T, text string) []byte {
	t.Helper()
	type r struct {
		XMLName xml.Name `xml:"w:r"`
		T       string   `xml:"w:t"`
	}
	type p struct {
		XMLName xml.Name `xml:"w:p"`
		R       r        `xml:"w:r"`
	}
	type body struct {
		XMLName xml.Name `xml:"w:body"`
		P       p        `xml:"w:p"`
	}
	type doc struct {
		XMLName xml.Name `xml:"w:document"`
		Body    body     `xml:"w:body"`
	}
	d := doc{Body: body{P: p{R: r{T: text}}}}
	xmlBytes, err := xml.Marshal(d)
	if err != nil {
		t.Fatalf("marshal docx xml: %v", err)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	fw.Write(xmlBytes)
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestDocParser_Supports(t *testing.T) {
	p := parser.NewDocParser(&stubDocConverter{})
	if !p.Supports(domain.FileTypeDOC) {
		t.Error("DocParser should support FileTypeDOC")
	}
	if p.Supports(domain.FileTypeDOCX) {
		t.Error("DocParser should NOT claim DOCX (handled by DocxReader)")
	}
	if p.Supports(domain.FileTypePDF) {
		t.Error("DocParser should NOT claim PDF")
	}
}

func TestDocParser_Parse(t *testing.T) {
	t.Run("converts .doc then extracts docx text", func(t *testing.T) {
		docx := buildMinimalDocx(t, "Hello from legacy doc")
		conv := &stubDocConverter{docx: docx}
		p := parser.NewDocParser(conv)

		text, err := p.Parse(context.Background(), "legacy.doc", []byte("ole2 bytes"))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		if text != "Hello from legacy doc" {
			t.Errorf("text = %q, want %q", text, "Hello from legacy doc")
		}
		if string(conv.got) != "ole2 bytes" {
			t.Errorf("converter received %q, want raw .doc bytes", string(conv.got))
		}
	})

	t.Run("converter error propagates", func(t *testing.T) {
		boom := errors.New("libreoffice down")
		conv := &stubDocConverter{err: boom}
		p := parser.NewDocParser(conv)

		_, err := p.Parse(context.Background(), "x.doc", []byte("x"))
		if !errors.Is(err, boom) {
			t.Errorf("err = %v, want wraps %v", err, boom)
		}
	})

	t.Run("empty input returns error without invoking converter", func(t *testing.T) {
		conv := &stubDocConverter{docx: []byte("never used")}
		p := parser.NewDocParser(conv)

		_, err := p.Parse(context.Background(), "x.doc", nil)
		if err == nil {
			t.Fatal("expected error on empty input")
		}
		if conv.got != nil {
			t.Errorf("converter was invoked with %d bytes; expected skip on empty", len(conv.got))
		}
	})
}
