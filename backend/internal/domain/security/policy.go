// Package security defines the SecurityPolicy domain invariant for LLM
// system prompts. Source: reflective-agent-defaults v1.4 Rule 4 v1.4.
//
// The policy enforces six directives; directive 4 (FACTUAL INTEGRITY) is
// the юр firewall — primary risk for Deal Sense, where a wrong legal
// answer = legal liability for the company.
//
// Adapters MUST wrap every LLM system prompt through Policy.Wrap so the
// firewall cannot be silently bypassed by adding a new prompt and forgetting
// the prefix.
package security

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed security_directives.md
var embeddedDirectives string

// ErrEmptyDirectives indicates the directives text is empty or whitespace-only.
var ErrEmptyDirectives = errors.New("security: directives text is empty")

// ErrMissingMarker indicates a required security marker is missing from the
// directives text. Guards the юр firewall against silent removal.
var ErrMissingMarker = errors.New("security: required marker missing in directives")

// requiredMarkers are the substrings every valid directives text must contain.
// Each represents one of the six directives or a critical phrase within.
var requiredMarkers = []string{
	"STRICT DOMAIN FOCUS",
	"ENCODED PAYLOAD ISOLATION",
	"NO CYBERATTACKS",
	"FACTUAL INTEGRITY",
	"RESOURCE ABUSE",
	"Обратитесь к юристу компании",
}

// Policy is the security invariant wrapping all LLM system prompts.
// Construct via NewPolicy or NewDefaultPolicy — the zero value is invalid.
type Policy struct {
	prefix string
}

// NewPolicy validates the directives text and returns a Policy. Returns
// ErrEmptyDirectives if text is empty/whitespace, or ErrMissingMarker
// wrapped with the missing token if a required marker is absent.
func NewPolicy(text string) (*Policy, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrEmptyDirectives
	}
	for _, m := range requiredMarkers {
		if !strings.Contains(text, m) {
			return nil, fmt.Errorf("%w: %q", ErrMissingMarker, m)
		}
	}
	return &Policy{prefix: text}, nil
}

// NewDefaultPolicy returns a Policy backed by the embedded directives file
// (security_directives.md). Fails with ErrEmptyDirectives or ErrMissingMarker
// if the embedded file is malformed — this is a build-time guarantee in
// practice, but the error surface is preserved for defence in depth.
func NewDefaultPolicy() (*Policy, error) {
	return NewPolicy(embeddedDirectives)
}

// Prefix returns the validated security prefix to prepend to LLM prompts.
func (p *Policy) Prefix() string {
	return p.prefix
}

// WrappedPrompt is a security-wrapped LLM system prompt. The only way to
// construct one is via Policy.Wrap, which guarantees the юр firewall prefix
// is prepended to every invocation. Callers retrieve the rendered prompt
// via Call(lang).
//
// The type is intentionally NOT a public alias for func(string) string —
// the compiler must refuse any code path that returns a raw inner prompt
// where a WrappedPrompt is expected. This forecloses an entire class of
// bugs (forgot to wrap; double-wrap by accident) at the type system level,
// not at runtime.
type WrappedPrompt struct {
	fn func(lang string) string
}

// Call renders the wrapped prompt for the given language.
func (w WrappedPrompt) Call(lang string) string {
	return w.fn(lang)
}

// Wrap composes raw with the policy prefix and returns a typed WrappedPrompt.
// The returned value cannot be confused with a raw prompt at compile time —
// passing the result back into Wrap will not type-check because Wrap accepts
// func(string) string, not WrappedPrompt.
func (p *Policy) Wrap(raw func(lang string) string) WrappedPrompt {
	return WrappedPrompt{
		fn: func(lang string) string {
			return p.prefix + raw(lang)
		},
	}
}
