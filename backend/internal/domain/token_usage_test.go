package domain_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestNewTokenUsage(t *testing.T) {
	tests := []struct {
		name       string
		prompt     int
		completion int
		wantTotal  int
	}{
		{name: "both positive", prompt: 100, completion: 200, wantTotal: 300},
		{name: "zero completion", prompt: 50, completion: 0, wantTotal: 50},
		{name: "both zero", prompt: 0, completion: 0, wantTotal: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := domain.NewTokenUsage(tt.prompt, tt.completion)
			if u.PromptTokens() != tt.prompt {
				t.Errorf("PromptTokens() = %d, want %d", u.PromptTokens(), tt.prompt)
			}
			if u.CompletionTokens() != tt.completion {
				t.Errorf("CompletionTokens() = %d, want %d", u.CompletionTokens(), tt.completion)
			}
			if u.TotalTokens() != tt.wantTotal {
				t.Errorf("TotalTokens() = %d, want %d", u.TotalTokens(), tt.wantTotal)
			}
		})
	}
}

func TestTokenUsage_Add(t *testing.T) {
	a := domain.NewTokenUsage(100, 200)
	b := domain.NewTokenUsage(50, 80)
	sum := a.Add(b)

	if sum.PromptTokens() != 150 {
		t.Errorf("PromptTokens() = %d, want 150", sum.PromptTokens())
	}
	if sum.CompletionTokens() != 280 {
		t.Errorf("CompletionTokens() = %d, want 280", sum.CompletionTokens())
	}
	if sum.TotalTokens() != 430 {
		t.Errorf("TotalTokens() = %d, want 430", sum.TotalTokens())
	}
}
