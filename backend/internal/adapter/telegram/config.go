// Package telegram is the Telegram bot adapter: config, handlers, and the
// runtime wiring for the cmd/telegram-bot frontend.
package telegram

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

// LoadConfig reads bot configuration from environment variables and returns
// ErrMissingBotToken / ErrInvalidAllowlistID for malformed input.
func LoadConfig() (Config, error) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, ErrMissingBotToken
	}

	ids, err := parseAllowlist(os.Getenv("ALLOWLIST_USER_IDS"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		BotToken:         token,
		AllowlistUserIDs: ids,
		APIBaseURL:       cmp.Or(strings.TrimSpace(os.Getenv("API_BASE_URL")), "http://localhost:8080"),
		APIKey:           os.Getenv("DEAL_SENSE_API_KEY"),
		LogLevel:         strings.ToLower(cmp.Or(strings.TrimSpace(os.Getenv("LOG_LEVEL")), "info")),
	}, nil
}

func parseAllowlist(raw string) ([]int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var ids []int64
	for part := range strings.SplitSeq(raw, ",") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", ErrInvalidAllowlistID, s)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
