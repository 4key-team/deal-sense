package telegram_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/telegram"
)

func TestLoadConfig_MissingBotToken(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	_, err := telegram.LoadConfig()
	if !errors.Is(err, telegram.ErrMissingBotToken) {
		t.Fatalf("err = %v, want %v", err, telegram.ErrMissingBotToken)
	}
}

func TestLoadConfig_AllowlistParsing(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []int64
	}{
		{"empty", "", nil},
		{"single", "123", []int64{123}},
		{"multiple", "123,456,789", []int64{123, 456, 789}},
		{"whitespace tolerated", " 123 , 456 ", []int64{123, 456}},
		{"trailing comma", "123,456,", []int64{123, 456}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
			t.Setenv("ALLOWLIST_USER_IDS", tt.env)

			cfg, err := telegram.LoadConfig()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(cfg.AllowlistUserIDs, tt.want) {
				t.Errorf("AllowlistUserIDs = %v, want %v", cfg.AllowlistUserIDs, tt.want)
			}
		})
	}
}

func TestLoadConfig_AllowlistInvalid(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ALLOWLIST_USER_IDS", "123,abc,456")

	_, err := telegram.LoadConfig()
	if !errors.Is(err, telegram.ErrInvalidAllowlistID) {
		t.Errorf("err = %v, want wrapping %v", err, telegram.ErrInvalidAllowlistID)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ALLOWLIST_USER_IDS", "")
	t.Setenv("API_BASE_URL", "")
	t.Setenv("DEAL_SENSE_API_KEY", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BotToken != "test-token" {
		t.Errorf("BotToken = %q, want test-token", cfg.BotToken)
	}
	if cfg.APIBaseURL != "http://localhost:8080" {
		t.Errorf("APIBaseURL = %q, want http://localhost:8080", cfg.APIBaseURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "bot:secret")
	t.Setenv("ALLOWLIST_USER_IDS", "42")
	t.Setenv("API_BASE_URL", "http://backend:8080")
	t.Setenv("DEAL_SENSE_API_KEY", "shared-key")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BotToken != "bot:secret" {
		t.Errorf("BotToken = %q", cfg.BotToken)
	}
	if !reflect.DeepEqual(cfg.AllowlistUserIDs, []int64{42}) {
		t.Errorf("AllowlistUserIDs = %v", cfg.AllowlistUserIDs)
	}
	if cfg.APIBaseURL != "http://backend:8080" {
		t.Errorf("APIBaseURL = %q", cfg.APIBaseURL)
	}
	if cfg.APIKey != "shared-key" {
		t.Errorf("APIKey = %q", cfg.APIKey)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}
