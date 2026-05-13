package llm_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
	"github.com/daniil/deal-sense/backend/internal/domain/security"
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
// is wrapped by the security prefix in the correct position (HEAD), exactly
// once, and contains the юр firewall redirect verbatim. Per-directive marker
// coverage lives in domain/security/policy_test.go.
//
// Strengthening over a plain Contains check:
//   - HasPrefix proves the security text comes FIRST (an attacker-controlled
//     suffix can't override an earlier system instruction).
//   - Count==1 guards against accidental double-wrap (re-init bug).
//   - Exact redirect phrase ensures the юр firewall has the literal text the
//     LLM is instructed to respond with — substring matches like "юрист" alone
//     would pass for unrelated phrasing.
func TestPromptsIncludeSecurityDirectives(t *testing.T) {
	const securityHeader = "[CRITICAL SECURITY DIRECTIVES"
	const exactRedirect = "Обратитесь к юристу компании"
	const uniqueMarker = "STRICT DOMAIN FOCUS"

	tests := []struct {
		name string
		got  string
	}{
		{"TenderAnalysisPrompt", llm.TenderAnalysisPrompt("Russian")},
		{"ProposalGenerationPrompt", llm.ProposalGenerationPrompt("Russian")},
		{"GenerativeProposalPrompt", llm.GenerativeProposalPrompt("Russian")},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/has_security_prefix", func(t *testing.T) {
			if !strings.HasPrefix(tt.got, securityHeader) {
				t.Errorf("%s must START with %q — order matters, suffix can't override", tt.name, securityHeader)
			}
		})
		t.Run(tt.name+"/exact_redirect_phrase", func(t *testing.T) {
			if !strings.Contains(tt.got, exactRedirect) {
				t.Errorf("%s missing exact redirect phrase %q — юр firewall verbatim text required", tt.name, exactRedirect)
			}
		})
		t.Run(tt.name+"/no_duplicate_prefix", func(t *testing.T) {
			if c := strings.Count(tt.got, uniqueMarker); c != 1 {
				t.Errorf("%s contains %q %d times, want 1 — double-wrap bug", tt.name, uniqueMarker, c)
			}
		})
	}
}

// TestInitWrappedPrompts_PanicsOnPolicyError covers the panic branch of
// initWrappedPrompts via the policyLoader testable seam. Without this, a
// silent regression in the embed.FS or marker set would only surface at
// production startup.
func TestInitWrappedPrompts_PanicsOnPolicyError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when policy loader returns error")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "default security policy init failed") {
			t.Errorf("panic message must explain root cause, got %q", msg)
		}
	}()

	orig := llm.PolicyLoaderForTest()
	t.Cleanup(func() { llm.SetPolicyLoaderForTest(orig) })

	llm.SetPolicyLoaderForTest(func() (*security.Policy, error) {
		return nil, errors.New("forced failure for test")
	})

	llm.InitWrappedPromptsForTest()
}
