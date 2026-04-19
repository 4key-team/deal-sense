package domain_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestNewMatchScore(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		want    int
		wantErr error
	}{
		{name: "zero", input: 0, want: 0},
		{name: "fifty", input: 50, want: 50},
		{name: "hundred", input: 100, want: 100},
		{name: "negative", input: -1, wantErr: domain.ErrInvalidScore},
		{name: "over hundred", input: 101, wantErr: domain.ErrInvalidScore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewMatchScore(tt.input)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewMatchScore(%d) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMatchScore(%d) unexpected error: %v", tt.input, err)
			}
			if got.Value() != tt.want {
				t.Errorf("NewMatchScore(%d).Value() = %d, want %d", tt.input, got.Value(), tt.want)
			}
		})
	}
}
