package llm_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
)

func TestGemini_Name(t *testing.T) {
	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: "http://localhost",
		APIKey:  "test-key",
		Model:   "gemini-2.5-pro",
	})
	if got := p.Name(); got != "gemini" {
		t.Errorf("Name() = %q, want %q", got, "gemini")
	}
}

func TestGemini_GenerateCompletion(t *testing.T) {
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
				"candidates": []map[string]any{
					{"content": map[string]any{
						"parts": []map[string]any{
							{"text": "Hello from Gemini"},
						},
					}},
				},
			},
			wantResult: "Hello from Gemini",
		},
		{
			name:   "empty candidates",
			status: http.StatusOK,
			response: map[string]any{
				"candidates": []map[string]any{},
			},
			wantErr: true,
		},
		{
			name:     "api error",
			status:   http.StatusBadRequest,
			response: map[string]any{"error": map[string]any{"message": "bad request"}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if !strings.Contains(r.URL.Path, "gemini-2.5-pro") {
					t.Errorf("path = %s, want to contain model name", r.URL.Path)
				}
				if r.URL.Query().Get("key") != "test-key" {
					t.Errorf("api key missing from query params")
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]any
				json.Unmarshal(body, &req)

				if req["systemInstruction"] == nil {
					t.Error("systemInstruction missing")
				}

				w.WriteHeader(tt.status)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			p := llm.NewGemini(llm.GeminiConfig{
				BaseURL: srv.URL,
				APIKey:  "test-key",
				Model:   "gemini-2.5-pro",
			})

			result, _, err := p.GenerateCompletion(t.Context(), "system", "user prompt")
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

func TestGemini_ConnectionRefused(t *testing.T) {
	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: "http://127.0.0.1:1", APIKey: "k", Model: "m",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestGemini_InvalidBaseURL(t *testing.T) {
	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: "://bad\x7furl", APIKey: "k", Model: "m",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected URL parse error")
	}
}

func TestGemini_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: srv.URL, APIKey: "k", Model: "m",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestGemini_EmptyParts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"parts": []map[string]any{}}},
			},
		})
	}))
	defer srv.Close()

	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: srv.URL, APIKey: "k", Model: "m",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected error for empty parts")
	}
}

func TestGemini_ListModels(t *testing.T) {
	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: "http://localhost", APIKey: "k", Model: "m",
	})
	models, err := p.ListModels(t.Context())
	if err != nil {
		t.Errorf("ListModels() error = %v", err)
	}
	if models != nil {
		t.Errorf("ListModels() = %v, want nil", models)
	}
}

func TestGemini_CheckConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]any{{"text": "ok"}},
				}},
			},
		})
	}))
	defer srv.Close()

	p := llm.NewGemini(llm.GeminiConfig{
		BaseURL: srv.URL, APIKey: "key", Model: "gemini-2.5-pro",
	})
	if err := p.CheckConnection(t.Context()); err != nil {
		t.Errorf("CheckConnection() error: %v", err)
	}
}
