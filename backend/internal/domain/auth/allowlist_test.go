package auth_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain/auth"
)

// --- Open mode -------------------------------------------------------------

func TestNewOpenAllowlist_IsOpen(t *testing.T) {
	a := auth.NewOpenAllowlist()
	if !a.IsOpen() {
		t.Error("NewOpenAllowlist().IsOpen() = false, want true")
	}
}

func TestNewOpenAllowlist_AllowsAnyone(t *testing.T) {
	a := auth.NewOpenAllowlist()
	tests := []struct {
		name string
		id   int64
	}{
		{"positive small", 1},
		{"positive large", 9999999999},
		{"zero", 0},
		{"negative", -42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !a.IsAllowed(tt.id) {
				t.Errorf("open allowlist must allow id=%d", tt.id)
			}
		})
	}
}

func TestNewOpenAllowlist_HasNoMembers(t *testing.T) {
	a := auth.NewOpenAllowlist()
	if got := a.Members(); len(got) != 0 {
		t.Errorf("Members() = %v, want empty for open allowlist", got)
	}
}

// --- Restricted mode -------------------------------------------------------

func TestNewRestrictedAllowlist_RejectsEmpty(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
	}{
		{"nil slice", nil},
		{"empty slice", []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.NewRestrictedAllowlist(tt.ids)
			if !errors.Is(err, auth.ErrEmptyAllowlist) {
				t.Errorf("err = %v, want %v", err, auth.ErrEmptyAllowlist)
			}
		})
	}
}

func TestNewRestrictedAllowlist_RejectsInvalidIDs(t *testing.T) {
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
			_, err := auth.NewRestrictedAllowlist(tt.ids)
			if !errors.Is(err, auth.ErrInvalidUserID) {
				t.Errorf("err = %v, want %v", err, auth.ErrInvalidUserID)
			}
		})
	}
}

func TestNewRestrictedAllowlist_IsAllowed(t *testing.T) {
	a, err := auth.NewRestrictedAllowlist([]int64{1, 2, 3})
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

func TestNewRestrictedAllowlist_IsOpenFalse(t *testing.T) {
	a, err := auth.NewRestrictedAllowlist([]int64{1, 2})
	if err != nil {
		t.Fatalf("constructor failed: %v", err)
	}
	if a.IsOpen() {
		t.Error("restricted allowlist must report IsOpen() == false")
	}
}

func TestRestrictedAllowlist_Immutable(t *testing.T) {
	ids := []int64{1, 2, 3}
	a, err := auth.NewRestrictedAllowlist(ids)
	if err != nil {
		t.Fatalf("constructor failed: %v", err)
	}
	ids[0] = 999

	if !a.IsAllowed(1) {
		t.Error("IsAllowed(1) should remain true after caller mutated source slice")
	}
	if a.IsAllowed(999) {
		t.Error("IsAllowed(999) should remain false — allowlist took a defensive copy")
	}
}

func TestRestrictedAllowlist_DuplicatesIgnored(t *testing.T) {
	a, err := auth.NewRestrictedAllowlist([]int64{1, 1, 2})
	if err != nil {
		t.Fatalf("duplicates should not error: %v", err)
	}
	if !a.IsAllowed(1) || !a.IsAllowed(2) {
		t.Error("IsAllowed should still cover both distinct IDs")
	}
}

func TestRestrictedAllowlist_Members_ReturnsAllIDs(t *testing.T) {
	a, err := auth.NewRestrictedAllowlist([]int64{3, 1, 2, 1})
	if err != nil {
		t.Fatalf("constructor failed: %v", err)
	}
	got := a.Members()
	if len(got) != 3 {
		t.Fatalf("Members() len = %d, want 3 (dedup'd): %v", len(got), got)
	}
	seen := map[int64]bool{1: false, 2: false, 3: false}
	for _, id := range got {
		seen[id] = true
	}
	for id, found := range seen {
		if !found {
			t.Errorf("Members() missing %d: %v", id, got)
		}
	}
}

// --- Parse smart factory ---------------------------------------------------

func TestParseAllowlist_EmptyReturnsOpen(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
	}{
		{"nil slice", nil},
		{"empty slice", []int64{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := auth.ParseAllowlist(tt.ids)
			if err != nil {
				t.Fatalf("ParseAllowlist(%v) unexpected error: %v", tt.ids, err)
			}
			if !a.IsOpen() {
				t.Errorf("ParseAllowlist(%v) should produce open allowlist", tt.ids)
			}
		})
	}
}

func TestParseAllowlist_NonEmptyReturnsRestricted(t *testing.T) {
	a, err := auth.ParseAllowlist([]int64{42, 100})
	if err != nil {
		t.Fatalf("ParseAllowlist unexpected error: %v", err)
	}
	if a.IsOpen() {
		t.Error("ParseAllowlist with IDs should produce restricted allowlist")
	}
	if !a.IsAllowed(42) {
		t.Error("ParseAllowlist should preserve member 42")
	}
	if a.IsAllowed(999) {
		t.Error("ParseAllowlist should not include 999")
	}
}

func TestParseAllowlist_PropagatesInvalidIDError(t *testing.T) {
	_, err := auth.ParseAllowlist([]int64{1, -5, 2})
	if !errors.Is(err, auth.ErrInvalidUserID) {
		t.Errorf("err = %v, want wrapping ErrInvalidUserID", err)
	}
}
