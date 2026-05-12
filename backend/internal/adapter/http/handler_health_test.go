package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

func TestHandleLiveness(t *testing.T) {
	h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{content: "x"}, &stubTemplateEngine{result: []byte("doc")}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.HandleLiveness(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestHandleReadiness_LLMConfigured(t *testing.T) {
	h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{content: "x"}, &stubTemplateEngine{result: []byte("doc")}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.HandleReadiness(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestHandleReadiness_NoLLMReturns503(t *testing.T) {
	// Handler constructed with nil LLM — readiness must report not-ready.
	h := handler.NewHandler(nil, nil, &stubParser{content: "x"}, &stubTemplateEngine{result: []byte("doc")}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h.HandleReadiness(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestRouter_HealthRoutes(t *testing.T) {
	h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{content: "x"}, &stubTemplateEngine{result: []byte("doc")}, stubPrompt, stubPrompt, nil, testLogger, nil, nil, nil, nil, nil)
	mux := handler.NewRouter(h, nil)

	for _, path := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", path, rec.Code)
		}
	}
}
