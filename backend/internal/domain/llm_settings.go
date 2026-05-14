package domain

import (
	"net/url"
	"strings"
)

// LLMSettings is a chat-scoped LLM provider configuration: which API to
// call (provider + base_url), what credential to use (api_key) and which
// model to request. Each value is validated at construction time so the
// transport layer can trust the invariants without re-checking.
type LLMSettings struct {
	provider string
	baseURL  string // optional; "" means "use provider default"
	apiKey   string
	model    string
}

// NewLLMSettings validates the inputs and returns a populated *LLMSettings.
// Empty / whitespace-only provider, api_key or model return field-specific
// errors. base_url is optional, but when supplied must parse as an absolute
// URL with a scheme and host.
func NewLLMSettings(provider, baseURL, apiKey, model string) (*LLMSettings, error) {
	p := strings.TrimSpace(provider)
	if p == "" {
		return nil, ErrEmptyLLMProvider
	}
	k := strings.TrimSpace(apiKey)
	if k == "" {
		return nil, ErrEmptyLLMAPIKey
	}
	m := strings.TrimSpace(model)
	if m == "" {
		return nil, ErrEmptyLLMModel
	}
	u := strings.TrimSpace(baseURL)
	if u != "" {
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, ErrInvalidLLMBaseURL
		}
	}
	return &LLMSettings{
		provider: p,
		baseURL:  u,
		apiKey:   k,
		model:    m,
	}, nil
}

// Provider returns the validated provider ID (e.g. "openai", "anthropic").
func (s *LLMSettings) Provider() string { return s.provider }

// BaseURL returns the validated base URL or "" when the caller should use
// the provider's default endpoint.
func (s *LLMSettings) BaseURL() string { return s.baseURL }

// APIKey returns the raw API key. Adapters that surface settings to a UI
// or to logs should prefer MaskedAPIKey to avoid leaking the secret.
func (s *LLMSettings) APIKey() string { return s.apiKey }

// Model returns the validated model identifier.
func (s *LLMSettings) Model() string { return s.model }

// MaskedAPIKey returns a redacted form of the key for UI/logs: the last
// 4 characters are kept for fingerprinting, the rest is asterisks. Keys
// shorter than 5 characters are fully masked (no fingerprint).
func (s *LLMSettings) MaskedAPIKey() string {
	if len(s.apiKey) <= 4 {
		return strings.Repeat("*", len(s.apiKey))
	}
	keep := s.apiKey[len(s.apiKey)-4:]
	return strings.Repeat("*", len(s.apiKey)-4) + keep
}
