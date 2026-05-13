package domain_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestParseFileType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.FileType
		wantErr error
	}{
		{name: "pdf without dot", input: "pdf", want: domain.FileTypePDF},
		{name: "pdf with dot", input: ".pdf", want: domain.FileTypePDF},
		{name: "docx without dot", input: "docx", want: domain.FileTypeDOCX},
		{name: "docx with dot", input: ".docx", want: domain.FileTypeDOCX},
		{name: "md without dot", input: "md", want: domain.FileTypeMD},
		{name: "md with dot", input: ".md", want: domain.FileTypeMD},
		{name: "doc without dot", input: "doc", want: domain.FileTypeDOC},
		{name: "doc with dot", input: ".doc", want: domain.FileTypeDOC},
		{name: "unsupported txt", input: "txt", wantErr: domain.ErrInvalidFileType},
		{name: "unsupported empty", input: "", wantErr: domain.ErrInvalidFileType},
		{name: "unsupported xlsx", input: "xlsx", wantErr: domain.ErrInvalidFileType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseFileType(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseFileType(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFileType(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseFileType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFileType_String(t *testing.T) {
	if got := domain.FileTypePDF.String(); got != "pdf" {
		t.Errorf("FileTypePDF.String() = %q, want %q", got, "pdf")
	}
}
