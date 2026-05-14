package telegram_test

import (
	"errors"
	"os"
	"path/filepath"
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
	if cfg.ProfileStorePath != "/data/telegram-profiles.json" {
		t.Errorf("ProfileStorePath = %q, want /data/telegram-profiles.json", cfg.ProfileStorePath)
	}
	if cfg.LLMStorePath != "/data/telegram-llm-settings.json" {
		t.Errorf("LLMStorePath = %q, want /data/telegram-llm-settings.json", cfg.LLMStorePath)
	}
}

func TestLoadConfig_RequirePerChatLLM_DefaultTrue(t *testing.T) {
	// Multi-tenant deployments: every chat MUST configure its own LLM
	// via /llm edit by default. The owner of a single-tenant deployment
	// can opt back into env fallback via ALLOW_SERVER_LLM_FALLBACK=true.
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ALLOWLIST_USER_IDS", "1")
	t.Setenv("ALLOW_SERVER_LLM_FALLBACK", "")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !cfg.RequirePerChatLLM {
		t.Error("RequirePerChatLLM = false by default, want true (BYOK is the multi-tenant default)")
	}
}

func TestLoadConfig_AllowServerLLMFallback_DisablesRequire(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want bool // RequirePerChatLLM
	}{
		{"empty → require=true (default)", "", true},
		{"explicit false → require=true", "false", true},
		{"true → require=false (legacy mode)", "true", false},
		{"1 → require=false", "1", false},
		{"yes → require=false", "yes", false},
		{"YES (case-insensitive) → require=false", "YES", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
			t.Setenv("ALLOWLIST_USER_IDS", "1")
			t.Setenv("ALLOW_SERVER_LLM_FALLBACK", tt.env)

			cfg, err := telegram.LoadConfig()
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if cfg.RequirePerChatLLM != tt.want {
				t.Errorf("RequirePerChatLLM = %v, want %v (ALLOW_SERVER_LLM_FALLBACK=%q)",
					cfg.RequirePerChatLLM, tt.want, tt.env)
			}
		})
	}
}

func TestLoadConfig_LLMStorePath_Override(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ALLOWLIST_USER_IDS", "1")
	t.Setenv("TELEGRAM_LLM_STORE_PATH", "/srv/state/llm.json")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMStorePath != "/srv/state/llm.json" {
		t.Errorf("LLMStorePath = %q, want /srv/state/llm.json", cfg.LLMStorePath)
	}
}

func TestLoadConfig_MetricsPort(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want int
	}{
		{"unset → disabled (0)", "", 0},
		{"explicit zero → disabled", "0", 0},
		{"valid port", "9091", 9091},
		{"high port", "65535", 65535},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
			t.Setenv("ALLOWLIST_USER_IDS", "1")
			t.Setenv("METRICS_PORT", tt.env)

			cfg, err := telegram.LoadConfig()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.MetricsPort != tt.want {
				t.Errorf("MetricsPort = %d, want %d", cfg.MetricsPort, tt.want)
			}
		})
	}
}

func TestLoadConfig_MetricsPort_InvalidRejected(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("ALLOWLIST_USER_IDS", "1")
	t.Setenv("METRICS_PORT", "abc")

	_, err := telegram.LoadConfig()
	if err == nil {
		t.Fatal("expected error for non-numeric METRICS_PORT, got nil")
	}
	if !errors.Is(err, telegram.ErrInvalidMetricsPort) {
		t.Errorf("err = %v, want wrapping ErrInvalidMetricsPort", err)
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

// --- *_FILE secrets pattern (12-factor) ---

func TestLoadConfig_BotTokenFromFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tg-token")
	if err := os.WriteFile(path, []byte("bot:from-file\n"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", path)

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BotToken != "bot:from-file" {
		t.Errorf("BotToken = %q, want bot:from-file (whitespace stripped)", cfg.BotToken)
	}
}

func TestLoadConfig_APIKeyFromFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "ds-key")
	if err := os.WriteFile(path, []byte("shared-from-file"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DEAL_SENSE_API_KEY", "")
	t.Setenv("DEAL_SENSE_API_KEY_FILE", path)

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "shared-from-file" {
		t.Errorf("APIKey = %q, want shared-from-file", cfg.APIKey)
	}
}

func TestLoadConfig_BotTokenFile_TakesPrecedence(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tg-token")
	if err := os.WriteFile(path, []byte("from-file"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "from-plain-env")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", path)

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BotToken != "from-file" {
		t.Errorf("BotToken = %q, want from-file (FILE wins over plain env)", cfg.BotToken)
	}
}

func TestLoadConfig_BotTokenFile_ErrorOnUnreadable(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", "/this/path/does/not/exist")

	_, err := telegram.LoadConfig()
	if err == nil {
		t.Fatal("expected error for unreadable TELEGRAM_BOT_TOKEN_FILE, got nil")
	}
	// An unreadable file must surface the underlying read error rather than
	// silently fall through to ErrMissingBotToken — operator who set *_FILE
	// expects exactly that source.
	if errors.Is(err, telegram.ErrMissingBotToken) {
		t.Errorf("err = %v, want file-read error, not ErrMissingBotToken (fall-through is unsafe)", err)
	}
}

func TestLoadConfig_APIKeyFile_ErrorOnUnreadable(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DEAL_SENSE_API_KEY", "")
	t.Setenv("DEAL_SENSE_API_KEY_FILE", "/this/path/does/not/exist")

	_, err := telegram.LoadConfig()
	if err == nil {
		t.Fatal("expected error for unreadable DEAL_SENSE_API_KEY_FILE, got nil")
	}
}

// --- BOT_CONFIG_PATH JSON overlay (UI-managed config) ---

const fixtureOverlayToken = "8829614348:AAH4OyBX8kX06aLl2DMk48Qk_2N9t5Q0bts"

// writeOverlayJSON helps tests stage a known-good config file.
func writeOverlayJSON(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "bot-config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write overlay: %v", err)
	}
	return path
}

func TestLoadConfig_BotConfigPath_Unset_UsesEnv(t *testing.T) {
	t.Setenv("BOT_CONFIG_PATH", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token:secret")
	t.Setenv("ALLOWLIST_USER_IDS", "42")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cfg.BotToken != "env-token:secret" {
		t.Errorf("BotToken = %q, want env-token:secret", cfg.BotToken)
	}
	if !reflect.DeepEqual(cfg.AllowlistUserIDs, []int64{42}) {
		t.Errorf("AllowlistUserIDs = %v, want [42]", cfg.AllowlistUserIDs)
	}
}

func TestLoadConfig_BotConfigPath_FileMissing_FallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist.json")
	t.Setenv("BOT_CONFIG_PATH", missing)
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token:secret")
	t.Setenv("ALLOWLIST_USER_IDS", "42")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cfg.BotToken != "env-token:secret" {
		t.Errorf("BotToken = %q, want env-token:secret (file missing → env)", cfg.BotToken)
	}
}

func TestLoadConfig_BotConfigPath_FileValid_OverridesEnv(t *testing.T) {
	dir := t.TempDir()
	overlay := `{
		"token": "` + fixtureOverlayToken + `",
		"allowlist_user_ids": [100, 200],
		"log_level": "warn"
	}`
	path := writeOverlayJSON(t, dir, overlay)
	t.Setenv("BOT_CONFIG_PATH", path)
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token-should-be-ignored")
	t.Setenv("ALLOWLIST_USER_IDS", "1,2,3")
	t.Setenv("LOG_LEVEL", "error")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cfg.BotToken != fixtureOverlayToken {
		t.Errorf("BotToken = %q, want JSON value %q", cfg.BotToken, fixtureOverlayToken)
	}
	if !reflect.DeepEqual(cfg.AllowlistUserIDs, []int64{100, 200}) && !reflect.DeepEqual(cfg.AllowlistUserIDs, []int64{200, 100}) {
		t.Errorf("AllowlistUserIDs = %v, want [100 200] (order tolerant)", cfg.AllowlistUserIDs)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want warn (JSON wins over env error)", cfg.LogLevel)
	}
}

func TestLoadConfig_BotConfigPath_OpenAllowlistInJSON_OverridesRestrictedEnv(t *testing.T) {
	dir := t.TempDir()
	// JSON omits allowlist_user_ids → open mode
	overlay := `{"token": "` + fixtureOverlayToken + `", "log_level": "info"}`
	path := writeOverlayJSON(t, dir, overlay)
	t.Setenv("BOT_CONFIG_PATH", path)
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token")
	// Env says restricted but JSON open should win.
	t.Setenv("ALLOWLIST_USER_IDS", "42")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(cfg.AllowlistUserIDs) != 0 {
		t.Errorf("AllowlistUserIDs = %v, want empty (JSON open mode wins)", cfg.AllowlistUserIDs)
	}
}

func TestLoadConfig_BotConfigPath_FileCorrupt_Errors(t *testing.T) {
	dir := t.TempDir()
	path := writeOverlayJSON(t, dir, "{not json")
	t.Setenv("BOT_CONFIG_PATH", path)
	t.Setenv("TELEGRAM_BOT_TOKEN", "env-token") // would normally succeed

	_, err := telegram.LoadConfig()
	if err == nil {
		t.Fatal("expected error for corrupt JSON overlay")
	}
}

func TestLoadConfig_BotConfigPath_PreservesEnvOnlyFields(t *testing.T) {
	// JSON overlay covers token/allowlist/log_level — operational fields
	// (API_BASE_URL, METRICS_PORT, profile store path) must still come
	// from env even when the overlay is present.
	dir := t.TempDir()
	overlay := `{"token": "` + fixtureOverlayToken + `", "log_level": "info"}`
	path := writeOverlayJSON(t, dir, overlay)
	t.Setenv("BOT_CONFIG_PATH", path)
	t.Setenv("API_BASE_URL", "http://custom-backend:9999")
	t.Setenv("METRICS_PORT", "7777")
	t.Setenv("TELEGRAM_PROFILE_STORE_PATH", "/custom/profiles.json")

	cfg, err := telegram.LoadConfig()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if cfg.APIBaseURL != "http://custom-backend:9999" {
		t.Errorf("APIBaseURL = %q, want from env", cfg.APIBaseURL)
	}
	if cfg.MetricsPort != 7777 {
		t.Errorf("MetricsPort = %d, want 7777 from env", cfg.MetricsPort)
	}
	if cfg.ProfileStorePath != "/custom/profiles.json" {
		t.Errorf("ProfileStorePath = %q, want from env", cfg.ProfileStorePath)
	}
}

func TestLoadConfig_BotTokenFile_EmptyAfterTrim(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "tg-token")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN_FILE", path)

	_, err := telegram.LoadConfig()
	if !errors.Is(err, telegram.ErrMissingBotToken) {
		t.Fatalf("err = %v, want %v (whitespace-only file = missing token)", err, telegram.ErrMissingBotToken)
	}
}
