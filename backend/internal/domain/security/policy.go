// Package security defines the SecurityPolicy domain invariant for LLM
// system prompts. Source: reflective-agent-defaults v1.4 Rule 4 v1.4.
// The policy enforces six directives, of which directive 4 (FACTUAL
// INTEGRITY) is the юр firewall — primary risk for Deal Sense.
//
// Stub — real implementation lands in the GREEN commit.
package security

import "errors"

// ErrEmptyDirectives indicates the directives text is empty or whitespace-only.
var ErrEmptyDirectives = errors.New("security: directives text is empty")

// ErrMissingMarker indicates a required security marker is missing from the
// directives text. Triggers the юр firewall guard against silent removal.
var ErrMissingMarker = errors.New("security: required marker missing in directives")

// Policy is the security invariant wrapping all LLM system prompts.
type Policy struct{}

// NewPolicy validates the directives text and returns a Policy. Returns
// ErrEmptyDirectives or ErrMissingMarker on invariant violation.
func NewPolicy(text string) (*Policy, error) {
	return nil, ErrEmptyDirectives
}

// NewDefaultPolicy returns a Policy using the embedded directives file.
func NewDefaultPolicy() (*Policy, error) {
	return nil, ErrEmptyDirectives
}

// Prefix returns the validated security prefix to prepend to LLM prompts.
func (p *Policy) Prefix() string {
	return ""
}

// Wrap returns a new prompt function that prepends the policy prefix.
func (p *Policy) Wrap(prompt func(string) string) func(string) string {
	return prompt
}
