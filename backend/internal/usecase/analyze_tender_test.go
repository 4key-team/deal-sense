package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

type stubParser struct {
	content string
	err     error
}

func (s *stubParser) Parse(_ context.Context, _ string, _ []byte) (string, error) {
	return s.content, s.err
}

func (s *stubParser) Supports(_ domain.FileType) bool { return true }

func TestAnalyzeTender_Execute(t *testing.T) {
	tests := []struct {
		name       string
		files      []usecase.FileInput
		profile    string
		parseText  string
		parseErr   error
		llmResp    string
		llmErr     error
		wantErr    bool
	}{
		{
			name:      "successful analysis",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp builds software",
			parseText: "tender requirements text",
			llmResp:   `{"verdict":"go","risk":"low","score":85,"summary":"Good fit"}`,
		},
		{
			name:    "no files",
			files:   nil,
			profile: "Acme Corp",
			wantErr: true,
		},
		{
			name:    "empty profile",
			files:   []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile: "",
			wantErr: true,
		},
		{
			name:      "parser failure",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "",
			parseErr:  errors.New("parse failed"),
			wantErr:   true,
		},
		{
			name:      "llm failure",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "requirements",
			llmErr:    errors.New("llm down"),
			wantErr:   true,
		},
		{
			name:      "parser returns empty text",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "",
			wantErr:   true,
		},
		{
			name:      "invalid llm JSON",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "requirements",
			llmResp:   "not json at all",
			wantErr:   true,
		},
		{
			name:      "invalid verdict in response",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "requirements",
			llmResp:   `{"verdict":"maybe","risk":"low","score":50,"summary":"x"}`,
			wantErr:   true,
		},
		{
			name:      "invalid risk in response",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "requirements",
			llmResp:   `{"verdict":"go","risk":"extreme","score":50,"summary":"x"}`,
			wantErr:   true,
		},
		{
			name:      "invalid score in response",
			files:     []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}},
			profile:   "Acme Corp",
			parseText: "requirements",
			llmResp:   `{"verdict":"go","risk":"low","score":200,"summary":"x"}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &stubParser{content: tt.parseText, err: tt.parseErr}
			llm := &stubLLM{response: tt.llmResp, err: tt.llmErr, name: "test"}

			uc := usecase.NewAnalyzeTender(llm, parser, "test prompt")
			result, err := uc.Execute(t.Context(), tt.files, tt.profile)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
		})
	}
}
