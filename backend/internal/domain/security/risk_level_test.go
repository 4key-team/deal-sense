package security_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain/security"
)

func TestNewRiskLevel_AcceptsCanonicalValues(t *testing.T) {
	tests := []struct {
		in   string
		want security.RiskLevel
	}{
		{"SAFE_READ", security.RiskSafeRead},
		{"MODIFY", security.RiskModify},
		{"DESTRUCTIVE", security.RiskDestructive},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := security.NewRiskLevel(tt.in)
			if err != nil {
				t.Fatalf("NewRiskLevel(%q) err = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewRiskLevel_RejectsUnknown(t *testing.T) {
	tests := []string{
		"",
		"safe_read", // case-sensitive
		"SAFE",
		"DELETE",
		"ANY",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, err := security.NewRiskLevel(tt)
			if !errors.Is(err, security.ErrInvalidRiskLevel) {
				t.Errorf("err = %v, want %v", err, security.ErrInvalidRiskLevel)
			}
		})
	}
}

func TestRiskLevel_String(t *testing.T) {
	if security.RiskSafeRead.String() != "SAFE_READ" {
		t.Errorf("got %q", security.RiskSafeRead.String())
	}
}

func TestEndpointRegistry_RegisterAndLookup(t *testing.T) {
	r := security.NewEndpointRegistry()
	if err := r.Register("/api/llm/providers", security.RiskSafeRead); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Register("/api/proposal/generate", security.RiskModify); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := r.Lookup("/api/llm/providers")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got != security.RiskSafeRead {
		t.Errorf("Lookup providers = %q, want SAFE_READ", got)
	}
}

func TestEndpointRegistry_RejectsDuplicate(t *testing.T) {
	r := security.NewEndpointRegistry()
	if err := r.Register("/api/x", security.RiskSafeRead); err != nil {
		t.Fatal(err)
	}
	err := r.Register("/api/x", security.RiskModify)
	if !errors.Is(err, security.ErrDuplicateEndpoint) {
		t.Errorf("err = %v, want %v", err, security.ErrDuplicateEndpoint)
	}
}

func TestEndpointRegistry_LookupUnknownReturnsError(t *testing.T) {
	r := security.NewEndpointRegistry()
	_, err := r.Lookup("/api/nope")
	if !errors.Is(err, security.ErrUnannotatedEndpoint) {
		t.Errorf("err = %v, want %v", err, security.ErrUnannotatedEndpoint)
	}
}

func TestEndpointRegistry_PathsPreservesInsertionOrder(t *testing.T) {
	r := security.NewEndpointRegistry()
	paths := []string{"/a", "/b", "/c"}
	for _, p := range paths {
		if err := r.Register(p, security.RiskSafeRead); err != nil {
			t.Fatal(err)
		}
	}
	got := r.Paths()
	if len(got) != len(paths) {
		t.Fatalf("got %d paths, want %d", len(got), len(paths))
	}
	for i := range paths {
		if got[i] != paths[i] {
			t.Errorf("paths[%d] = %q, want %q", i, got[i], paths[i])
		}
	}
}
