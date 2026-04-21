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
			llm := &stubLLM{response: tt.llmResp, err: tt.llmErr, name: "test", usage: domain.NewTokenUsage(50, 100)}

			uc := usecase.NewAnalyzeTender(llm, parser, "test prompt")
			result, usage, err := uc.Execute(t.Context(), tt.files, tt.profile)

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
			if usage.TotalTokens() != 150 {
				t.Errorf("usage.TotalTokens() = %d, want 150", usage.TotalTokens())
			}
		})
	}
}

func TestAnalyzeTender_ProsConsRequirements(t *testing.T) {
	llmResp := `{
		"verdict":"go","risk":"medium","score":72,"summary":"Decent fit",
		"pros":[{"title":"Strong team","desc":"10 engineers"},{"title":"","desc":"skip me"}],
		"cons":[{"title":"No ISO","desc":"Missing cert"},{"title":"","desc":"bad"}],
		"requirements":[
			{"label":"Go experience","status":"met"},
			{"label":"ISO 27001","status":"partial"},
			{"label":"Track record","status":"miss"},
			{"label":"","status":"met"},
			{"label":"Bad status","status":"unknown"}
		],
		"effort":"~40 hours"
	}`

	parser := &stubParser{content: "requirements text"}
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(50, 100)}

	uc := usecase.NewAnalyzeTender(llm, parser, "test prompt")
	files := []usecase.FileInput{{Name: "spec.pdf", Data: []byte("data"), Type: domain.FileTypePDF}}
	result, _, err := uc.Execute(t.Context(), files, "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pros: 1 valid (empty title skipped)
	if len(result.Pros()) != 1 {
		t.Errorf("Pros() len = %d, want 1", len(result.Pros()))
	}
	if result.Pros()[0].Title() != "Strong team" {
		t.Errorf("Pros()[0].Title() = %q", result.Pros()[0].Title())
	}

	// Cons: 1 valid (empty title skipped)
	if len(result.Cons()) != 1 {
		t.Errorf("Cons() len = %d, want 1", len(result.Cons()))
	}

	// Requirements: 3 valid (empty label and unknown status skipped)
	if len(result.Requirements()) != 3 {
		t.Errorf("Requirements() len = %d, want 3", len(result.Requirements()))
	}

	if result.Effort() != "~40 hours" {
		t.Errorf("Effort() = %q, want ~40 hours", result.Effort())
	}
}

func TestAnalyzeTender_MultipleFiles(t *testing.T) {
	llmResp := `{"verdict":"no-go","risk":"high","score":30,"summary":"Bad fit"}`
	parser := &stubParser{content: "text"}
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(50, 100)}

	uc := usecase.NewAnalyzeTender(llm, parser, "test prompt")
	files := []usecase.FileInput{
		{Name: "spec.pdf", Data: []byte("data1"), Type: domain.FileTypePDF},
		{Name: "req.docx", Data: []byte("data2"), Type: domain.FileTypeDOCX},
	}
	result, _, err := uc.Execute(t.Context(), files, "Acme Corp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Documents()) != 2 {
		t.Errorf("Documents() len = %d, want 2", len(result.Documents()))
	}
}

func TestNewFileInput(t *testing.T) {
	tests := []struct {
		name    string
		fname   string
		wantErr bool
		wantFT  domain.FileType
	}{
		{name: "pdf", fname: "spec.pdf", wantFT: domain.FileTypePDF},
		{name: "docx", fname: "offer.docx", wantFT: domain.FileTypeDOCX},
		{name: "unsupported", fname: "notes.txt", wantErr: true},
		{name: "no extension", fname: "readme", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fi, err := usecase.NewFileInput(tt.fname, []byte("data"))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fi.Name != tt.fname {
				t.Errorf("Name = %q, want %q", fi.Name, tt.fname)
			}
			if fi.Type != tt.wantFT {
				t.Errorf("Type = %v, want %v", fi.Type, tt.wantFT)
			}
		})
	}
}
