package llm

import (
	"context"
	"errors"
	"fmt"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// ErrParseResponse marks a JSON-decode failure of a provider response. The
// Metered decorator uses errors.Is to count parse errors under the
// dealsense_security_decline_total{kind="llm_parse_error"} series — those
// are operational signals (provider drift, malformed responses) rather
// than just generic LLM failures.
var ErrParseResponse = errors.New("llm: parse response")

// LLMObserver is the narrow port the Metered decorator uses to record call
// outcomes. Implementations live in adapter/metrics.Collector.
type LLMObserver interface {
	IncLLMCall(provider, status string)
	Inc(kind string)
}

// Metered wraps a usecase.LLMProvider, counting every GenerateCompletion
// call via the observer. Parse errors (wrapping ErrParseResponse) emit an
// additional decline.
type Metered struct {
	inner    usecase.LLMProvider
	observer LLMObserver
}

// NewMetered wraps inner with metric counting. A nil observer makes the
// decorator a pure pass-through — callers do not need to branch on
// "metrics enabled" themselves.
func NewMetered(inner usecase.LLMProvider, observer LLMObserver) usecase.LLMProvider {
	if observer == nil {
		return inner
	}
	return &Metered{inner: inner, observer: observer}
}

// Name returns the wrapped provider's name verbatim.
func (m *Metered) Name() string { return m.inner.Name() }

// GenerateCompletion forwards to the inner provider and records the call.
// Parse errors (wrapping ErrParseResponse) emit an extra decline so
// provider drift is visible in /metrics independent of generic upstream
// failures.
func (m *Metered) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, domain.TokenUsage, error) {
	out, usage, err := m.inner.GenerateCompletion(ctx, systemPrompt, userPrompt)
	if err != nil {
		m.observer.IncLLMCall(m.inner.Name(), "error")
		if errors.Is(err, ErrParseResponse) {
			m.observer.Inc("llm_parse_error")
		}
		return out, usage, err
	}
	m.observer.IncLLMCall(m.inner.Name(), "ok")
	return out, usage, nil
}

// CheckConnection delegates to the inner provider; not counted (CheckConnection
// is a sanity-check, not real work).
func (m *Metered) CheckConnection(ctx context.Context) error {
	return m.inner.CheckConnection(ctx)
}

// ListModels delegates without counting.
func (m *Metered) ListModels(ctx context.Context) ([]string, error) {
	return m.inner.ListModels(ctx)
}

// wrapParseErr is a helper used by individual provider files to mark a
// JSON-decode error so Metered can attribute the decline. Production
// code calls it; tests don't.
func wrapParseErr(prefix string, err error) error {
	return fmt.Errorf("%s: %w", prefix, errors.Join(ErrParseResponse, err))
}
