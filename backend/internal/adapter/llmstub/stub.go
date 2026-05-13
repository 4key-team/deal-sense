// Package llmstub provides a scripted in-memory LLM provider used by
// Layer 4 coupling tests. It captures every prompt sent through the
// usecase.LLMProvider interface so tests can assert on system-prompt
// composition (e.g. "the юр firewall directives are present in every
// call") without depending on a real LLM provider.
package llmstub

import (
	"context"
	"errors"
	"sync"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// ErrScriptExhausted indicates GenerateCompletion was called more times
// than the script provided responses for.
var ErrScriptExhausted = errors.New("llmstub: scripted response sequence exhausted")

// Provider is a scripted, thread-safe usecase.LLMProvider used by tests.
// Construct via New. The zero value is invalid (Name() will return "").
type Provider struct {
	mu sync.Mutex

	name      string
	responses []string
	idx       int

	systemPrompts []string
	userPrompts   []string
}

// New returns a Provider that replies with the given responses in order.
// When the sequence is exhausted GenerateCompletion returns
// ErrScriptExhausted. Pass a single response to use it for every call,
// or duplicate it manually for longer scripts.
func New(name string, responses ...string) *Provider {
	cp := make([]string, len(responses))
	copy(cp, responses)
	return &Provider{
		name:      name,
		responses: cp,
	}
}

// GenerateCompletion records the prompts and returns the next scripted
// response, or ErrScriptExhausted if no responses remain.
func (p *Provider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, domain.TokenUsage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.systemPrompts = append(p.systemPrompts, systemPrompt)
	p.userPrompts = append(p.userPrompts, userPrompt)

	if p.idx >= len(p.responses) {
		return "", domain.TokenUsage{}, ErrScriptExhausted
	}
	resp := p.responses[p.idx]
	p.idx++
	return resp, domain.TokenUsage{}, nil
}

// CheckConnection always succeeds.
func (p *Provider) CheckConnection(ctx context.Context) error {
	return nil
}

// ListModels returns a fixed token so tests don't have to fake provider
// metadata.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"stub-model"}, nil
}

// Name returns the configured provider name.
func (p *Provider) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// SystemPrompts returns a defensive copy of every system prompt observed.
func (p *Provider) SystemPrompts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.systemPrompts))
	copy(out, p.systemPrompts)
	return out
}

// UserPrompts returns a defensive copy of every user prompt observed.
func (p *Provider) UserPrompts() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.userPrompts))
	copy(out, p.userPrompts)
	return out
}

// CallCount returns the number of GenerateCompletion calls made.
func (p *Provider) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.systemPrompts)
}
