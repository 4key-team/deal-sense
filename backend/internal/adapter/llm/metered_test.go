package llm_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llm"
	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

type stubInner struct {
	name string
	out  string
	err  error
}

func (s *stubInner) Name() string { return s.name }
func (s *stubInner) GenerateCompletion(_ context.Context, _, _ string) (string, domain.TokenUsage, error) {
	return s.out, domain.ZeroTokenUsage(), s.err
}
func (s *stubInner) CheckConnection(_ context.Context) error        { return s.err }
func (s *stubInner) ListModels(_ context.Context) ([]string, error) { return nil, s.err }

type recordingObserver struct {
	mu       sync.Mutex
	calls    []callRec
	declines []string
}

type callRec struct {
	provider string
	status   string
}

func (r *recordingObserver) IncLLMCall(provider, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, callRec{provider, status})
}

func (r *recordingObserver) Inc(kind string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.declines = append(r.declines, kind)
}

func TestMetered_IncrementsOnOK(t *testing.T) {
	inner := &stubInner{name: "anthropic", out: "ok"}
	obs := &recordingObserver{}
	wrapped := llm.NewMetered(inner, obs)

	if _, _, err := wrapped.GenerateCompletion(context.Background(), "sys", "user"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(obs.calls) != 1 || obs.calls[0] != (callRec{"anthropic", "ok"}) {
		t.Errorf("calls = %v, want [{anthropic ok}]", obs.calls)
	}
	if len(obs.declines) != 0 {
		t.Errorf("declines = %v, want []", obs.declines)
	}
}

func TestMetered_IncrementsOnGenericError(t *testing.T) {
	inner := &stubInner{name: "openai", err: errors.New("upstream 500")}
	obs := &recordingObserver{}
	wrapped := llm.NewMetered(inner, obs)

	if _, _, err := wrapped.GenerateCompletion(context.Background(), "s", "u"); err == nil {
		t.Fatal("want err")
	}

	if len(obs.calls) != 1 || obs.calls[0] != (callRec{"openai", "error"}) {
		t.Errorf("calls = %v, want [{openai error}]", obs.calls)
	}
	if len(obs.declines) != 0 {
		t.Errorf("declines = %v, want [] (generic error is not a parse error)", obs.declines)
	}
}

func TestMetered_IncrementsParseError(t *testing.T) {
	parseErr := fmt.Errorf("parse response: %w", errors.Join(llm.ErrParseResponse, errors.New("unexpected token")))
	inner := &stubInner{name: "gemini", err: parseErr}
	obs := &recordingObserver{}
	wrapped := llm.NewMetered(inner, obs)

	if _, _, err := wrapped.GenerateCompletion(context.Background(), "s", "u"); err == nil {
		t.Fatal("want err")
	}

	if len(obs.calls) != 1 || obs.calls[0] != (callRec{"gemini", "error"}) {
		t.Errorf("calls = %v, want [{gemini error}]", obs.calls)
	}
	if len(obs.declines) != 1 || obs.declines[0] != "llm_parse_error" {
		t.Errorf("declines = %v, want [llm_parse_error]", obs.declines)
	}
}

func TestMetered_NilObserver_IsPassthrough(t *testing.T) {
	inner := &stubInner{name: "anthropic", out: "ok"}
	wrapped := llm.NewMetered(inner, nil)
	// Same pointer ⇒ no decorator applied; cheaper and matches "metrics off" expectation.
	if wrapped != usecase.LLMProvider(inner) {
		t.Errorf("nil observer should return inner verbatim, got wrapped")
	}
}

func TestMetered_DelegatesName(t *testing.T) {
	inner := &stubInner{name: "custom"}
	obs := &recordingObserver{}
	wrapped := llm.NewMetered(inner, obs)
	if got := wrapped.Name(); got != "custom" {
		t.Errorf("Name() = %q, want custom", got)
	}
}
