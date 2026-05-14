package domain

import (
	"regexp"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

// LogLevel is the ubiquitous-language enum for bot/server log verbosity.
// Stored as a string so JSON serialisation is human-readable.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// String satisfies fmt.Stringer for logging/serialisation contexts.
func (l LogLevel) String() string { return string(l) }

// ParseLogLevel normalises (trim + lowercase) and validates a user-supplied
// level. Empty or unknown values return ErrInvalidLogLevel — no silent
// fallback to "info" at the domain layer; callers wanting a default must
// apply it explicitly.
func ParseLogLevel(s string) (LogLevel, error) {
	switch LogLevel(strings.ToLower(strings.TrimSpace(s))) {
	case LogLevelDebug:
		return LogLevelDebug, nil
	case LogLevelInfo:
		return LogLevelInfo, nil
	case LogLevelWarn:
		return LogLevelWarn, nil
	case LogLevelError:
		return LogLevelError, nil
	default:
		return "", ErrInvalidLogLevel
	}
}

// botTokenPattern accepts Telegram bot tokens: numeric ID, colon, then a
// secret of base64url-style characters at least 20 chars long. Real tokens
// are ~35 chars; we keep the floor permissive but still reject obvious
// non-tokens.
var botTokenPattern = regexp.MustCompile(`^\d{1,32}:[A-Za-z0-9_-]{20,}$`)

// BotConfig is the validated runtime configuration of the Telegram bot.
// Each field is enforced at construction time so handlers/adapters can rely
// on the invariants without re-validating.
type BotConfig struct {
	token     string
	allowlist *auth.Allowlist
	logLevel  LogLevel
}

// NewBotConfig validates a raw token, allowlist IDs and log level and
// returns a BotConfig. An empty allowlist is intentionally accepted and
// promoted to "open mode" via auth.ParseAllowlist; callers that want strict
// production policy must pass a non-empty ID list.
func NewBotConfig(token string, allowlistUserIDs []int64, logLevel string) (*BotConfig, error) {
	tok := strings.TrimSpace(token)
	if !botTokenPattern.MatchString(tok) {
		return nil, ErrInvalidBotToken
	}
	level, err := ParseLogLevel(logLevel)
	if err != nil {
		return nil, err
	}
	allowlist, err := auth.ParseAllowlist(allowlistUserIDs)
	if err != nil {
		return nil, err
	}
	return &BotConfig{
		token:     tok,
		allowlist: allowlist,
		logLevel:  level,
	}, nil
}

// Token returns the raw bot token. Adapters that handle UI/transport should
// prefer MaskedToken to avoid leaking the secret into logs or responses.
func (c *BotConfig) Token() string { return c.token }

// Allowlist returns the composed allowlist VO. Never nil for a constructed
// BotConfig.
func (c *BotConfig) Allowlist() *auth.Allowlist { return c.allowlist }

// LogLevel returns the validated level.
func (c *BotConfig) LogLevel() LogLevel { return c.logLevel }

// MaskedToken returns a redacted form of the token suitable for UI/logs:
// the bot ID prefix is kept intact (for operator recognition) and the last
// 4 characters of the secret remain (for fingerprinting). Everything else
// is replaced with asterisks. Tokens whose secret is too short to mask
// safely are fully redacted.
func (c *BotConfig) MaskedToken() string {
	idx := strings.IndexByte(c.token, ':')
	if idx < 0 {
		return strings.Repeat("*", len(c.token))
	}
	secret := c.token[idx+1:]
	if len(secret) <= 4 {
		return c.token[:idx+1] + strings.Repeat("*", len(secret))
	}
	keep := secret[len(secret)-4:]
	return c.token[:idx+1] + strings.Repeat("*", len(secret)-4) + keep
}
