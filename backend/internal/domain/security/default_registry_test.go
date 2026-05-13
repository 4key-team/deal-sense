package security_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain/security"
)

func TestDefaultEndpointRegistry_CoversAllRouterPaths(t *testing.T) {
	// These paths mirror backend/internal/adapter/http/router.go. The
	// router_test ensures the router serves them; this test ensures the
	// risk registry has an opinion on each.
	wantPaths := []string{
		"/api/llm/check",
		"/api/llm/providers",
		"/api/llm/models",
		"/api/tender/analyze",
		"/api/proposal/generate",
	}

	r := security.NewDefaultEndpointRegistry()
	for _, p := range wantPaths {
		if _, err := r.Lookup(p); err != nil {
			t.Errorf("path %q has no risk annotation: %v", p, err)
		}
	}
}

func TestDefaultEndpointRegistry_RiskAssignments(t *testing.T) {
	// Each endpoint's risk class is documented here, not inferred. Changes
	// here are intentional — if you reclassify something, the diff
	// flags it for review.
	tests := []struct {
		path string
		want security.RiskLevel
	}{
		{"/api/llm/check", security.RiskSafeRead},
		{"/api/llm/providers", security.RiskSafeRead},
		{"/api/llm/models", security.RiskSafeRead},
		// Tender analyze returns a verdict, doesn't persist anything.
		{"/api/tender/analyze", security.RiskSafeRead},
		// Proposal generate writes a document into the response body; the
		// artifact is "modified state" from the client's perspective even
		// though nothing is sent to a third party.
		{"/api/proposal/generate", security.RiskModify},
	}

	r := security.NewDefaultEndpointRegistry()
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := r.Lookup(tt.path)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("Lookup(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultEndpointRegistry_NoDestructiveYet(t *testing.T) {
	// Sanity: until destructive operations land (sendProposalToClient etc.),
	// no path should be classified DESTRUCTIVE. Catching an accidental
	// classification early keeps the type honest.
	r := security.NewDefaultEndpointRegistry()
	for _, p := range r.Paths() {
		level, _ := r.Lookup(p)
		if level == security.RiskDestructive {
			t.Errorf("path %q classified DESTRUCTIVE but no Layer 5 confirmation infra is wired yet", p)
		}
	}
}
