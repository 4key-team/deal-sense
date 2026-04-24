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
		{
			name:     "empty section title skipped",
			tmplName: "proposal.docx",
			tmplData: []byte("template"),
			llmResp:  `{"params":{"x":"y"},"sections":[{"title":"","status":"ai","tokens":50},{"title":"Valid","status":"ai","tokens":30}],"summary":"ok"}`,
			tmplResult: []byte("filled"),
			wantSecs: 1,
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

func TestGenerateProposal_MetaMerge(t *testing.T) {
	llmResp := `{
		"params":{"company_name":"Acme","summary":"Great project"},
		"meta":{"client":"ClientCo","project":"Portal","price":"1M RUB","timeline":"3 months"},
		"sections":[
			{"title":"Intro","status":"filled","tokens":50},
			{"title":"Body","status":"ai","tokens":200},
			{"title":"Pricing","status":"review","tokens":80}
		],
		"summary":"КП сгенерировано",
		"log":[
			{"time":"14:00:01","msg":"reading template"},
			{"time":"14:00:02","msg":"indexing context"},
			{"time":"14:00:03","msg":""},
			{"time":"14:00:04","msg":"generating sections"}
		]
	}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "parsed text"}
	tmplEng := &stubTemplateEngine{result: []byte("filled")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "test prompt")
	result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, map[string]string{"override": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Meta should be set
	if result.Meta() == nil {
		t.Fatal("expected non-nil meta")
	}
	if result.Meta()["client"] != "ClientCo" {
		t.Errorf("Meta()[client] = %q, want ClientCo", result.Meta()["client"])
	}

	// Sections: 3 (filled, ai, review)
	if len(result.Sections()) != 3 {
		t.Errorf("Sections() len = %d, want 3", len(result.Sections()))
	}
	if result.Sections()[0].Status() != domain.SectionFilled {
		t.Errorf("Sections()[0].Status() = %q, want filled", result.Sections()[0].Status())
	}
	if result.Sections()[2].Status() != domain.SectionReview {
		t.Errorf("Sections()[2].Status() = %q, want review", result.Sections()[2].Status())
	}

	// Log: 3 valid entries (empty msg skipped)
	if len(result.Log()) != 3 {
		t.Errorf("Log() len = %d, want 3", len(result.Log()))
	}
	if result.Log()[0].Time() != "14:00:01" {
		t.Errorf("Log()[0].Time() = %q", result.Log()[0].Time())
	}

	// Summary
	if result.Summary() != "КП сгенерировано" {
		t.Errorf("Summary() = %q", result.Summary())
	}
}

func TestGenerateProposal_TemplateParseFallback(t *testing.T) {
	llmResp := `{"params":{"x":"y"},"sections":[{"title":"Sec","status":"ai","tokens":50}],"summary":"ok"}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	// Parser returns error — simulates complex template that can't be read as text
	parser := &stubParser{content: "", err: errors.New("unsupported format")}
	tmplEng := &stubTemplateEngine{result: []byte("filled-docx")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "test prompt")
	result, _, err := uc.Execute(t.Context(), "complex.docx", []byte("template-data"), nil, nil)
	if err != nil {
		t.Fatalf("expected no error with fallback, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Result()) == 0 {
		t.Error("expected non-empty result bytes")
	}
}

func TestGenerateProposal_ContextFileParseError(t *testing.T) {
	llmResp := `{"params":{"x":"y"},"sections":[],"summary":"ok"}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "", err: errors.New("parse failed")}
	tmplEng := &stubTemplateEngine{result: []byte("filled")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "test prompt")
	// Context files with parse errors should be skipped, not fail
	ctx := []usecase.FileInput{{Name: "bad.pdf", Data: []byte("data"), Type: domain.FileTypePDF}}
	result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), ctx, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
