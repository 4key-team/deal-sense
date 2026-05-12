package config

import (
	"cmp"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	LLMProvider string
	LLMBaseURL  string
	LLMAPIKey   string
	LLMModel    string
	LogLevel    string
	APIKey      string
	// RateLimitRPS is the per-IP requests-per-second cap. Zero disables.
	RateLimitRPS float64
	// RateLimitBurst is the bucket size for the per-IP limiter.
	RateLimitBurst int
}

// Load reads configuration from environment variables. Secrets (LLM_API_KEY,
// DEAL_SENSE_API_KEY) additionally honour the 12-factor `<NAME>_FILE` pattern:
// when set, the secret is read from the file path (whitespace-trimmed).
// Returns an error when a `<NAME>_FILE` is set but unreadable.
func Load() (Config, error) {
	llmAPIKey, err := readSecret("LLM_API_KEY")
	if err != nil {
		return Config{}, err
	}
	apiKey, err := readSecret("DEAL_SENSE_API_KEY")
	if err != nil {
		return Config{}, err
	}
	return Config{
		Port:           cmp.Or(os.Getenv("PORT"), "8080"),
		LLMProvider:    cmp.Or(os.Getenv("LLM_PROVIDER"), "anthropic"),
		LLMBaseURL:     os.Getenv("LLM_BASE_URL"),
		LLMAPIKey:      llmAPIKey,
		LLMModel:       cmp.Or(os.Getenv("LLM_MODEL"), "claude-sonnet-4-5"),
		LogLevel:       strings.ToLower(cmp.Or(os.Getenv("LOG_LEVEL"), "info")),
		APIKey:         apiKey,
		RateLimitRPS:   parseFloat(os.Getenv("RATE_LIMIT_RPS"), 0),
		RateLimitBurst: parseInt(os.Getenv("RATE_LIMIT_BURST"), 30),
	}, nil
}

// readSecret resolves a secret from `<NAME>_FILE` (preferred) or plain env
// `<NAME>`. The file path is read, whitespace-trimmed, and returned. An
// unreadable `<NAME>_FILE` is a fatal config error — callers should fail
// startup rather than silently fall back to plain env (an operator who set
// `<NAME>_FILE` expects exactly that source).
func readSecret(name string) (string, error) {
	if path := os.Getenv(name + "_FILE"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("config: read %s_FILE %q: %w", name, path, err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return os.Getenv(name), nil
}

func parseFloat(s string, fallback float64) float64 {
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fallback
	}
	return v
}

func parseInt(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
