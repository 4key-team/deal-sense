package llm_test

import (
	"strings"
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

func TestTenderAnalysisPrompt(t *testing.T) {
	tests := []struct {
		lang     string
		wantLang string
	}{
		{"Russian", "Russian"},
		{"English", "English"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			prompt := llm.TenderAnalysisPrompt(tt.lang)
			if !strings.Contains(prompt, tt.wantLang) {
				t.Errorf("prompt does not contain %q", tt.wantLang)
			}
			if !strings.Contains(prompt, "verdict") {
				t.Error("prompt missing expected structure (verdict)")
			}
		})
	}
}

func TestProposalGenerationPrompt(t *testing.T) {
	tests := []struct {
		lang     string
		wantLang string
	}{
		{"Russian", "Russian"},
		{"English", "English"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			prompt := llm.ProposalGenerationPrompt(tt.lang)
			if !strings.Contains(prompt, tt.wantLang) {
				t.Errorf("prompt does not contain %q", tt.wantLang)
			}
			if !strings.Contains(prompt, "params") {
				t.Error("prompt missing expected structure (params)")
			}
		})
	}
}
