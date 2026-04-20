package http_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

type stubTemplateEngine struct {
	result []byte
	err    error
}

func (s *stubTemplateEngine) Fill(_ context.Context, _ []byte, _ map[string]string) ([]byte, error) {
	return s.result, s.err
}

func TestHandleGenerateProposal(t *testing.T) {
	llmResp := `{"params":{"client_name":"Acme"},"sections":[{"title":"Резюме","status":"ai","tokens":100}],"summary":"Done"}`

	t.Run("successful generation returns JSON", func(t *testing.T) {
		llm := &stubLLM{response: llmResp, name: "test"}
		tmpl := &stubTemplateEngine{result: []byte("filled document")}
		h := handler.NewHandler(llm, nil, &stubParser{content: "template text"}, tmpl)

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
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{})

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
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{})

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
		h := handler.NewHandler(llm, nil, &stubParser{content: "text"}, tmpl)

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
		h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{}, &stubTemplateEngine{})
		req := httptest.NewRequest(http.MethodPost, "/api/proposal/generate", strings.NewReader("bad"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=bad")
		rec := httptest.NewRecorder()
		h.HandleGenerateProposal(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
