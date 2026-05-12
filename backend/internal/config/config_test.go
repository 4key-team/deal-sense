package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/config"
)

func TestLoad_LogLevel(t *testing.T) {
	tests := []struct {
		env  string
		want string
	}{
		{"", "info"},
		{"debug", "debug"},
		{"warn", "warn"},
		{"error", "error"},
		{"DEBUG", "debug"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", tt.env)
			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.LogLevel != tt.want {
				t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, tt.want)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LLMProvider != "anthropic" {
		t.Errorf("LLMProvider = %q, want anthropic", cfg.LLMProvider)
	}
	if cfg.LLMModel != "claude-sonnet-4-5" {
		t.Errorf("LLMModel = %q, want claude-sonnet-4-5", cfg.LLMModel)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("LLM_API_KEY", "sk-test-key")
	t.Setenv("LLM_MODEL", "gpt-4o")
	t.Setenv("DEAL_SENSE_API_KEY", "deal-sense-secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
	if cfg.LLMProvider != "openai" {
		t.Errorf("LLMProvider = %q, want openai", cfg.LLMProvider)
	}
	if cfg.LLMBaseURL != "https://api.openai.com/v1" {
		t.Errorf("LLMBaseURL = %q, want https://api.openai.com/v1", cfg.LLMBaseURL)
	}
	if cfg.LLMAPIKey != "sk-test-key" {
		t.Errorf("LLMAPIKey = %q, want sk-test-key", cfg.LLMAPIKey)
	}
	if cfg.LLMModel != "gpt-4o" {
		t.Errorf("LLMModel = %q, want gpt-4o", cfg.LLMModel)
	}
	if cfg.APIKey != "deal-sense-secret" {
		t.Errorf("APIKey = %q, want deal-sense-secret", cfg.APIKey)
	}
}

func TestLoad_APIKeyEmptyByDefault(t *testing.T) {
	t.Setenv("DEAL_SENSE_API_KEY", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty (open access by default)", cfg.APIKey)
	}
}

// --- *_FILE secrets pattern (12-factor) ---

func TestLoad_SecretFromFile_LLMAPIKey(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "llm-key")
	if err := os.WriteFile(path, []byte("sk-from-file"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_API_KEY_FILE", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMAPIKey != "sk-from-file" {
		t.Errorf("LLMAPIKey = %q, want sk-from-file", cfg.LLMAPIKey)
	}
}

func TestLoad_SecretFromFile_DealSenseAPIKey(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "ds-key")
	if err := os.WriteFile(path, []byte("deal-sense-secret"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("DEAL_SENSE_API_KEY", "")
	t.Setenv("DEAL_SENSE_API_KEY_FILE", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "deal-sense-secret" {
		t.Errorf("APIKey = %q, want deal-sense-secret", cfg.APIKey)
	}
}

func TestLoad_SecretFromFile_TrimsWhitespace(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "llm-key")
	if err := os.WriteFile(path, []byte("  sk-trimmed  \n"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_API_KEY_FILE", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMAPIKey != "sk-trimmed" {
		t.Errorf("LLMAPIKey = %q, want sk-trimmed (whitespace stripped)", cfg.LLMAPIKey)
	}
}

func TestLoad_SecretFromFile_TakesPrecedenceOverPlainEnv(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "llm-key")
	if err := os.WriteFile(path, []byte("from-file"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("LLM_API_KEY", "from-plain-env")
	t.Setenv("LLM_API_KEY_FILE", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMAPIKey != "from-file" {
		t.Errorf("LLMAPIKey = %q, want from-file (FILE wins over plain env)", cfg.LLMAPIKey)
	}
}

func TestLoad_SecretFromFile_ErrorOnUnreadable(t *testing.T) {
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_API_KEY_FILE", "/this/path/does/not/exist")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for unreadable *_FILE, got nil")
	}
}

func TestLoad_PlainEnvFallback_NoFile(t *testing.T) {
	t.Setenv("LLM_API_KEY", "from-plain")
	t.Setenv("LLM_API_KEY_FILE", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMAPIKey != "from-plain" {
		t.Errorf("LLMAPIKey = %q, want from-plain (backward-compat)", cfg.LLMAPIKey)
	}
}
