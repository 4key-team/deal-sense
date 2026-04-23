package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

func TestRouter_Routes(t *testing.T) {
	h := handler.NewHandler(&stubLLM{name: "test"}, nil, &stubParser{content: "text"}, &stubTemplateEngine{result: []byte("doc")}, stubPrompt, stubPrompt, nil, testLogger)
	mux := handler.NewRouter(h)

	tests := []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodGet, "/api/llm/providers", http.StatusOK},
		{http.MethodPost, "/api/llm/check", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}
