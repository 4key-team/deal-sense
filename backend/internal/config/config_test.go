package config_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("LLM_BASE_URL", "")
	t.Setenv("LLM_API_KEY", "")
	t.Setenv("LLM_MODEL", "")
	t.Setenv("LLM_SOCKS5_PROXY", "")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LLMProvider != "anthropic" {
		t.Errorf("LLMProvider = %q, want anthropic", cfg.LLMProvider)
	}
	if cfg.LLMModel != "claude-sonnet-4-5" {
		t.Errorf("LLMModel = %q, want claude-sonnet-4-5", cfg.LLMModel)
	}
	if cfg.LLMSOCKS5Proxy != "" {
		t.Errorf("LLMSOCKS5Proxy = %q, want empty", cfg.LLMSOCKS5Proxy)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_BASE_URL", "https://api.openai.com/v1")
	t.Setenv("LLM_API_KEY", "sk-test-key")
	t.Setenv("LLM_MODEL", "gpt-4o")
	t.Setenv("LLM_SOCKS5_PROXY", "socks5://127.0.0.1:1080")

	cfg := config.Load()

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
	if cfg.LLMSOCKS5Proxy != "socks5://127.0.0.1:1080" {
		t.Errorf("LLMSOCKS5Proxy = %q, want socks5://127.0.0.1:1080", cfg.LLMSOCKS5Proxy)
	}
}
