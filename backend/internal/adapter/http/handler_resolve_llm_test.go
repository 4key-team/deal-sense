package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	handler "github.com/daniil/deal-sense/backend/internal/adapter/http"
	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

type stubFactory struct {
	provider usecase.LLMProvider
	err      error
}

func (f *stubFactory) Create(_ usecase.LLMProviderConfig) (usecase.LLMProvider, error) {
	return f.provider, f.err
}

type stubLLMForResolve struct {
	name string
}

func (s *stubLLMForResolve) GenerateCompletion(_ context.Context, _, _ string) (string, domain.TokenUsage, error) {
	return "", domain.TokenUsage{}, nil
}
func (s *stubLLMForResolve) CheckConnection(_ context.Context) error        { return nil }
func (s *stubLLMForResolve) ListModels(_ context.Context) ([]string, error) { return nil, nil }
func (s *stubLLMForResolve) Name() string                                   { return s.name }

func TestResolveLLM(t *testing.T) {
	tests := []struct {
		name         string
		defaultLLM   usecase.LLMProvider
		factory      usecase.LLMProviderFactory
		headers      map[string]string
		wantProvider string
	}{
		{
			name:         "no headers returns default",
			defaultLLM:   &stubLLMForResolve{name: "default-provider"},
			factory:      &stubFactory{provider: &stubLLMForResolve{name: "custom"}},
			headers:      nil,
			wantProvider: "default-provider",
		},
		{
			name:         "with provider header creates new",
			defaultLLM:   &stubLLMForResolve{name: "default-provider"},
			factory:      &stubFactory{provider: &stubLLMForResolve{name: "custom-openai"}},
			headers:      map[string]string{"X-LLM-Provider": "openai", "X-LLM-Key": "sk-test"},
			wantProvider: "custom-openai",
		},
		{
			name:         "factory error falls back to default",
			defaultLLM:   &stubLLMForResolve{name: "default-provider"},
			factory:      &stubFactory{err: errors.New("unsupported provider")},
			headers:      map[string]string{"X-LLM-Provider": "bad"},
			wantProvider: "default-provider",
		},
		{
			name:         "nil factory returns default",
			defaultLLM:   &stubLLMForResolve{name: "default-provider"},
			factory:      nil,
			headers:      map[string]string{"X-LLM-Provider": "openai"},
			wantProvider: "default-provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := handler.NewHandler(tt.defaultLLM, tt.factory, nil, nil, stubPrompt, stubPrompt)

			req := httptest.NewRequest(http.MethodPost, "/api/llm/check", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Use HandleCheckConnection as a proxy to exercise resolveLLM —
			// the response includes the provider name from the resolved LLM.
			rec := httptest.NewRecorder()
			h.HandleCheckConnection(rec, req)

			// The handler writes {"ok":true,"provider":"..."} on success.
			body := rec.Body.String()
			if !containsSubstring(body, tt.wantProvider) {
				t.Errorf("response body %q does not contain provider %q", body, tt.wantProvider)
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && findSubstring(s, sub)
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
