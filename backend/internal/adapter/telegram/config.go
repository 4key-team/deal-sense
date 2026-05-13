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
	ProfileStorePath string
}

// LoadConfig reads bot configuration from environment variables and returns
// ErrMissingBotToken / ErrInvalidAllowlistID for malformed input. Secrets
// (TELEGRAM_BOT_TOKEN, DEAL_SENSE_API_KEY) additionally honour the
// `<NAME>_FILE` 12-factor pattern; an unreadable *_FILE fails startup.
func LoadConfig() (Config, error) {
	token, err := readSecret("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return Config{}, err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return Config{}, ErrMissingBotToken
	}

	ids, err := parseAllowlist(os.Getenv("ALLOWLIST_USER_IDS"))
	if err != nil {
		return Config{}, err
	}

	apiKey, err := readSecret("DEAL_SENSE_API_KEY")
	if err != nil {
		return Config{}, err
	}

	return Config{
		BotToken:         token,
		AllowlistUserIDs: ids,
		APIBaseURL:       cmp.Or(strings.TrimSpace(os.Getenv("API_BASE_URL")), "http://localhost:8080"),
		APIKey:           apiKey,
		LogLevel:         strings.ToLower(cmp.Or(strings.TrimSpace(os.Getenv("LOG_LEVEL")), "info")),
		ProfileStorePath: cmp.Or(strings.TrimSpace(os.Getenv("TELEGRAM_PROFILE_STORE_PATH")), "/data/telegram-profiles.json"),
	}, nil
}

// readSecret resolves a secret from `<NAME>_FILE` (preferred) or plain env
// `<NAME>`. File content is whitespace-trimmed. An unreadable `<NAME>_FILE`
// returns a wrapped error so LoadConfig fails the bot at startup — an
// operator who set *_FILE expects exactly that source and should not get a
// silent fall-through to plain env.
func readSecret(name string) (string, error) {
	if path := os.Getenv(name + "_FILE"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("telegram: read %s_FILE %q: %w", name, path, err)
		}
		return strings.TrimSpace(string(raw)), nil
	}
	return os.Getenv(name), nil
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
