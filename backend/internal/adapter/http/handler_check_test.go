package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
)

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
			h := handler.NewHandler(llm, nil, nil)

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
