package domain_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestParseVerdict(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.Verdict
		wantErr bool
	}{
		{name: "go", input: "go", want: domain.VerdictGo},
		{name: "no-go", input: "no-go", want: domain.VerdictNoGo},
		{name: "invalid empty", input: "", wantErr: true},
		{name: "invalid random", input: "maybe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseVerdict(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseVerdict(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseVerdict(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseVerdict(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
