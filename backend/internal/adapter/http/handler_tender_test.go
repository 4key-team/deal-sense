package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"net/http/httptest"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

type stubParser struct {
	content string
	err     error
}

func (s *stubParser) Parse(_ context.Context, _ string, _ []byte) (string, error) {
	return s.content, s.err
}
func (s *stubParser) Supports(_ domain.FileType) bool { return true }

func makeMultipartRequest(t *testing.T, files map[string][]byte, fields map[string]string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for name, data := range files {
		fw, err := w.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write(data)
	}
	for k, v := range fields {
		w.WriteField(k, v)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/tender/analyze", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestHandleAnalyzeTender(t *testing.T) {
	t.Run("successful analysis", func(t *testing.T) {
		llm := &stubLLM{
			response: `{"verdict":"go","risk":"low","score":82,"summary":"Good fit"}`,
			name:     "test",
		}
		parser := &stubParser{content: "tender requirements"}
		h := handler.NewHandler(llm, parser, nil)

		req := makeMultipartRequest(t,
			map[string][]byte{"spec.pdf": []byte("pdf data")},
			map[string]string{"company_profile": "We build software"},
		)
		rec := httptest.NewRecorder()
		h.HandleAnalyzeTender(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d, body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["verdict"] != "go" {
			t.Errorf("verdict = %v, want go", resp["verdict"])
		}
	})

	t.Run("missing files", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, &stubParser{}, nil)

		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		w.WriteField("company_profile", "We build software")
		w.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/tender/analyze", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rec := httptest.NewRecorder()
		h.HandleAnalyzeTender(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing company profile", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, &stubParser{}, nil)

		req := makeMultipartRequest(t,
			map[string][]byte{"spec.pdf": []byte("pdf data")},
			nil,
		)
		rec := httptest.NewRecorder()
		h.HandleAnalyzeTender(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("unsupported file type", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, &stubParser{content: "text"}, nil)

		req := makeMultipartRequest(t,
			map[string][]byte{"spec.txt": []byte("data")},
			map[string]string{"company_profile": "Acme"},
		)
		rec := httptest.NewRecorder()
		h.HandleAnalyzeTender(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("llm error returns 500", func(t *testing.T) {
		llm := &stubLLM{response: "not json", name: "test"}
		p := &stubParser{content: "requirements"}
		h := handler.NewHandler(llm, p, nil)

		req := makeMultipartRequest(t,
			map[string][]byte{"spec.pdf": []byte("data")},
			map[string]string{"company_profile": "Acme"},
		)
		rec := httptest.NewRecorder()
		h.HandleAnalyzeTender(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("invalid multipart", func(t *testing.T) {
		h := handler.NewHandler(&stubLLM{name: "test"}, &stubParser{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/tender/analyze", strings.NewReader("not multipart"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=bad")
		rec := httptest.NewRecorder()
		h.HandleAnalyzeTender(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
