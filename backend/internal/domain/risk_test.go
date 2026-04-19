package domain_test

import (
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestParseRisk(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    domain.Risk
		wantErr bool
	}{
		{name: "low", input: "low", want: domain.RiskLow},
		{name: "medium", input: "medium", want: domain.RiskMedium},
		{name: "high", input: "high", want: domain.RiskHigh},
		{name: "invalid empty", input: "", wantErr: true},
		{name: "invalid critical", input: "critical", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseRisk(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRisk(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRisk(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseRisk(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
