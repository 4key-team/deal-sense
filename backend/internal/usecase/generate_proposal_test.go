package usecase_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// makeTestDocxForUsecase builds a minimal DOCX (zip) with given XML as word/document.xml.
func makeTestDocxForUsecase(documentXML string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("word/document.xml")
	f.Write([]byte(documentXML))
	w.Close()
	return buf.Bytes()
}

type stubTemplateEngine struct {
	result []byte
	err    error
}

func (s *stubTemplateEngine) Fill(_ context.Context, _ []byte, _ map[string]string) ([]byte, error) {
	return s.result, s.err
}

type stubGenerativeEngine struct {
	result      []byte
	err         error
	called      bool
	cleanCalled bool
}

func (s *stubGenerativeEngine) GenerativeFill(_ context.Context, _ []byte, _ []usecase.ContentSection) ([]byte, error) {
	s.called = true
	return s.result, s.err
}

func (s *stubGenerativeEngine) GenerateClean(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	s.cleanCalled = true
	return s.result, s.err
}

type stubPDFGenerator struct {
	result []byte
	err    error
	called bool
}

func (s *stubPDFGenerator) Generate(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	s.called = true
	return s.result, s.err
}

type stubDOCXToPDF struct {
	result   []byte
	err      error
	called   bool
	received []byte
}

func (s *stubDOCXToPDF) Convert(_ context.Context, docx []byte) ([]byte, error) {
	s.called = true
	s.received = docx
	return s.result, s.err
}

type stubMDGenerator struct {
	result []byte
	err    error
	called bool
}

func (s *stubMDGenerator) Render(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	s.called = true
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

func TestGenerateProposal_GenerativeMode(t *testing.T) {
	// Build a DOCX without {{placeholders}} so DetectTemplateMode returns ModeGenerative.
	docx := makeTestDocxForUsecase(`<w:document><w:body><w:p><w:r><w:t>Plain text</w:t></w:r></w:p></w:body></w:document>`)

	llmResp := `{
		"meta":{"client":"Acme","project":"Portal"},
		"sections":[{"title":"Intro","content":"Generated intro text","status":"ai","tokens":50}],
		"summary":"Generated proposal",
		"log":[{"time":"14:00","msg":"done"}]
	}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "parsed text"}
	tmplEng := &stubTemplateEngine{result: []byte("should not be called")}
	genEng := &stubGenerativeEngine{result: []byte("generative-output")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "placeholder prompt")
	uc.SetGenerativeEngine(genEng, "generative prompt")
	result, _, err := uc.Execute(t.Context(), "offer.docx", docx, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !genEng.called {
		t.Error("expected GenerativeFill to be called")
	}
	if result.Mode() != domain.ModeGenerative {
		t.Errorf("Mode() = %q, want generative", result.Mode())
	}
	if string(result.Result()) != "generative-output" {
		t.Errorf("Result() = %q, want generative-output", result.Result())
	}
}

func TestGenerateProposal_CleanMode(t *testing.T) {
	llmResp := `{
		"meta":{"client":"Acme","project":"Portal"},
		"sections":[{"title":"Intro","content":"Clean content","status":"ai","tokens":50}],
		"summary":"Clean proposal",
		"log":[{"time":"14:00","msg":"done"}]
	}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "parsed text"}
	tmplEng := &stubTemplateEngine{result: []byte("should not be called")}
	genEng := &stubGenerativeEngine{result: []byte("clean-output")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "placeholder prompt")
	uc.SetGenerativeEngine(genEng, "generative prompt")
	result, _, err := uc.Execute(t.Context(), "", nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !genEng.cleanCalled {
		t.Error("expected GenerateClean to be called")
	}
	if genEng.called {
		t.Error("GenerativeFill should NOT be called in clean mode")
	}
	if result.Mode() != domain.ModeClean {
		t.Errorf("Mode() = %q, want clean", result.Mode())
	}
	if string(result.Result()) != "clean-output" {
		t.Errorf("Result() = %q, want clean-output", result.Result())
	}
}

func TestGenerateProposal_PDFGeneration(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Intro","status":"ai","tokens":50}],"summary":"ok"}`

	tests := []struct {
		name       string
		pdfGen     *stubPDFGenerator
		wantPDF    bool
	}{
		{
			name:    "pdf generator produces output",
			pdfGen:  &stubPDFGenerator{result: []byte("%PDF-1.4")},
			wantPDF: true,
		},
		{
			name:    "nil pdf generator — no PDF",
			pdfGen:  nil,
			wantPDF: false,
		},
		{
			name:    "pdf generator error — DOCX still ok",
			pdfGen:  &stubPDFGenerator{err: errors.New("pdf failed")},
			wantPDF: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
			parser := &stubParser{content: "text"}
			tmplEng := &stubTemplateEngine{result: []byte("filled-docx")}

			uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "prompt")
			if tt.pdfGen != nil {
				uc.SetPDFGenerator(tt.pdfGen)
			}
			result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// DOCX always present
			if len(result.Result()) == 0 {
				t.Error("expected DOCX result")
			}

			// PDF conditionally present
			hasPDF := len(result.PDFResult()) > 0
			if hasPDF != tt.wantPDF {
				t.Errorf("PDFResult present = %v, want %v", hasPDF, tt.wantPDF)
			}

			if tt.pdfGen != nil && tt.pdfGen.err == nil && !tt.pdfGen.called {
				t.Error("expected PDF generator to be called")
			}
		})
	}
}

func TestGenerateProposal_MDGeneration(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Intro","content":"Hello","status":"ai","tokens":50}],"summary":"ok"}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "text"}
	tmplEng := &stubTemplateEngine{result: []byte("filled")}
	mdGen := &stubMDGenerator{result: []byte("# Proposal\n\nContent")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "prompt")
	uc.SetMDGenerator(mdGen)
	result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mdGen.called {
		t.Error("expected MD generator to be called")
	}
	if len(result.MDResult()) == 0 {
		t.Error("expected non-empty MD result")
	}
}

func TestGenerateProposal_MDGeneratorError_StillOk(t *testing.T) {
	llmResp := `{"params":{"x":"y"},"sections":[],"summary":"ok"}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "text"}
	tmplEng := &stubTemplateEngine{result: []byte("filled")}
	mdGen := &stubMDGenerator{err: errors.New("md failed")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "prompt")
	uc.SetMDGenerator(mdGen)
	result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.MDResult()) != 0 {
		t.Error("expected empty MD result on error")
	}
}

func TestGenerateProposal_PDFTemplate_GenerativeMode(t *testing.T) {
	// PDF шаблон — не DOCX, значит DetectTemplateMode вернёт ошибку,
	// но у нас generative engine → должен сработать generative mode.
	llmResp := `{
		"meta":{"client":"Acme"},
		"sections":[{"title":"About","content":"We are great","status":"ai","tokens":30}],
		"summary":"PDF proposal",
		"log":[]
	}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "pdf text content"}
	tmplEng := &stubTemplateEngine{result: []byte("should not be used")}
	genEng := &stubGenerativeEngine{result: []byte("generative-pdf-output")}
	pdfGen := &stubPDFGenerator{result: []byte("%PDF-1.4")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "placeholder prompt")
	uc.SetGenerativeEngine(genEng, "generative prompt")
	uc.SetPDFGenerator(pdfGen)

	// "report.pdf" — не DOCX, DetectTemplateMode вернёт ошибку → fallback к placeholder,
	// НО у нас есть generative engine, нужно перейти в generative mode для не-DOCX.
	result, _, err := uc.Execute(t.Context(), "report.pdf", []byte("pdf-data"), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Для не-DOCX шаблонов должен быть generative mode
	if result.Mode() != domain.ModeGenerative {
		t.Errorf("Mode() = %q, want generative", result.Mode())
	}
}

func TestGenerateProposal_DOCXToPDFConverter(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Intro","content":"Hello","status":"ai","tokens":50}],"summary":"ok"}`

	t.Run("converter produces PDF from filled DOCX", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
		parser := &stubParser{content: "text"}
		tmplEng := &stubTemplateEngine{result: []byte("filled-docx-bytes")}
		converter := &stubDOCXToPDF{result: []byte("%PDF-from-docx")}
		marotoPDF := &stubPDFGenerator{result: []byte("%PDF-maroto")}

		uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "prompt")
		uc.SetDOCXToPDFConverter(converter)
		uc.SetPDFGenerator(marotoPDF)
		result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !converter.called {
			t.Fatal("expected converter to be called")
		}
		if string(converter.received) != "filled-docx-bytes" {
			t.Errorf("converter received %q, want filled-docx-bytes", converter.received)
		}
		if string(result.PDFResult()) != "%PDF-from-docx" {
			t.Errorf("PDFResult = %q, want %%PDF-from-docx", result.PDFResult())
		}
	})

	t.Run("converter error falls back to Maroto", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
		parser := &stubParser{content: "text"}
		tmplEng := &stubTemplateEngine{result: []byte("filled-docx")}
		converter := &stubDOCXToPDF{err: errors.New("soffice failed")}
		marotoPDF := &stubPDFGenerator{result: []byte("%PDF-maroto")}

		uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "prompt")
		uc.SetDOCXToPDFConverter(converter)
		uc.SetPDFGenerator(marotoPDF)
		result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !marotoPDF.called {
			t.Error("expected Maroto fallback to be called")
		}
		if string(result.PDFResult()) != "%PDF-maroto" {
			t.Errorf("PDFResult = %q, want %%PDF-maroto", result.PDFResult())
		}
	})

	t.Run("no converter — uses Maroto directly", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
		parser := &stubParser{content: "text"}
		tmplEng := &stubTemplateEngine{result: []byte("filled-docx")}
		marotoPDF := &stubPDFGenerator{result: []byte("%PDF-maroto")}

		uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "prompt")
		uc.SetPDFGenerator(marotoPDF)
		result, _, err := uc.Execute(t.Context(), "offer.docx", []byte("template"), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !marotoPDF.called {
			t.Error("expected Maroto to be called")
		}
		if string(result.PDFResult()) != "%PDF-maroto" {
			t.Errorf("PDFResult = %q, want %%PDF-maroto", result.PDFResult())
		}
	})
}

func TestGenerateProposal_PlaceholderMode_WithGenerativeEngine(t *testing.T) {
	// Build a DOCX WITH {{placeholders}} — should use placeholder mode.
	docx := makeTestDocxForUsecase(`<w:document><w:body><w:p><w:r><w:t>Hello {{client_name}}</w:t></w:r></w:p></w:body></w:document>`)

	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Intro","status":"ai","tokens":50}],"summary":"ok"}`
	llm := &stubLLM{response: llmResp, name: "test", usage: domain.NewTokenUsage(100, 200)}
	parser := &stubParser{content: "parsed text"}
	tmplEng := &stubTemplateEngine{result: []byte("placeholder-filled")}
	genEng := &stubGenerativeEngine{result: []byte("should-not-be-used")}

	uc := usecase.NewGenerateProposal(llm, parser, tmplEng, "placeholder prompt")
	uc.SetGenerativeEngine(genEng, "generative prompt")
	result, _, err := uc.Execute(t.Context(), "offer.docx", docx, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if genEng.called {
		t.Error("GenerativeFill should NOT be called in placeholder mode")
	}
	if result.Mode() != domain.ModePlaceholder {
		t.Errorf("Mode() = %q, want placeholder", result.Mode())
	}
}
