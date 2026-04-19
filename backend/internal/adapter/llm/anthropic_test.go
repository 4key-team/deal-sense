package llm_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
)

func TestAnthropic_Name(t *testing.T) {
	p := llm.NewAnthropic(llm.AnthropicConfig{
		BaseURL: "http://localhost",
		APIKey:  "sk-ant-test",
		Model:   "claude-sonnet-4-5",
	})
	if got := p.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
}

func TestAnthropic_GenerateCompletion(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		response   map[string]any
		wantErr    bool
		wantResult string
	}{
		{
			name:   "successful completion",
			status: http.StatusOK,
			response: map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": "Hello from Claude"},
				},
			},
			wantResult: "Hello from Claude",
		},
		{
			name:   "empty content",
			status: http.StatusOK,
			response: map[string]any{
				"content": []map[string]any{},
			},
			wantErr: true,
		},
		{
			name:     "auth error",
			status:   http.StatusUnauthorized,
			response: map[string]any{"error": map[string]any{"message": "invalid api key"}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/v1/messages" {
					t.Errorf("path = %s, want /v1/messages", r.URL.Path)
				}
				if r.Header.Get("x-api-key") != "sk-ant-test" {
					t.Errorf("x-api-key header missing or wrong")
				}
				if r.Header.Get("anthropic-version") == "" {
					t.Error("anthropic-version header missing")
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]any
				json.Unmarshal(body, &req)

				if req["model"] != "claude-sonnet-4-5" {
					t.Errorf("model = %v, want claude-sonnet-4-5", req["model"])
				}
				if req["system"] == nil {
					t.Error("system prompt missing")
				}

				w.WriteHeader(tt.status)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			p := llm.NewAnthropic(llm.AnthropicConfig{
				BaseURL: srv.URL,
				APIKey:  "sk-ant-test",
				Model:   "claude-sonnet-4-5",
			})

			result, err := p.GenerateCompletion(t.Context(), "system", "user")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.wantResult {
				t.Errorf("result = %q, want %q", result, tt.wantResult)
			}
		})
	}
}

func TestAnthropic_ConnectionRefused(t *testing.T) {
	p := llm.NewAnthropic(llm.AnthropicConfig{
		BaseURL: "http://127.0.0.1:1", APIKey: "sk", Model: "m",
	})
	_, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestAnthropic_InvalidBaseURL(t *testing.T) {
	p := llm.NewAnthropic(llm.AnthropicConfig{
		BaseURL: "://bad\x7furl", APIKey: "sk", Model: "m",
	})
	_, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected URL parse error")
	}
}

func TestAnthropic_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	p := llm.NewAnthropic(llm.AnthropicConfig{
		BaseURL: srv.URL, APIKey: "sk", Model: "m",
	})
	_, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestAnthropic_NoTextBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "image", "source": "data"},
			},
		})
	}))
	defer srv.Close()

	p := llm.NewAnthropic(llm.AnthropicConfig{
		BaseURL: srv.URL, APIKey: "sk", Model: "m",
	})
	_, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected error for no text block")
	}
}

func TestAnthropic_CheckConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "ok"},
			},
		})
	}))
	defer srv.Close()

	p := llm.NewAnthropic(llm.AnthropicConfig{
		BaseURL: srv.URL, APIKey: "sk", Model: "claude-sonnet-4-5",
	})
	if err := p.CheckConnection(t.Context()); err != nil {
		t.Errorf("CheckConnection() error: %v", err)
	}
}
