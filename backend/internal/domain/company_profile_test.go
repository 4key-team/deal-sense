package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestNewCompanyProfile_AllEmpty_ReturnsErrEmptyCompany(t *testing.T) {
	_, err := domain.NewCompanyProfile("", "", "", nil, nil, nil, "", "")
	if !errors.Is(err, domain.ErrEmptyCompany) {
		t.Fatalf("err = %v, want wrapping ErrEmptyCompany", err)
	}
}

func TestNewCompanyProfile_AllEmptyStrings_ReturnsErrEmptyCompany(t *testing.T) {
	// Whitespace-only strings and empty slices are treated as empty.
	_, err := domain.NewCompanyProfile("   ", "\t", "", []string{}, []string{"  "}, nil, " ", "")
	if !errors.Is(err, domain.ErrEmptyCompany) {
		t.Fatalf("err = %v, want wrapping ErrEmptyCompany (whitespace-only is empty)", err)
	}
}

func TestNewCompanyProfile_OnlyName_OK(t *testing.T) {
	p, err := domain.NewCompanyProfile("Acme", "", "", nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("nil profile returned with nil error")
	}
}

func TestCompanyProfile_Render_FullProfile(t *testing.T) {
	p, err := domain.NewCompanyProfile(
		"Acme",
		"15",
		"7",
		[]string{"Go", "React"},
		[]string{"ISO 9001"},
		[]string{"backend", "mobile"},
		"Sberbank, Yandex",
		"Remote-first",
	)
	if err != nil {
		t.Fatalf("NewCompanyProfile: %v", err)
	}
	got := p.Render()
	wants := []string{
		"Company: Acme",
		"Team: 15 people",
		"Experience: 7 years",
		"Tech stack: Go, React",
		"Certifications: ISO 9001",
		"Specializations: backend, mobile",
		"Key clients/projects: Sberbank, Yandex",
		"Additional: Remote-first",
	}
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("Render missing %q\nout: %s", want, got)
		}
	}
}

func TestCompanyProfile_Render_OnlyNameOmitsOthers(t *testing.T) {
	p, err := domain.NewCompanyProfile("Acme", "", "", nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("NewCompanyProfile: %v", err)
	}
	got := p.Render()
	if !strings.Contains(got, "Company: Acme") {
		t.Errorf("Render missing 'Company: Acme' in %q", got)
	}
	for _, forbidden := range []string{"Team:", "Experience:", "Tech stack:", "Certifications:", "Specializations:", "Key clients", "Additional:"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("Render should omit empty section %q, got %q", forbidden, got)
		}
	}
}

func TestCompanyProfile_Render_FiltersEmptySliceItems(t *testing.T) {
	// Wizard input "Go, , React" → empty items filtered, no trailing comma.
	p, err := domain.NewCompanyProfile("Acme", "", "", []string{"Go", "", "  ", "React"}, nil, nil, "", "")
	if err != nil {
		t.Fatalf("NewCompanyProfile: %v", err)
	}
	got := p.Render()
	if !strings.Contains(got, "Tech stack: Go, React") {
		t.Errorf("Render should join non-empty stack items, got %q", got)
	}
}

func TestCompanyProfile_Render_TrimsFieldWhitespace(t *testing.T) {
	p, err := domain.NewCompanyProfile("  Acme  ", "  15 ", "", nil, nil, nil, "", "")
	if err != nil {
		t.Fatalf("NewCompanyProfile: %v", err)
	}
	got := p.Render()
	if !strings.Contains(got, "Company: Acme") {
		t.Errorf("Render should trim name whitespace, got %q", got)
	}
	if !strings.Contains(got, "Team: 15 people") {
		t.Errorf("Render should trim teamSize whitespace, got %q", got)
	}
}
