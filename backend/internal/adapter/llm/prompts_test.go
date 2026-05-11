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

// TestSecurityDirectives covers the six directives required by
// reflective-agent-defaults v1.4 Rule 4 v1.4 (agent-side security prompt).
// Markers prove each directive is present in the system prompt prefix.
func TestSecurityDirectives(t *testing.T) {
	tests := []struct {
		name    string
		marker  string
		purpose string
	}{
		{"domain_focus", "STRICT DOMAIN FOCUS", "anti-jailbreak / roleplay refusal"},
		{"encoded_payload", "ENCODED PAYLOAD ISOLATION", "Base64/Hex isolation"},
		{"no_cyberattacks", "NO CYBERATTACKS", "block hacking tools / raw payloads"},
		{"factual_integrity", "FACTUAL INTEGRITY", "juridical firewall — primary risk"},
		{"resource_abuse", "RESOURCE ABUSE", "block infinite loops / N>10 requests"},
		{"juridical_ru", "юрист компании", "RU redirect text for юр-вопросы"},
		{"polite_refusal", "Politely", "polite firm refusal without alt-paths"},
	}

	got := llm.SecurityDirectives()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(got, tt.marker) {
				t.Errorf("SecurityDirectives missing marker %q (%s)", tt.marker, tt.purpose)
			}
		})
	}
}

// TestPromptsIncludeSecurityDirectives verifies every existing LLM prompt
// is wrapped by the security prefix. Without this, the юр firewall is bypassed.
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
