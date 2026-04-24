package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

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

func (s *stubGenerativeEngine) GenerativeFill(_ context.Context, _ []byte, _ []usecase.GenerativeSection) ([]byte, error) {
	return s.result, s.err
}

type stubPDFGenerator struct {
	result []byte
	err    error
}

func (s *stubPDFGenerator) Generate(_ context.Context, _ usecase.PDFInput) ([]byte, error) {
	return s.result, s.err
}

func TestHandleGenerateProposal(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Резюме","status":"ai","tokens":100}],"summary":"Done"}`

	t.Run("successful generation returns JSON", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("filled document")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "template text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

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
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

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
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

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
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

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
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)
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
		h := handler.NewHandler(llm, nil, &stubParser{content: "template text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

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
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil)

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
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, stubPrompt, stubPrompt, nil, testLogger, pdfGen, genEng, stubPrompt)

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

	t.Run("with lang=en", func(t *testing.T) {
		llmResp := `{"params":{"x":"y"},"sections":[],"summary":"ok"}`
		var capturedLang string
		promptFn := func(lang string) string {
			capturedLang = lang
			return "test prompt"
		}
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("doc")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl, promptFn, promptFn, nil, testLogger, nil, nil, nil)

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
