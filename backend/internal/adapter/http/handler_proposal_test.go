package http_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

type recordingParser struct {
	content string
	parsed  []string
}

func (s *recordingParser) Parse(_ context.Context, name string, _ []byte) (string, error) {
	s.parsed = append(s.parsed, name)
	return s.content, nil
}
func (s *recordingParser) Supports(_ domain.FileType) bool { return true }

type stubTemplateEngine struct {
	result []byte
	err    error
}

func (s *stubTemplateEngine) Fill(_ context.Context, _ []byte, _ map[string]string) ([]byte, error) {
	return s.result, s.err
}

type stubGenerativeEngine struct {
	result []byte
	err    error
}

func (s *stubGenerativeEngine) GenerativeFill(_ context.Context, _ []byte, _ []usecase.ContentSection) ([]byte, error) {
	return s.result, s.err
}

func (s *stubGenerativeEngine) GenerateClean(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	return s.result, s.err
}

type stubPDFGenerator struct {
	result []byte
	err    error
}

func (s *stubPDFGenerator) Generate(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	return s.result, s.err
}

type stubMDGenerator struct {
	result []byte
	err    error
}

func (s *stubMDGenerator) Render(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	return s.result, s.err
}

func TestHandleGenerateProposal(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Резюме","status":"ai","tokens":100}],"summary":"Done"}`

	t.Run("successful generation returns JSON", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("filled document")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "template text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template data"))
		w.WriteField("params", `{"company":"Acme"}`)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want JSON", ct)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `"summary"`) {
			t.Errorf("response missing summary: %s", body)
		}
		if !strings.Contains(body, `"sections"`) {
			t.Errorf("response missing sections: %s", body)
		}
	})

	t.Run("missing template", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		w.WriteField("params", `{}`)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid params JSON", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("data"))
		w.WriteField("params", `{invalid json}`)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("llm error returns 500", func(t *testing.T) {
		llm := &stubLLM{err: errors.New("llm down"), name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("doc")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template data"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("invalid multipart", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=bad")
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("full response with sections, meta, log, and docx", func(t *testing.T) {
		llmResp := `{
			"params":{"company_name":"Acme"},
			"meta":{"client":"ClientCo","project":"Portal"},
			"sections":[
				{"title":"Intro","status":"filled","tokens":50},
				{"title":"Body","status":"ai","tokens":200}
			],
			"summary":"КП сгенерировано",
			"log":[{"time":"14:00:01","msg":"started"},{"time":"14:00:02","msg":"done"}]
		}`
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("filled document bytes")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "template text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template data"))
		w.WriteField("params", `{"company":"Acme"}`)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)

		// Check sections
		sections, ok := resp["sections"].([]any)
		if !ok || len(sections) != 2 {
			t.Errorf("sections len = %v, want 2", len(sections))
		}

		// Check meta
		if resp["meta"] == nil {
			t.Error("expected meta in response")
		}

		// Check log
		logEntries, ok := resp["log"].([]any)
		if !ok || len(logEntries) != 2 {
			t.Errorf("log len = %v, want 2", len(logEntries))
		}

		// Check docx base64
		if resp["docx"] == nil || resp["docx"] == "" {
			t.Error("expected non-empty docx base64")
		}

		// Check usage
		usage, ok := resp["usage"].(map[string]any)
		if !ok {
			t.Error("expected usage object")
		} else if usage["total_tokens"] == nil {
			t.Error("expected total_tokens in usage")
		}
	})

	t.Run("with context files", func(t *testing.T) {
		llmResp := `{"params":{"x":"y"},"sections":[],"summary":"ok"}`
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("doc")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		// template file
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template"))
		// context file (pdf)
		cw, _ := w.CreateFormFile("context", "brief.pdf")
		cw.Write([]byte("brief data"))
		// context file (unsupported - should be skipped)
		cw2, _ := w.CreateFormFile("context", "notes.txt")
		cw2.Write([]byte("notes"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("response contains pdf and mode fields", func(t *testing.T) {
		llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Intro","status":"ai","tokens":100}],"summary":"Done"}`
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("filled")}
		pdfGen := &stubPDFGenerator{result: []byte("%PDF-1.4 test content")}
		genEng := &stubGenerativeEngine{result: []byte("gen")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, pdfGen, nil, genEng, stubPrompt, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template data"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp["pdf"] == nil || resp["pdf"] == "" {
			t.Error("expected non-empty pdf in response")
		}
		if resp["mode"] == nil {
			t.Error("expected mode in response")
		}
	})

	t.Run("response contains md field", func(t *testing.T) {
		llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Intro","status":"ai","tokens":100}],"summary":"Done"}`
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("filled")}
		pdfGen := &stubPDFGenerator{result: []byte("%PDF")}
		genEng := &stubGenerativeEngine{result: []byte("gen")}
		mdGen := &stubMDGenerator{result: []byte("# Proposal")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, pdfGen, nil, genEng, stubPrompt, mdGen)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template data"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp["md"] == nil || resp["md"] == "" {
			t.Error("expected non-empty md in response")
		}
	})

	t.Run("accepts pdf template", func(t *testing.T) {
		llmResp := `{"meta":{"client":"Acme"},"sections":[{"title":"About","content":"Text","status":"ai","tokens":30}],"summary":"ok","log":[]}`
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("should-not-be-used")}
		genEng := &stubGenerativeEngine{result: []byte("generative-output")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "pdf text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, genEng, stubPrompt, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "report.pdf")
		fw.Write([]byte("pdf-data"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp["mode"] != "generative" {
			t.Errorf("mode = %v, want generative", resp["mode"])
		}
	})

	t.Run("no template with generative engine returns clean mode", func(t *testing.T) {
		llmResp := `{"meta":{"client":"Acme"},"sections":[{"title":"Intro","content":"Clean text","status":"ai","tokens":30}],"summary":"clean","log":[]}`
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("unused")}
		genEng := &stubGenerativeEngine{result: []byte("clean-docx")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, genEng, stubPrompt, nil)

		// Send request without template file
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)

		if resp["mode"] != "clean" {
			t.Errorf("mode = %v, want clean", resp["mode"])
		}
	})

	t.Run("no template without generative engine returns 400", func(t *testing.T) {
		llm := &stubLLM{response: `{}`, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("unused")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, stubPrompt, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("zip in context: valid archive extracted to inner files", func(t *testing.T) {
		var zipBuf bytes.Buffer
		zw := zip.NewWriter(&zipBuf)
		fw, _ := zw.Create("brief.pdf")
		fw.Write([]byte("brief content"))
		zw.Close()

		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("doc")}
		parser := &recordingParser{content: "text"}
		h := handler.NewHandler(llm, nil, parser, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		tw, _ := w.CreateFormFile("template", "offer.docx")
		tw.Write([]byte("template"))
		cw, _ := w.CreateFormFile("context", "ctx.zip")
		cw.Write(zipBuf.Bytes())
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		if !slices.Contains(parser.parsed, "brief.pdf") {
			t.Errorf("parser was not called with extracted brief.pdf; parsed=%v", parser.parsed)
		}
	})

	t.Run("zip in context: invalid archive returns 400", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("doc")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		tw, _ := w.CreateFormFile("template", "offer.docx")
		tw.Write([]byte("template"))
		cw, _ := w.CreateFormFile("context", "broken.zip")
		cw.Write([]byte("not a zip"))
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("with lang=en", func(t *testing.T) {
		llmResp := `{"params":{"x":"y"},"sections":[],"summary":"ok"}`
		var capturedLang string
		promptFn := func(lang string) string {
			capturedLang = lang
			return "test prompt"
		}
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("doc")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, promptFn, promptFn, nil, testLogger, nil, nil, nil, nil, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("template", "offer.docx")
		fw.Write([]byte("template"))
		w.WriteField("lang", "en")
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if capturedLang != "English" {
			t.Errorf("lang = %q, want English", capturedLang)
		}
	})
}

// TestHandleGenerateProposal_MarkdownTemplate is the legacy /generate
// counterpart of the streaming markdown test. The Telegram bot adapter
// posts to this endpoint (it does not use SSE), so a regression here
// would surface first in the bot path. End-to-end check: .md template
// in, mode=markdown out, docx attachment populated.
func TestHandleGenerateProposal_MarkdownTemplate(t *testing.T) {
	llmResp := `{"meta":{"client":"Acme"},"sections":[{"title":"Intro","content":"LLM intro","status":"ai","tokens":40}],"summary":"Done","log":[]}`
	llm := &stubLLM{response: llmResp, name: "test"}
	tmpl := &stubTemplateEngine{result: []byte("should-not-fire")}
	genEng := &stubGenerativeEngine{result: []byte("md-clean-docx")}
	h := handler.NewHandler(llm, nil, &stubParser{content: "n/a"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, genEng, stubPrompt, nil)

	mdTemplate := []byte("# Doc\n## Intro\n\n## Pricing\nFixed: 1 200 000 RUB.\n")

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("template", "template.md")
	fw.Write(mdTemplate)
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	h.HandleGenerateProposal(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["mode"] != "markdown" {
		t.Errorf("mode = %v, want markdown", resp["mode"])
	}
	if resp["docx"] == nil || resp["docx"] == "" {
		t.Error("expected non-empty docx base64 in response")
	}
	sections, _ := resp["sections"].([]any)
	if len(sections) != 2 {
		t.Errorf("sections len = %d, want 2 (Intro + Pricing)", len(sections))
	}
}
