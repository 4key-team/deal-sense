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
