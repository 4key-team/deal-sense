package llmstub_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/adapter/llmstub"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestProvider_ImplementsLLMProviderInterface(t *testing.T) {
	var _ usecase.LLMProvider = llmstub.New("stub", "ok")
}

func TestProvider_ReturnsResponsesInOrder(t *testing.T) {
	p := llmstub.New("stub", "first", "second", "third")
	ctx := context.Background()

	got1, _, err := p.GenerateCompletion(ctx, "sys", "u1")
	if err != nil || got1 != "first" {
		t.Errorf("call 1: %q, %v", got1, err)
	}
	got2, _, err := p.GenerateCompletion(ctx, "sys", "u2")
	if err != nil || got2 != "second" {
		t.Errorf("call 2: %q, %v", got2, err)
	}
	got3, _, err := p.GenerateCompletion(ctx, "sys", "u3")
	if err != nil || got3 != "third" {
		t.Errorf("call 3: %q, %v", got3, err)
	}
}

func TestProvider_ExhaustedScriptReturnsError(t *testing.T) {
	p := llmstub.New("stub", "only")
	ctx := context.Background()
	_, _, _ = p.GenerateCompletion(ctx, "sys", "u")
	_, _, err := p.GenerateCompletion(ctx, "sys", "u")
	if !errors.Is(err, llmstub.ErrScriptExhausted) {
		t.Errorf("err = %v, want ErrScriptExhausted", err)
	}
}

func TestProvider_CapturesPrompts(t *testing.T) {
	p := llmstub.New("stub", "r1", "r2")
	ctx := context.Background()
	_, _, _ = p.GenerateCompletion(ctx, "sys-1", "user-1")
	_, _, _ = p.GenerateCompletion(ctx, "sys-2", "user-2")

	sys := p.SystemPrompts()
	usr := p.UserPrompts()
	if len(sys) != 2 || sys[0] != "sys-1" || sys[1] != "sys-2" {
		t.Errorf("SystemPrompts = %v", sys)
	}
	if len(usr) != 2 || usr[0] != "user-1" || usr[1] != "user-2" {
		t.Errorf("UserPrompts = %v", usr)
	}
	if p.CallCount() != 2 {
		t.Errorf("CallCount = %d, want 2", p.CallCount())
	}
}

func TestProvider_DefensiveCopies(t *testing.T) {
	p := llmstub.New("stub", "r")
	_, _, _ = p.GenerateCompletion(context.Background(), "sys", "user")
	sys := p.SystemPrompts()
	sys[0] = "MUTATED"
	if p.SystemPrompts()[0] == "MUTATED" {
		t.Error("SystemPrompts should return a defensive copy")
	}
}

func TestProvider_Name(t *testing.T) {
	p := llmstub.New("myname", "r")
	if p.Name() != "myname" {
		t.Errorf("Name = %q", p.Name())
	}
}

func TestProvider_CheckConnectionAndListModels(t *testing.T) {
	p := llmstub.New("stub", "r")
	if err := p.CheckConnection(context.Background()); err != nil {
		t.Errorf("CheckConnection: %v", err)
	}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Errorf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Error("expected at least one model")
	}
}

func TestProvider_ConcurrentSafety(t *testing.T) {
	// Coupling tests may fan out parallel goroutines; ensure capture is
	// race-safe.
	p := llmstub.New("stub", makeResponses(100)...)
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = p.GenerateCompletion(context.Background(), "sys", "u")
		}()
	}
	wg.Wait()
	if p.CallCount() != 100 {
		t.Errorf("CallCount = %d, want 100", p.CallCount())
	}
	if len(p.SystemPrompts()) != 100 {
		t.Errorf("SystemPrompts count = %d, want 100", len(p.SystemPrompts()))
	}
}

func makeResponses(n int) []string {
	out := make([]string, n)
	for i := range n {
		out[i] = strings.Repeat("ok", 1) // keep short
		_ = i
	}
	return out
}
