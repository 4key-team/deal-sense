package usecase_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestCheckLLMConnection_Execute(t *testing.T) {
	tests := []struct {
		name    string
		llmErr  error
		llmName string
		wantErr bool
	}{
		{name: "healthy connection", llmName: "openai", llmErr: nil},
		{name: "failed connection", llmName: "openai", llmErr: errors.New("timeout"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			llm := &stubLLM{err: tt.llmErr, name: tt.llmName}
			uc := usecase.NewCheckLLMConnection(llm)

			result, err := uc.Execute(t.Context())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Provider != tt.llmName {
				t.Errorf("Provider = %q, want %q", result.Provider, tt.llmName)
			}
			if !result.OK {
				t.Error("expected OK = true")
			}
		})
	}
}
