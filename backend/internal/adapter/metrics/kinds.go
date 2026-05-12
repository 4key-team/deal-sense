package metrics

// DeclineKind enumerates the canonical values of the `kind` label on
// dealsense_security_decline_total. Defined here (in the metrics adapter)
// rather than in domain/ because the labels are an observability shape,
// not a domain invariant — Layer 4 risk classification stays in domain.
type DeclineKind string

const (
	DeclineAllowlist     DeclineKind = "allowlist"
	DeclineAPIKey        DeclineKind = "api_key"
	DeclineRateLimit     DeclineKind = "rate_limit"
	DeclineLLMParseError DeclineKind = "llm_parse_error"
)

// String returns the canonical token used as the Prometheus label value.
func (k DeclineKind) String() string { return string(k) }
