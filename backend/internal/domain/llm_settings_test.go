package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// --- Required fields -------------------------------------------------------

func TestNewLLMSettings_EmptyProvider_ReturnsErrEmptyLLMProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"tab", "\t"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewLLMSettings(tc.provider, "", "sk-abc", "gpt-4o")
			if !errors.Is(err, domain.ErrEmptyLLMProvider) {
				t.Fatalf("err = %v, want wrapping ErrEmptyLLMProvider", err)
			}
		})
	}
}

func TestNewLLMSettings_EmptyAPIKey_ReturnsErrEmptyLLMAPIKey(t *testing.T) {
	_, err := domain.NewLLMSettings("openai", "", "   ", "gpt-4o")
	if !errors.Is(err, domain.ErrEmptyLLMAPIKey) {
		t.Fatalf("err = %v, want wrapping ErrEmptyLLMAPIKey", err)
	}
}

func TestNewLLMSettings_EmptyModel_ReturnsErrEmptyLLMModel(t *testing.T) {
	_, err := domain.NewLLMSettings("openai", "", "sk-abc", "")
	if !errors.Is(err, domain.ErrEmptyLLMModel) {
		t.Fatalf("err = %v, want wrapping ErrEmptyLLMModel", err)
	}
}

// --- Optional base URL with validation ------------------------------------

func TestNewLLMSettings_BaseURL_OptionalEmpty_OK(t *testing.T) {
	cfg, err := domain.NewLLMSettings("openai", "", "sk-abc", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.BaseURL() != "" {
		t.Errorf("BaseURL = %q, want empty (provider default)", cfg.BaseURL())
	}
}

func TestNewLLMSettings_BaseURL_Invalid_ReturnsErrInvalidLLMBaseURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"no scheme", "openrouter.ai/api/v1"},
		{"only path", "/api/v1"},
		{"scheme without host", "https://"},
		{"garbage", "not a url"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewLLMSettings("openai", tc.url, "sk-abc", "gpt-4o")
			if !errors.Is(err, domain.ErrInvalidLLMBaseURL) {
				t.Fatalf("err = %v, want wrapping ErrInvalidLLMBaseURL", err)
			}
		})
	}
}

func TestNewLLMSettings_BaseURL_Valid_OK(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"https", "https://openrouter.ai/api/v1"},
		{"http localhost", "http://localhost:11434"},
		{"with trailing slash", "https://api.anthropic.com/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := domain.NewLLMSettings("openai", tc.url, "sk-abc", "gpt-4o")
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if cfg.BaseURL() == "" {
				t.Errorf("BaseURL must round-trip, got empty for input %q", tc.url)
			}
		})
	}
}

// --- Accessors -------------------------------------------------------------

func TestNewLLMSettings_TrimsWhitespace(t *testing.T) {
	cfg, err := domain.NewLLMSettings("  openai ", "  ", "  sk-abc  ", "  gpt-4o  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.Provider() != "openai" {
		t.Errorf("Provider = %q, want trimmed", cfg.Provider())
	}
	if cfg.APIKey() != "sk-abc" {
		t.Errorf("APIKey = %q, want trimmed", cfg.APIKey())
	}
	if cfg.Model() != "gpt-4o" {
		t.Errorf("Model = %q, want trimmed", cfg.Model())
	}
}

// --- MaskedAPIKey ---------------------------------------------------------

func TestLLMSettings_MaskedAPIKey_HidesSecretBody(t *testing.T) {
	cfg, err := domain.NewLLMSettings("openai", "", "sk-1234567890abcdefABCDEFwxyz", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	masked := cfg.MaskedAPIKey()
	if masked == cfg.APIKey() {
		t.Fatal("MaskedAPIKey returned the raw key")
	}
	// Last 4 chars kept for fingerprinting.
	if !strings.HasSuffix(masked, "wxyz") {
		t.Errorf("MaskedAPIKey = %q, want suffix wxyz", masked)
	}
	// Raw middle must not leak.
	if strings.Contains(masked, "1234567890abcdef") {
		t.Errorf("MaskedAPIKey leaked secret body: %q", masked)
	}
}

func TestLLMSettings_MaskedAPIKey_ShortKey_FullyMasked(t *testing.T) {
	cfg, err := domain.NewLLMSettings("openai", "", "abcd", "gpt-4o")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	masked := cfg.MaskedAPIKey()
	if masked == "abcd" {
		t.Errorf("short key must not leak, got %q", masked)
	}
}
