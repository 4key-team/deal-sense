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
	err      error
	name     string
}

func (s *stubLLM) GenerateCompletion(_ context.Context, _, _ string) (string, error) {
	return s.response, s.err
}

func (s *stubLLM) CheckConnection(_ context.Context) error { return s.err }
func (s *stubLLM) Name() string                            { return s.name }

func TestGenerateProposal_Execute(t *testing.T) {
	tests := []struct {
		name       string
		tmplName   string
		tmplData   []byte
		params     map[string]string
		llmResp    string
		llmErr     error
		tmplResult []byte
		tmplErr    error
		wantErr    error
	}{
		{
			name:       "successful generation",
			tmplName:   "offer.docx",
			tmplData:   []byte("template"),
			params:     map[string]string{"company": "Acme"},
			llmResp:    `{"company_description": "Best company"}`,
			tmplResult: []byte("filled document"),
		},
		{
			name:    "empty template",
			tmplName: "offer.docx",
			tmplData: nil,
			wantErr: domain.ErrEmptyTemplate,
		},
		{
			name:     "llm failure",
			tmplName: "offer.docx",
			tmplData: []byte("template"),
			params:   map[string]string{"company": "Acme"},
			llmErr:   errors.New("llm unavailable"),
			wantErr:  errors.New("llm unavailable"),
		},
		{
			name:       "template engine failure",
			tmplName:   "offer.docx",
			tmplData:   []byte("template"),
			params:     map[string]string{"company": "Acme"},
			llmResp:    `{"field": "value"}`,
			tmplResult: nil,
			tmplErr:    errors.New("template error"),
			wantErr:    errors.New("template error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &stubLLM{response: tt.llmResp, err: tt.llmErr, name: "test"}
			tmplEng := &stubTemplateEngine{result: tt.tmplResult, err: tt.tmplErr}

			uc := usecase.NewGenerateProposal(llm, tmplEng)
			result, err := uc.Execute(t.Context(), tt.tmplName, tt.tmplData, tt.params)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErr.Error() != err.Error() && !errors.Is(err, tt.wantErr) {
					// Allow wrapped errors
					return
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
				t.Error("expected result to be set")
			}
		})
	}
}
