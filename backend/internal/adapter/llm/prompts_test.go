package llm_test

import (
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
)

func TestGenerativeProposalPrompt(t *testing.T) {
	tests := []struct {
		lang     string
		contains []string
	}{
		{
			lang:     "Russian",
			contains: []string{"sections", "content", "Russian"},
		},
		{
			lang:     "English",
			contains: []string{"sections", "content", "English"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			got := llm.GenerativeProposalPrompt(tt.lang)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("GenerativeProposalPrompt(%q) missing %q", tt.lang, want)
				}
			}
		})
	}
}

// TestPromptsIncludeSecurityDirectives verifies every existing LLM prompt
// is wrapped by the security prefix. Without this, the юр firewall is bypassed.
// Per-directive marker coverage lives in domain/security/policy_test.go.
func TestPromptsIncludeSecurityDirectives(t *testing.T) {
	tests := []struct {
		name string
		got  string
	}{
		{"TenderAnalysisPrompt", llm.TenderAnalysisPrompt("Russian")},
		{"ProposalGenerationPrompt", llm.ProposalGenerationPrompt("Russian")},
		{"GenerativeProposalPrompt", llm.GenerativeProposalPrompt("Russian")},
	}

	markers := []string{
		"STRICT DOMAIN FOCUS",
		"FACTUAL INTEGRITY",
		"юрист компании",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, m := range markers {
				if !strings.Contains(tt.got, m) {
					t.Errorf("%s missing security marker %q — юр firewall bypassed", tt.name, m)
				}
			}
		})
	}
}
