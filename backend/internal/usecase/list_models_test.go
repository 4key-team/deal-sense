package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

type stubLLMWithModels struct {
	stubLLM
	models []string
	modErr error
}

func (s *stubLLMWithModels) ListModels(_ context.Context) ([]string, error) {
	return s.models, s.modErr
}

func TestListModels_Execute(t *testing.T) {
	t.Run("returns models", func(t *testing.T) {
		llm := &stubLLMWithModels{models: []string{"gpt-4o", "gpt-3.5"}}
		uc := usecase.NewListModels(llm)
		models, err := uc.Execute(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 2 {
			t.Errorf("got %d models, want 2", len(models))
		}
	})

	t.Run("returns empty list", func(t *testing.T) {
		llm := &stubLLMWithModels{}
		uc := usecase.NewListModels(llm)
		models, err := uc.Execute(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if models != nil {
			t.Errorf("expected nil, got %v", models)
		}
	})

	t.Run("returns error", func(t *testing.T) {
		llm := &stubLLMWithModels{modErr: errors.New("network error")}
		uc := usecase.NewListModels(llm)
		_, err := uc.Execute(t.Context())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
