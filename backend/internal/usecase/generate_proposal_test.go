package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

type stubTemplateEngine struct {
	result []byte
	err    error
}

func (s *stubTemplateEngine) Fill(_ context.Context, _ []byte, _ map[string]string) ([]byte, error) {
	return s.result, s.err
}

type stubLLM struct {
	response string
	usage    domain.TokenUsage
	err      error
	name     string
}

func (s *stubLLM) GenerateCompletion(_ context.Context, _, _ string) (string, domain.TokenUsage, error) {
	return s.response, s.usage, s.err
}

func (s *stubLLM) CheckConnection(_ context.Context) error            { return s.err }
func (s *stubLLM) ListModels(_ context.Context) ([]string, error)     { return nil, nil }
func (s *stubLLM) Name() string                                       { return s.name }

func TestGenerateProposal_Execute(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme","summary":"Great project"},"sections":[{"title":"Резюме","status":"ai","tokens":120}],"summary":"КП сгенерировано"}`

	tests := []struct {
		name       string
		tmplName   string
		tmplData   []byte
		context    []usecase.FileInput
		params     map[string]string
		llmResp    string
		llmErr     error
		tmplResult []byte
		tmplErr    error
		wantErr    bool
		wantSecs   int
	}{
		{
			name:       "successful generation",
			tmplName:   "proposal.docx",
			tmplData:   []byte("template"),
			params:     map[string]string{"company": "Acme"},
			llmResp:    llmResp,
			tmplResult: []byte("filled document"),
			wantSecs:   1,
		},
		{
			name:     "empty template",
			tmplName: "proposal.docx",
			tmplData: nil,
			params:   map[string]string{"company": "Acme"},
			wantErr:  true,
		},
		{
			name:     "empty template name",
			tmplName: "",
			tmplData: []byte("template"),
			params:   map[string]string{"company": "Acme"},
			wantErr:  true,
		},
		{
			name:     "llm failure",
			tmplName: "offer.docx",
			tmplData: []byte("template"),
			params:   map[string]string{"company": "Acme"},
			llmErr:   errors.New("llm unavailable"),
			wantErr:  true,
		},
		{
			name:       "template engine failure",
			tmplName:   "offer.docx",
			tmplData:   []byte("template"),
			params:     map[string]string{"company": "Acme"},
			llmResp:    llmResp,
			tmplResult: nil,
			tmplErr:    errors.New("template error"),
			wantErr:    true,
		},
		{
			name:       "invalid llm json",
			tmplName:   "offer.docx",
			tmplData:   []byte("template"),
			llmResp:    "not json at all",
			tmplResult: []byte("filled"),
			wantErr:    true,
		},
		{
			name:       "with context files",
			tmplName:   "proposal.docx",
			tmplData:   []byte("template"),
			context:    []usecase.FileInput{{Name: "brief.pdf", Data: []byte("brief"), Type: domain.FileTypePDF}},
			llmResp:    llmResp,
			tmplResult: []byte("filled"),
			wantSecs:   1,
		},
		{
			name:       "invalid section status defaults to ai",
			tmplName:   "proposal.docx",
			tmplData:   []byte("template"),
			llmResp:    `{"params":{"x":"y"},"sections":[{"title":"Sec","status":"unknown","tokens":50}],"summary":"ok"}`,
			tmplResult: []byte("filled"),
			wantSecs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &stubLLM{response: tt.llmResp, err: tt.llmErr, name: "test", usage: domain.NewTokenUsage(100, 200)}
			parser := &stubParser{content: "parsed text"}
			tmplEng := &stubTemplateEngine{result: tt.tmplResult, err: tt.tmplErr}

			uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "test prompt")
			result, usage, err := uc.Execute(t.Context(), tt.tmplName, tt.tmplData, tt.context, tt.params)

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
			if result.Result() == nil {
				t.Error("expected result bytes to be set")
			}
			if len(result.Sections()) != tt.wantSecs {
				t.Errorf("sections = %d, want %d", len(result.Sections()), tt.wantSecs)
			}
			if tt.wantSecs > 0 && result.Summary() == "" {
				t.Error("expected non-empty summary")
			}
			if usage.TotalTokens() != 300 {
				t.Errorf("usage.TotalTokens() = %d, want 300", usage.TotalTokens())
			}
		})
	}
}
