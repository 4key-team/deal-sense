package auth_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

func TestNewAllowlist_RejectsEmpty(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
	}{
		{"nil slice", nil},
		{"empty slice", []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.NewAllowlist(tt.ids)
			if !errors.Is(err, auth.ErrEmptyAllowlist) {
				t.Errorf("err = %v, want %v", err, auth.ErrEmptyAllowlist)
			}
		})
	}
}

func TestNewAllowlist_RejectsInvalidIDs(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
	}{
		{"contains zero", []int64{1, 0, 2}},
		{"contains negative", []int64{1, -5, 2}},
		{"only zero", []int64{0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.NewAllowlist(tt.ids)
			if !errors.Is(err, auth.ErrInvalidUserID) {
				t.Errorf("err = %v, want %v", err, auth.ErrInvalidUserID)
			}
		})
	}
}

func TestAllowlist_IsAllowed(t *testing.T) {
	a, err := auth.NewAllowlist([]int64{1, 2, 3})
	if err != nil {
		t.Fatalf("constructor failed: %v", err)
	}
	tests := []struct {
		name string
		id   int64
		want bool
	}{
		{"first member", 1, true},
		{"middle member", 2, true},
		{"last member", 3, true},
		{"non-member", 99, false},
		{"zero", 0, false},
		{"negative", -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := a.IsAllowed(tt.id); got != tt.want {
				t.Errorf("IsAllowed(%d) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestAllowlist_Immutable(t *testing.T) {
	ids := []int64{1, 2, 3}
	a, err := auth.NewAllowlist(ids)
	if err != nil {
		t.Fatalf("constructor failed: %v", err)
	}
	// Mutate the caller's slice. The Allowlist must already have a copy
	// so its decisions don't shift under us.
	ids[0] = 999

	if !a.IsAllowed(1) {
		t.Error("IsAllowed(1) should remain true after caller mutated source slice")
	}
	if a.IsAllowed(999) {
		t.Error("IsAllowed(999) should remain false — allowlist took a defensive copy")
	}
}

func TestAllowlist_DuplicatesIgnored(t *testing.T) {
	a, err := auth.NewAllowlist([]int64{1, 1, 2})
	if err != nil {
		t.Fatalf("duplicates should not error: %v", err)
	}
	if !a.IsAllowed(1) || !a.IsAllowed(2) {
		t.Error("IsAllowed should still cover both distinct IDs")
	}
}
