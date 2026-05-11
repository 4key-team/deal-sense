package security

import (
	"errors"
	"fmt"
)

// ErrInvalidRiskLevel indicates a string that is not one of the allowed
// risk-level tokens was passed to NewRiskLevel.
var ErrInvalidRiskLevel = errors.New("security: invalid risk level")

// ErrUnannotatedEndpoint indicates an HTTP path was queried in the
// EndpointRegistry that wasn't registered with a risk level.
var ErrUnannotatedEndpoint = errors.New("security: endpoint missing risk annotation")

// ErrDuplicateEndpoint indicates Register was called twice for the same path.
var ErrDuplicateEndpoint = errors.New("security: endpoint already registered")

// RiskLevel classifies how dangerous an endpoint is. Source:
// reflective-agent-defaults v1.4 §Layer 4 / Rule 11 — coupling rule action
// to its risk class.
//
//   - SAFE_READ:    pure reads (list providers, analyze without persistence)
//   - MODIFY:       creates artifacts (generate proposal — bytes in response,
//                   no external delivery)
//   - DESTRUCTIVE:  external delivery / mutation that cannot be undone
//                   (send proposal to client, change tender status). None
//                   exist in the current codebase; the type is in place so
//                   future endpoints land annotated from day one.
//
// Construct via NewRiskLevel — the zero value (empty string) is invalid.
type RiskLevel string

const (
	RiskSafeRead    RiskLevel = "SAFE_READ"
	RiskModify      RiskLevel = "MODIFY"
	RiskDestructive RiskLevel = "DESTRUCTIVE"
)

// NewRiskLevel validates the supplied string and returns the typed value.
// Stub for the RED step — always returns ErrInvalidRiskLevel so behavioural
// tests fail at runtime.
func NewRiskLevel(s string) (RiskLevel, error) {
	return "", ErrInvalidRiskLevel
}

// String returns the canonical token for the level.
func (r RiskLevel) String() string {
	return string(r)
}

// EndpointRegistry maps HTTP paths to risk levels. Construct via
// NewEndpointRegistry and Register; queries use Lookup. Registry is the
// machine-readable equivalent of an ops runbook — Layer 4 coupling tests
// assert every wired route appears here.
type EndpointRegistry struct{}

// NewEndpointRegistry returns an empty registry. Stub for the RED step.
func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{}
}

// Register stores a risk level for the given path. Stub returns nil so
// tests fall through to Lookup misses.
func (r *EndpointRegistry) Register(path string, level RiskLevel) error {
	return nil
}

// Lookup returns the risk level for path. Stub always returns
// ErrUnannotatedEndpoint.
func (r *EndpointRegistry) Lookup(path string) (RiskLevel, error) {
	return "", fmt.Errorf("%w: %q", ErrUnannotatedEndpoint, path)
}

// Paths returns all registered paths in insertion order. Stub returns nil.
func (r *EndpointRegistry) Paths() []string {
	return nil
}
