// Package telegram is the Telegram bot adapter: config, handlers, and the
// runtime wiring for the cmd/telegram-bot frontend.
package telegram

import "errors"

// ErrMissingBotToken indicates TELEGRAM_BOT_TOKEN was empty or unset.
var ErrMissingBotToken = errors.New("telegram: TELEGRAM_BOT_TOKEN is required")

// ErrInvalidAllowlistID indicates ALLOWLIST_USER_IDS contained a non-integer
// entry. Wrap the underlying parse error for diagnostics.
var ErrInvalidAllowlistID = errors.New("telegram: invalid ALLOWLIST_USER_IDS entry")

// Config carries deployment knobs for the Telegram bot. It is a DTO; load
// it via LoadConfig which reads from environment variables and validates.
type Config struct {
	BotToken         string
	AllowlistUserIDs []int64
	APIBaseURL       string
	APIKey           string
	LogLevel         string
}

// LoadConfig reads bot configuration from environment variables. Stub: this
// is the RED step — it returns a zero Config, failing the runtime
// assertions in the accompanying tests.
func LoadConfig() (Config, error) {
	return Config{}, nil
}
