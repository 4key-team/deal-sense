package llm_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
)

func TestNewLLMProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   llm.ProviderConfig
		wantName string
		wantErr  bool
	}{
		{
			name:     "openai",
			config:   llm.ProviderConfig{Provider: "openai", BaseURL: "http://localhost", APIKey: "sk", Model: "gpt-4o"},
			wantName: "openai",
		},
		{
			name:     "groq via openai-compat",
			config:   llm.ProviderConfig{Provider: "groq", BaseURL: "http://localhost", APIKey: "sk", Model: "llama-3.3-70b"},
			wantName: "groq",
		},
		{
			name:     "ollama via openai-compat",
			config:   llm.ProviderConfig{Provider: "ollama", BaseURL: "http://localhost", APIKey: "", Model: "llama3.1"},
			wantName: "ollama",
		},
		{
			name:     "anthropic",
			config:   llm.ProviderConfig{Provider: "anthropic", BaseURL: "http://localhost", APIKey: "sk-ant", Model: "claude-sonnet-4-5"},
			wantName: "anthropic",
		},
		{
			name:     "gemini",
			config:   llm.ProviderConfig{Provider: "gemini", BaseURL: "http://localhost", APIKey: "key", Model: "gemini-2.5-pro"},
			wantName: "gemini",
		},
		{
			name:     "custom uses openai-compat",
			config:   llm.ProviderConfig{Provider: "custom", BaseURL: "http://localhost", APIKey: "sk", Model: "my-model"},
			wantName: "custom",
		},
		{
			name:    "unknown provider",
			config:  llm.ProviderConfig{Provider: "deepseek-unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := llm.NewLLMProvider(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", p.Name(), tt.wantName)
			}
		})
	}
}
