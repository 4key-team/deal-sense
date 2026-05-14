package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

// Realistic Telegram token shape: numeric bot ID, colon, 35-char secret made
// of [A-Za-z0-9_-].
const fixtureValidToken = "8829614348:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"

// --- Token validation ------------------------------------------------------

func TestNewBotConfig_InvalidToken_ReturnsErrInvalidBotToken(t *testing.T) {
	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"no colon", "12345abcdefghijklmnopqrstuvwxyz"},
		{"no id part", ":AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"},
		{"no secret part", "8829614348:"},
		{"non-digit id", "abc123:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"},
		{"secret too short", "8829614348:short"},
		{"forbidden char in secret", "8829614348:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bt!"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewBotConfig(tc.token, []int64{42}, "info")
			if !errors.Is(err, domain.ErrInvalidBotToken) {
				t.Fatalf("err = %v, want wrapping ErrInvalidBotToken", err)
			}
		})
	}
}

// --- Allowlist composition (delegates to auth.ParseAllowlist) -------------

func TestNewBotConfig_EmptyAllowlist_OpensInOpenMode(t *testing.T) {
	// Empty allowlist is no longer an error — BotConfig composes
	// auth.ParseAllowlist, which yields an open allowlist for bootstrap.
	cfg, err := domain.NewBotConfig(fixtureValidToken, nil, "info")
	if err != nil {
		t.Fatalf("NewBotConfig with empty allowlist must not error: %v", err)
	}
	if !cfg.Allowlist().IsOpen() {
		t.Error("empty allowlist input must yield open mode")
	}
	if !cfg.Allowlist().IsAllowed(99999) {
		t.Error("open allowlist must admit any user ID")
	}
}

func TestNewBotConfig_InvalidUserID_ReturnsErrInvalidUserID(t *testing.T) {
	// Non-empty list with a bad ID still errors out.
	_, err := domain.NewBotConfig(fixtureValidToken, []int64{0, -1, 42}, "info")
	if !errors.Is(err, auth.ErrInvalidUserID) {
		t.Fatalf("err = %v, want wrapping auth.ErrInvalidUserID", err)
	}
}

func TestNewBotConfig_ValidIDs_RestrictedMode(t *testing.T) {
	cfg, err := domain.NewBotConfig(fixtureValidToken, []int64{42, 100}, "info")
	if err != nil {
		t.Fatalf("NewBotConfig: %v", err)
	}
	if cfg.Allowlist().IsOpen() {
		t.Error("non-empty IDs must yield restricted mode")
	}
	if !cfg.Allowlist().IsAllowed(42) {
		t.Error("Allowlist should admit 42")
	}
	if cfg.Allowlist().IsAllowed(999) {
		t.Error("Allowlist should not admit 999")
	}
}

// --- Log level validation --------------------------------------------------

func TestNewBotConfig_InvalidLogLevel_ReturnsErrInvalidLogLevel(t *testing.T) {
	cases := []struct {
		name  string
		level string
	}{
		{"empty", ""},
		{"unknown", "verbose"},
		{"mixed case typo", "INFOO"},
		{"numeric", "5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewBotConfig(fixtureValidToken, []int64{42}, tc.level)
			if !errors.Is(err, domain.ErrInvalidLogLevel) {
				t.Fatalf("err = %v, want wrapping ErrInvalidLogLevel", err)
			}
		})
	}
}

func TestNewBotConfig_LogLevelCaseInsensitive(t *testing.T) {
	cases := []struct {
		input string
		want  domain.LogLevel
	}{
		{"debug", domain.LogLevelDebug},
		{"DEBUG", domain.LogLevelDebug},
		{"  Info  ", domain.LogLevelInfo},
		{"WARN", domain.LogLevelWarn},
		{"error", domain.LogLevelError},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			cfg, err := domain.NewBotConfig(fixtureValidToken, []int64{42}, tc.input)
			if err != nil {
				t.Fatalf("NewBotConfig: %v", err)
			}
			if got := cfg.LogLevel(); got != tc.want {
				t.Errorf("LogLevel() = %q, want %q", got, tc.want)
			}
		})
	}
}

// --- Accessors -------------------------------------------------------------

func TestBotConfig_Accessors(t *testing.T) {
	cfg, err := domain.NewBotConfig(fixtureValidToken, []int64{42, 100, 42}, "warn")
	if err != nil {
		t.Fatalf("NewBotConfig: %v", err)
	}
	if got := cfg.Token(); got != fixtureValidToken {
		t.Errorf("Token() = %q, want %q", got, fixtureValidToken)
	}
	if got := cfg.LogLevel(); got != domain.LogLevelWarn {
		t.Errorf("LogLevel() = %q, want warn", got)
	}
	if cfg.Allowlist() == nil {
		t.Fatal("Allowlist() returned nil")
	}
}

// --- MaskedToken -----------------------------------------------------------

func TestBotConfig_MaskedToken_HidesSecret(t *testing.T) {
	cfg, err := domain.NewBotConfig(fixtureValidToken, []int64{42}, "info")
	if err != nil {
		t.Fatalf("NewBotConfig: %v", err)
	}
	masked := cfg.MaskedToken()
	if masked == fixtureValidToken {
		t.Fatal("MaskedToken returned the raw token")
	}
	// Bot ID before the colon is kept for operator recognition.
	if !strings.HasPrefix(masked, "8829614348:") {
		t.Errorf("MaskedToken = %q, want prefix %q", masked, "8829614348:")
	}
	// Last 4 chars of the secret remain for fingerprinting.
	if !strings.HasSuffix(masked, "0bts") {
		t.Errorf("MaskedToken = %q, want suffix %q", masked, "0bts")
	}
	// The middle of the secret must be redacted with bullets/asterisks —
	// raw secret characters must NOT survive.
	rawSecretMiddle := "AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q"
	if strings.Contains(masked, rawSecretMiddle) {
		t.Errorf("MaskedToken = %q, must redact secret body", masked)
	}
}

// --- LogLevel value object behaviour --------------------------------------

func TestLogLevel_String_RoundTrips(t *testing.T) {
	cases := []struct {
		level domain.LogLevel
		want  string
	}{
		{domain.LogLevelDebug, "debug"},
		{domain.LogLevelInfo, "info"},
		{domain.LogLevelWarn, "warn"},
		{domain.LogLevelError, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.level.String(); got != tc.want {
				t.Errorf("LogLevel(%q).String() = %q, want %q", tc.level, got, tc.want)
			}
		})
	}
}
