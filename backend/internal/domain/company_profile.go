package domain

import (
	"strings"
)

// CompanyProfile is the team/company context fed to the LLM during tender
// analysis. Empty profiles are rejected at construction time so the invariant
// "at least one descriptive field" holds throughout the system.
type CompanyProfile struct {
	name            string
	teamSize        string
	experience      string
	techStack       []string
	certifications  []string
	specializations []string
	keyClients      string
	extra           string
}

// NewCompanyProfile constructs a CompanyProfile. Strings are trimmed and
// slice entries are filtered for emptiness; if nothing descriptive remains
// the constructor returns ErrEmptyCompany.
func NewCompanyProfile(
	name, teamSize, experience string,
	techStack, certifications, specializations []string,
	keyClients, extra string,
) (*CompanyProfile, error) {
	p := &CompanyProfile{
		name:            strings.TrimSpace(name),
		teamSize:        strings.TrimSpace(teamSize),
		experience:      strings.TrimSpace(experience),
		techStack:       trimFilter(techStack),
		certifications:  trimFilter(certifications),
		specializations: trimFilter(specializations),
		keyClients:      strings.TrimSpace(keyClients),
		extra:           strings.TrimSpace(extra),
	}
	if p.isEmpty() {
		return nil, ErrEmptyCompany
	}
	return p, nil
}

// Accessors return the validated, trimmed/filtered fields. They exist for
// serialisation in adapters (e.g. profilestore) — the VO itself remains
// immutable from outside.
func (p *CompanyProfile) Name() string              { return p.name }
func (p *CompanyProfile) TeamSize() string          { return p.teamSize }
func (p *CompanyProfile) Experience() string        { return p.experience }
func (p *CompanyProfile) TechStack() []string       { return p.techStack }
func (p *CompanyProfile) Certifications() []string  { return p.certifications }
func (p *CompanyProfile) Specializations() []string { return p.specializations }
func (p *CompanyProfile) KeyClients() string        { return p.keyClients }
func (p *CompanyProfile) Extra() string             { return p.extra }

func (p *CompanyProfile) isEmpty() bool {
	return p.name == "" &&
		p.teamSize == "" &&
		p.experience == "" &&
		len(p.techStack) == 0 &&
		len(p.certifications) == 0 &&
		len(p.specializations) == 0 &&
		p.keyClients == "" &&
		p.extra == ""
}

// Render serialises the profile in the same shape the web flow builds
// (frontend/src/screens/Tender/TenderReport.tsx) so the LLM userPrompt is
// consistent between web and Telegram entry points.
func (p *CompanyProfile) Render() string {
	parts := make([]string, 0, 8)
	if p.name != "" {
		parts = append(parts, "Company: "+p.name)
	}
	if p.teamSize != "" {
		parts = append(parts, "Team: "+p.teamSize+" people")
	}
	if p.experience != "" {
		parts = append(parts, "Experience: "+p.experience+" years")
	}
	if len(p.techStack) > 0 {
		parts = append(parts, "Tech stack: "+strings.Join(p.techStack, ", "))
	}
	if len(p.certifications) > 0 {
		parts = append(parts, "Certifications: "+strings.Join(p.certifications, ", "))
	}
	if len(p.specializations) > 0 {
		parts = append(parts, "Specializations: "+strings.Join(p.specializations, ", "))
	}
	if p.keyClients != "" {
		parts = append(parts, "Key clients/projects: "+p.keyClients)
	}
	if p.extra != "" {
		parts = append(parts, "Additional: "+p.extra)
	}
	return strings.Join(parts, ". ")
}

func trimFilter(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
