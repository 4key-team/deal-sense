// Package telegram is the Telegram bot adapter: config, handlers, and the
// runtime wiring for the cmd/telegram-bot frontend.
package telegram

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/adapter/botconfigstore"
)

// ErrMissingBotToken indicates TELEGRAM_BOT_TOKEN was empty or unset.
var ErrMissingBotToken = errors.New("telegram: TELEGRAM_BOT_TOKEN is required")

// ErrInvalidAllowlistID indicates ALLOWLIST_USER_IDS contained a non-integer
// entry. Wrap the underlying parse error for diagnostics.
var ErrInvalidAllowlistID = errors.New("telegram: invalid ALLOWLIST_USER_IDS entry")

// ErrInvalidMetricsPort indicates METRICS_PORT was not a valid integer.
var ErrInvalidMetricsPort = errors.New("telegram: invalid METRICS_PORT")

// Config carries deployment knobs for the Telegram bot. It is a DTO; load
// it via LoadConfig which reads from environment variables and validates.
type Config struct {
	BotToken         string
	AllowlistUserIDs []int64
	APIBaseURL       string
	APIKey           string
	LogLevel         string
	ProfileStorePath string
	LLMStorePath     string
	MetricsPort      int // 0 disables the /metrics + /healthz listener.
}

// LoadConfig reads bot configuration. Precedence rules:
//  1. If BOT_CONFIG_PATH is set and the file exists and parses, the file's
//     token / allowlist / log_level OVERRIDE the corresponding env vars.
//     This is the path the admin UI /settings writes to.
//  2. Otherwise (path unset or file absent) all three fields come from env.
//  3. Corrupt JSON at BOT_CONFIG_PATH is a fatal error — operator intent
//     unclear, no silent fall-through.
//
// Operational fields (API_BASE_URL, METRICS_PORT, TELEGRAM_PROFILE_STORE_PATH,
// DEAL_SENSE_API_KEY) are infra concerns and ALWAYS come from env, regardless
// of the overlay.
//
// Secrets honour the `<NAME>_FILE` 12-factor pattern; an unreadable *_FILE
// fails startup.
func LoadConfig() (Config, error) {
	overlay, hasOverlay, err := loadJSONOverlay()
	if err != nil {
		return Config{}, err
	}

	token, ids, logLevel, err := mergeBotFields(overlay, hasOverlay)
	if err != nil {
		return Config{}, err
	}

	apiKey, err := readSecret("DEAL_SENSE_API_KEY")
	if err != nil {
		return Config{}, err
	}

	metricsPort, err := parseMetricsPort(os.Getenv("METRICS_PORT"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		BotToken:         token,
		AllowlistUserIDs: ids,
		APIBaseURL:       cmp.Or(strings.TrimSpace(os.Getenv("API_BASE_URL")), "http://localhost:8080"),
		APIKey:           apiKey,
		LogLevel:         logLevel,
		ProfileStorePath: cmp.Or(strings.TrimSpace(os.Getenv("TELEGRAM_PROFILE_STORE_PATH")), "/data/telegram-profiles.json"),
		LLMStorePath:     cmp.Or(strings.TrimSpace(os.Getenv("TELEGRAM_LLM_STORE_PATH")), "/data/telegram-llm-settings.json"),
		MetricsPort:      metricsPort,
	}, nil
}

// botConfigOverlay captures the bot-tunable fields after JSON overlay
// resolution; consumers should check the corresponding `has*` flag to know
// whether the JSON file actually provided a value.
type botConfigOverlay struct {
	token     string
	allowlist []int64
	openMode  bool
	logLevel  string
}

// loadJSONOverlay returns (overlay, true, nil) when BOT_CONFIG_PATH points
// at a readable, valid bot config JSON. Returns (zero, false, nil) when
// the path is unset OR the file is absent (legitimate "no overlay yet"
// states). Corrupt JSON or domain-validation failures surface as errors.
func loadJSONOverlay() (botConfigOverlay, bool, error) {
	path := strings.TrimSpace(os.Getenv("BOT_CONFIG_PATH"))
	if path == "" {
		return botConfigOverlay{}, false, nil
	}
	store, err := botconfigstore.NewFileStore(path)
	if err != nil {
		return botConfigOverlay{}, false, fmt.Errorf("telegram: bot config store: %w", err)
	}
	cfg, found, err := store.Load(context.Background())
	if err != nil {
		return botConfigOverlay{}, false, fmt.Errorf("telegram: load bot config: %w", err)
	}
	if !found {
		return botConfigOverlay{}, false, nil
	}
	o := botConfigOverlay{
		token:    cfg.Token(),
		logLevel: cfg.LogLevel().String(),
	}
	if cfg.Allowlist().IsOpen() {
		o.openMode = true
	} else {
		o.allowlist = cfg.Allowlist().Members()
	}
	return o, true, nil
}

// mergeBotFields applies overlay precedence over env. With an overlay,
// token/allowlist/log_level come from the overlay (including the open-mode
// "empty allowlist" semantics). Without one, the historical env behaviour
// applies (and ErrMissingBotToken is returned if TELEGRAM_BOT_TOKEN is unset).
func mergeBotFields(overlay botConfigOverlay, hasOverlay bool) (token string, allowlist []int64, logLevel string, err error) {
	if hasOverlay {
		token = overlay.token
		if !overlay.openMode {
			allowlist = overlay.allowlist
		}
		logLevel = overlay.logLevel
		return token, allowlist, logLevel, nil
	}
	rawToken, err := readSecret("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return "", nil, "", err
	}
	token = strings.TrimSpace(rawToken)
	if token == "" {
		return "", nil, "", ErrMissingBotToken
	}
	allowlist, err = parseAllowlist(os.Getenv("ALLOWLIST_USER_IDS"))
	if err != nil {
		return "", nil, "", err
	}
	logLevel = strings.ToLower(cmp.Or(strings.TrimSpace(os.Getenv("LOG_LEVEL")), "info"))
	return token, allowlist, logLevel, nil
}

// parseMetricsPort returns 0 (disabled) for empty input, the parsed port
// for a valid integer, and ErrInvalidMetricsPort otherwise.
func parseMetricsPort(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	p, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidMetricsPort, raw)
	}
	return p, nil
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
