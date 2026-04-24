package parser_test

import (
	"context"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/parser"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestMDParser_Parse(t *testing.T) {
	p := parser.NewMDParser()

	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:  "simple markdown",
			input: []byte("# Title\n\nSome content here."),
			want:  "# Title\n\nSome content here.",
		},
		{
			name:  "russian markdown",
			input: []byte("# Коммерческое предложение\n\n## О компании\nМы лучшие."),
			want:  "# Коммерческое предложение\n\n## О компании\nМы лучшие.",
		},
		{
			name:    "empty content",
			input:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Parse(context.Background(), "doc.md", tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Parse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMDParser_Supports(t *testing.T) {
	p := parser.NewMDParser()
	if !p.Supports(domain.FileTypeMD) {
		t.Error("should support MD")
	}
	if p.Supports(domain.FileTypePDF) {
		t.Error("should not support PDF")
	}
}
