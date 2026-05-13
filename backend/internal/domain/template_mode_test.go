package domain_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestParseTemplateMode(t *testing.T) {
	tests := []struct {
		input   string
		want    domain.TemplateMode
		wantErr error
	}{
		{"placeholder", domain.ModePlaceholder, nil},
		{"generative", domain.ModeGenerative, nil},
		{"clean", domain.ModeClean, nil},
		{"markdown", domain.ModeMarkdown, nil},
		{"", "", domain.ErrInvalidTemplateMode},
		{"unknown", "", domain.ErrInvalidTemplateMode},
		{"PLACEHOLDER", "", domain.ErrInvalidTemplateMode},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := domain.ParseTemplateMode(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseTemplateMode(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTemplateMode(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseTemplateMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTemplateMode_String(t *testing.T) {
	tests := []struct {
		mode domain.TemplateMode
		want string
	}{
		{domain.ModePlaceholder, "placeholder"},
		{domain.ModeGenerative, "generative"},
		{domain.ModeClean, "clean"},
		{domain.ModeMarkdown, "markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("TemplateMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
