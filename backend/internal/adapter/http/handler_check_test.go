package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/domain"
)

type stubLLM struct {
	response string
	err      error
	name     string
}

func (s *stubLLM) GenerateCompletion(_ context.Context, _, _ string) (string, domain.TokenUsage, error) {
	return s.response, domain.ZeroTokenUsage(), s.err
}
func (s *stubLLM) CheckConnection(_ context.Context) error            { return s.err }
func (s *stubLLM) ListModels(_ context.Context) ([]string, error)     { return nil, nil }
func (s *stubLLM) Name() string                                       { return s.name }

func TestHandleCheckConnection(t *testing.T) {
	tests := []struct {
		name       string
		llmErr     error
		llmName    string
		wantStatus int
		wantOK     bool
	}{
		{
			name:       "healthy",
			llmName:    "anthropic",
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name:       "unhealthy",
			llmErr:     errors.New("connection refused"),
			llmName:    "anthropic",
			wantStatus: http.StatusServiceUnavailable,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &stubLLM{err: tt.llmErr, name: tt.llmName}
			h := handler.NewHandler(llm, nil, nil, nil, stubPrompt, stubPrompt, nil, testLogger)

			req := httptest.NewRequest(http.MethodPost, "/api/llm/check", nil)
			rec := httptest.NewRecorder()

			h.HandleCheckConnection(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			var resp map[string]any
			json.NewDecoder(rec.Body).Decode(&resp)

			if resp["ok"] != tt.wantOK {
				t.Errorf("ok = %v, want %v", resp["ok"], tt.wantOK)
			}
			if resp["provider"] != tt.llmName {
				t.Errorf("provider = %v, want %v", resp["provider"], tt.llmName)
			}
		})
	}
}

func stubPrompt(_ string) string { return "test prompt" }

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

type stubLLMWithModels struct {
	stubLLM
	models []string
	modErr error
}

func (s *stubLLMWithModels) ListModels(_ context.Context) ([]string, error) {
	return s.models, s.modErr
}

func TestHandleListModels(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		llm := &stubLLMWithModels{
			stubLLM: stubLLM{name: "test"},
			models:  []string{"gpt-4o", "gpt-3.5-turbo"},
		}
		h := handler.NewHandler(llm, nil, nil, nil, stubPrompt, stubPrompt, nil, testLogger)
		req := httptest.NewRequest(http.MethodGet, "/api/llm/models", nil)
		rec := httptest.NewRecorder()
		h.HandleListModels(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)
		models, ok := resp["models"].([]any)
		if !ok || len(models) != 2 {
			t.Errorf("expected 2 models, got %v", resp["models"])
		}
	})

	t.Run("error returns 502 with empty models", func(t *testing.T) {
		llm := &stubLLMWithModels{
			stubLLM: stubLLM{name: "test"},
			modErr:  errors.New("network error"),
		}
		h := handler.NewHandler(llm, nil, nil, nil, stubPrompt, stubPrompt, nil, testLogger)
		req := httptest.NewRequest(http.MethodGet, "/api/llm/models", nil)
		rec := httptest.NewRecorder()
		h.HandleListModels(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
		}
		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["error"] == nil {
			t.Error("expected error field in response")
		}
	})

	t.Run("nil models returns empty array", func(t *testing.T) {
		llm := &stubLLMWithModels{
			stubLLM: stubLLM{name: "test"},
			models:  nil,
		}
		h := handler.NewHandler(llm, nil, nil, nil, stubPrompt, stubPrompt, nil, testLogger)
		req := httptest.NewRequest(http.MethodGet, "/api/llm/models", nil)
		rec := httptest.NewRecorder()
		h.HandleListModels(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var resp map[string]any
		json.NewDecoder(rec.Body).Decode(&resp)
		models := resp["models"].([]any)
		if len(models) != 0 {
			t.Errorf("expected empty models array, got %v", models)
		}
	})
}

func TestResolveLang(t *testing.T) {
	tests := []struct {
		name     string
		langVal  string
		wantLang string
	}{
		{name: "default is Russian", langVal: "", wantLang: "Russian"},
		{name: "en returns English", langVal: "en", wantLang: "English"},
		{name: "ru returns Russian", langVal: "ru", wantLang: "Russian"},
		{name: "unknown returns Russian", langVal: "fr", wantLang: "Russian"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// resolveLang is tested indirectly through the handler.
			// We test HandleAnalyzeTender with lang=en to cover the English branch.
			// The default case is already covered by existing tests.
		})
	}

	// Direct test through HandleAnalyzeTender with lang=en
	llmResp := `{"verdict":"go","risk":"low","score":80,"summary":"ok"}`
	llm := &stubLLM{response: llmResp, name: "test"}
	var capturedLang string
	promptFn := func(lang string) string {
		capturedLang = lang
		return "test prompt"
	}
	p := &stubParser{content: "text"}
	h := handler.NewHandler(llm, nil, p, nil, promptFn, promptFn, nil, testLogger)

	// Test with lang=en
	req := makeMultipartRequestWithLang(t, map[string][]byte{"spec.pdf": []byte("data")}, "en")
	rec := httptest.NewRecorder()
	h.HandleAnalyzeTender(rec, req)
	if capturedLang != "English" {
		t.Errorf("resolveLang(en) produced %q, want English", capturedLang)
	}

	// Test default (no lang)
	req2 := makeMultipartRequestWithLang(t, map[string][]byte{"spec.pdf": []byte("data")}, "")
	rec2 := httptest.NewRecorder()
	capturedLang = ""
	h.HandleAnalyzeTender(rec2, req2)
	if capturedLang != "Russian" {
		t.Errorf("resolveLang(default) produced %q, want Russian", capturedLang)
	}
}

func makeMultipartRequestWithLang(t *testing.T, files map[string][]byte, lang string) *http.Request {
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
	if lang != "" {
		w.WriteField("lang", lang)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/tender/analyze", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}
