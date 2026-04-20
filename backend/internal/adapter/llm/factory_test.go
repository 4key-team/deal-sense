package llm_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestFactory_Create(t *testing.T) {
	factory := llm.Factory{}
	tests := []struct {
		provider string
		wantName string
		wantErr  bool
	}{
		{"openai", "openai", false},
		{"groq", "groq", false},
		{"anthropic", "anthropic", false},
		{"gemini", "gemini", false},
		{"unknown", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			p, err := factory.Create(usecase.LLMProviderConfig{
				Provider: tt.provider, BaseURL: "http://localhost",
				APIKey: "key", Model: "model",
			})
			if tt.wantErr {
				if err == nil { t.Fatal("expected error") }
				return
			}
			if err != nil { t.Fatalf("unexpected error: %v", err) }
			if p.Name() != tt.wantName { t.Errorf("name = %q, want %q", p.Name(), tt.wantName) }
		})
	}
}
