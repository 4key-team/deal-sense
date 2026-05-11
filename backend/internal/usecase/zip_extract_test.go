package usecase_test

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func makeZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(data)
	}
	w.Close()
	return buf.Bytes()
}

func TestExtractZip(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantCount int
		wantErr   bool
		wantNames []string
	}{
		{
			name: "pdf and docx extracted",
			data: makeZip(t, map[string][]byte{
				"spec.pdf":   []byte("pdf-data"),
				"brief.docx": []byte("docx-data"),
			}),
			wantCount: 2,
		},
		{
			name: "md files extracted",
			data: makeZip(t, map[string][]byte{
				"spec.pdf":   []byte("pdf-data"),
				"brief.md":   []byte("# Brief"),
				"notes.docx": []byte("docx-data"),
			}),
			wantCount: 3,
		},
		{
			name: "unsupported files skipped",
			data: makeZip(t, map[string][]byte{
				"spec.pdf":   []byte("pdf-data"),
				"readme.txt": []byte("text"),
				"image.png":  []byte("img"),
			}),
			wantCount: 1,
			wantNames: []string{"spec.pdf"},
		},
		{
			name: "macOS artifacts skipped",
			data: makeZip(t, map[string][]byte{
				"spec.pdf":            []byte("pdf-data"),
				"__MACOSX/._spec.pdf": []byte("mac-meta"),
				".DS_Store":           []byte("ds"),
			}),
			wantCount: 1,
			wantNames: []string{"spec.pdf"},
		},
		{
			name: "nested directory — uses basename",
			data: makeZip(t, map[string][]byte{
				"docs/tender/spec.pdf": []byte("pdf-data"),
			}),
			wantCount: 1,
			wantNames: []string{"spec.pdf"},
		},
		{
			name:    "empty archive",
			data:    makeZip(t, map[string][]byte{}),
			wantErr: true,
		},
		{
			name:    "no supported files",
			data:    makeZip(t, map[string][]byte{"notes.txt": []byte("text")}),
			wantErr: true,
		},
		{
			name:    "invalid zip data",
			data:    []byte("not a zip"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := usecase.ExtractZip(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != tt.wantCount {
				t.Errorf("got %d files, want %d", len(result), tt.wantCount)
			}
			if tt.wantNames != nil {
				for i, want := range tt.wantNames {
					if i >= len(result) {
						break
					}
					if result[i].Name != want {
						t.Errorf("result[%d].Name = %q, want %q", i, result[i].Name, want)
					}
				}
			}
		})
	}
}
