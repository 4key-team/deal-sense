package usecase_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// makeTestDocx builds a minimal DOCX (zip) with the given XML as word/document.xml.
// Additional entries can be passed as name→content pairs.
func makeTestDocx(documentXML string, extra ...string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	f, _ := w.Create("word/document.xml")
	f.Write([]byte(documentXML))

	for i := 0; i+1 < len(extra); i += 2 {
		ef, _ := w.Create(extra[i])
		ef.Write([]byte(extra[i+1]))
	}

	w.Close()
	return buf.Bytes()
}

func TestDetectTemplateMode(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    domain.TemplateMode
		wantErr bool
	}{
		{
			name: "placeholder in document.xml",
			data: makeTestDocx(`<w:document><w:body><w:p><w:r><w:t>Hello {{client_name}}</w:t></w:r></w:p></w:body></w:document>`),
			want: domain.ModePlaceholder,
		},
		{
			name: "placeholder in header1.xml",
			data: makeTestDocx(
				`<w:document><w:body><w:p><w:r><w:t>No placeholders here</w:t></w:r></w:p></w:body></w:document>`,
				"word/header1.xml", `<w:hdr><w:p><w:r><w:t>{{company}}</w:t></w:r></w:p></w:hdr>`,
			),
			want: domain.ModePlaceholder,
		},
		{
			name: "placeholder in footer2.xml",
			data: makeTestDocx(
				`<w:document><w:body><w:p><w:r><w:t>Plain text</w:t></w:r></w:p></w:body></w:document>`,
				"word/footer2.xml", `<w:ftr><w:p><w:r><w:t>Page {{page}}</w:t></w:r></w:p></w:ftr>`,
			),
			want: domain.ModePlaceholder,
		},
		{
			name: "no placeholders — generative",
			data: makeTestDocx(`<w:document><w:body><w:p><w:r><w:t>Plain proposal text</w:t></w:r></w:p></w:body></w:document>`),
			want: domain.ModeGenerative,
		},
		{
			name: "invalid zip",
			data: []byte("not a zip"),
			wantErr: true,
		},
		{
			name: "empty template",
			data: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := usecase.DetectTemplateMode(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DetectTemplateMode() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DetectTemplateMode() unexpected error: %v", err)
			}
			if !errors.Is(err, nil) && got != tt.want {
				t.Errorf("DetectTemplateMode() = %q, want %q", got, tt.want)
			}
			if got != tt.want {
				t.Errorf("DetectTemplateMode() = %q, want %q", got, tt.want)
			}
		})
	}
}
