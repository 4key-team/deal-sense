package llm_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
)

func TestOpenAICompatible_Name(t *testing.T) {
	p := llm.NewOpenAICompatible(llm.OpenAIConfig{
		BaseURL: "http://localhost",
		APIKey:  "sk-test",
		Model:   "gpt-4o",
		Name:    "openai",
	})

	if got := p.Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

func TestOpenAICompatible_GenerateCompletion(t *testing.T) {
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
				"choices": []map[string]any{
					{"message": map[string]any{"content": "Hello from LLM"}},
				},
			},
			wantResult: "Hello from LLM",
		},
		{
			name:   "empty choices",
			status: http.StatusOK,
			response: map[string]any{
				"choices": []map[string]any{},
			},
			wantErr: true,
		},
		{
			name:     "server error",
			status:   http.StatusInternalServerError,
			response: map[string]any{"error": map[string]any{"message": "internal"}},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request structure
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if r.URL.Path != "/chat/completions" {
					t.Errorf("path = %s, want /chat/completions", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer sk-test" {
					t.Errorf("auth header missing or wrong")
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]any
				json.Unmarshal(body, &req)

				if req["model"] != "gpt-4o" {
					t.Errorf("model = %v, want gpt-4o", req["model"])
				}

				w.WriteHeader(tt.status)
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer srv.Close()

			p := llm.NewOpenAICompatible(llm.OpenAIConfig{
				BaseURL: srv.URL,
				APIKey:  "sk-test",
				Model:   "gpt-4o",
				Name:    "openai",
			})

			result, _, err := p.GenerateCompletion(t.Context(), "system prompt", "user prompt")
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

func TestOpenAICompatible_ConnectionRefused(t *testing.T) {
	p := llm.NewOpenAICompatible(llm.OpenAIConfig{
		BaseURL: "http://127.0.0.1:1", APIKey: "sk", Model: "m", Name: "test",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestOpenAICompatible_InvalidBaseURL(t *testing.T) {
	p := llm.NewOpenAICompatible(llm.OpenAIConfig{
		BaseURL: "://bad\x7furl", APIKey: "sk", Model: "m", Name: "test",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected URL parse error")
	}
}

func TestOpenAICompatible_ServerErrorNoMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	p := llm.NewOpenAICompatible(llm.OpenAIConfig{
		BaseURL: srv.URL, APIKey: "sk", Model: "m", Name: "test",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected error")
	}
}

func TestOpenAICompatible_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	p := llm.NewOpenAICompatible(llm.OpenAIConfig{
		BaseURL: srv.URL, APIKey: "sk", Model: "m", Name: "test",
	})
	_, _, err := p.GenerateCompletion(t.Context(), "s", "u")
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestOpenAICompatible_ListModels(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/models" {
				t.Errorf("path = %s, want /models", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer sk-test" {
				t.Errorf("auth header missing or wrong")
			}
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "gpt-4o"},
					{"id": "gpt-3.5-turbo"},
				},
			})
		}))
		defer srv.Close()

		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: srv.URL, APIKey: "sk-test", Model: "gpt-4o", Name: "openai",
		})
		models, err := p.ListModels(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 2 {
			t.Errorf("got %d models, want 2", len(models))
		}
		if models[0] != "gpt-4o" {
			t.Errorf("models[0] = %q, want gpt-4o", models[0])
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: srv.URL, APIKey: "sk", Model: "m", Name: "test",
		})
		_, err := p.ListModels(t.Context())
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not json"))
		}))
		defer srv.Close()

		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: srv.URL, APIKey: "sk", Model: "m", Name: "test",
		})
		_, err := p.ListModels(t.Context())
		if err == nil {
			t.Error("expected parse error")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: "http://127.0.0.1:1", APIKey: "sk", Model: "m", Name: "test",
		})
		_, err := p.ListModels(t.Context())
		if err == nil {
			t.Error("expected connection error")
		}
	})

	t.Run("invalid base URL", func(t *testing.T) {
		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: "://bad\x7furl", APIKey: "sk", Model: "m", Name: "test",
		})
		_, err := p.ListModels(t.Context())
		if err == nil {
			t.Error("expected URL parse error")
		}
	})
}

func TestOpenAICompatible_CheckConnection(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{
					{"message": map[string]any{"content": "ok"}},
				},
			})
		}))
		defer srv.Close()

		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: srv.URL, APIKey: "sk-test", Model: "gpt-4o", Name: "test",
		})
		if err := p.CheckConnection(t.Context()); err != nil {
			t.Errorf("CheckConnection() error: %v", err)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "invalid key"}})
		}))
		defer srv.Close()

		p := llm.NewOpenAICompatible(llm.OpenAIConfig{
			BaseURL: srv.URL, APIKey: "bad-key", Model: "gpt-4o", Name: "test",
		})
		if err := p.CheckConnection(t.Context()); err == nil {
			t.Error("expected error for bad key")
		}
	})
}
