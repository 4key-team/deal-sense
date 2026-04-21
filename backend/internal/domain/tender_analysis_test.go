package domain_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestNewDocument(t *testing.T) {
	tests := []struct {
		name    string
		docName string
		ft      domain.FileType
		content string
		wantErr error
	}{
		{name: "valid pdf", docName: "spec.pdf", ft: domain.FileTypePDF, content: "some text"},
		{name: "valid docx", docName: "req.docx", ft: domain.FileTypeDOCX, content: "requirements"},
		{name: "empty content", docName: "spec.pdf", ft: domain.FileTypePDF, content: "", wantErr: domain.ErrEmptyContent},
		{name: "empty name", docName: "", ft: domain.FileTypePDF, content: "text", wantErr: domain.ErrEmptyName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewDocument(tt.docName, tt.ft, tt.content)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewDocument() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewDocument() unexpected error: %v", err)
			}
			if got.Name() != tt.docName {
				t.Errorf("Name() = %q, want %q", got.Name(), tt.docName)
			}
			if got.FileType() != tt.ft {
				t.Errorf("FileType() = %v, want %v", got.FileType(), tt.ft)
			}
			if got.Content() != tt.content {
				t.Errorf("Content() = %q, want %q", got.Content(), tt.content)
			}
		})
	}
}

func TestNewTenderAnalysis(t *testing.T) {
	validDoc, _ := domain.NewDocument("spec.pdf", domain.FileTypePDF, "content")

	tests := []struct {
		name    string
		docs    []domain.Document
		profile string
		wantErr error
	}{
		{name: "valid", docs: []domain.Document{*validDoc}, profile: "Acme Corp"},
		{name: "no documents", docs: nil, profile: "Acme Corp", wantErr: domain.ErrEmptyContent},
		{name: "empty documents", docs: []domain.Document{}, profile: "Acme Corp", wantErr: domain.ErrEmptyContent},
		{name: "empty profile", docs: []domain.Document{*validDoc}, profile: "", wantErr: domain.ErrEmptyCompany},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewTenderAnalysis(tt.docs, tt.profile)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewTenderAnalysis() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewTenderAnalysis() unexpected error: %v", err)
			}
			if len(got.Documents()) != len(tt.docs) {
				t.Errorf("Documents() len = %d, want %d", len(got.Documents()), len(tt.docs))
			}
		})
	}
}

func TestTenderAnalysis_CompanyProfile(t *testing.T) {
	doc, _ := domain.NewDocument("spec.pdf", domain.FileTypePDF, "content")
	ta, _ := domain.NewTenderAnalysis([]domain.Document{*doc}, "Acme Corp")
	if ta.CompanyProfile() != "Acme Corp" {
		t.Errorf("CompanyProfile() = %q, want %q", ta.CompanyProfile(), "Acme Corp")
	}
}

func TestTenderAnalysis_SetResult(t *testing.T) {
	doc, _ := domain.NewDocument("spec.pdf", domain.FileTypePDF, "content")
	ta, err := domain.NewTenderAnalysis([]domain.Document{*doc}, "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	score, _ := domain.NewMatchScore(85)
	ta.SetResult(domain.VerdictGo, domain.RiskLow, score, "Good fit")

	if ta.Verdict() != domain.VerdictGo {
		t.Errorf("Verdict() = %v, want %v", ta.Verdict(), domain.VerdictGo)
	}
	if ta.Risk() != domain.RiskLow {
		t.Errorf("Risk() = %v, want %v", ta.Risk(), domain.RiskLow)
	}
	if ta.Score().Value() != 85 {
		t.Errorf("Score() = %d, want 85", ta.Score().Value())
	}
	if ta.Summary() != "Good fit" {
		t.Errorf("Summary() = %q, want %q", ta.Summary(), "Good fit")
	}
}

func TestTenderAnalysis_SetExtras(t *testing.T) {
	doc, _ := domain.NewDocument("spec.pdf", domain.FileTypePDF, "content")
	ta, err := domain.NewTenderAnalysis([]domain.Document{*doc}, "Acme Corp")
	if err != nil {
		t.Fatal(err)
	}

	// Initially empty
	if len(ta.Pros()) != 0 {
		t.Errorf("Pros() len = %d, want 0", len(ta.Pros()))
	}
	if len(ta.Cons()) != 0 {
		t.Errorf("Cons() len = %d, want 0", len(ta.Cons()))
	}
	if len(ta.Requirements()) != 0 {
		t.Errorf("Requirements() len = %d, want 0", len(ta.Requirements()))
	}
	if ta.Effort() != "" {
		t.Errorf("Effort() = %q, want empty", ta.Effort())
	}

	// SetExtras
	pro, _ := domain.NewProCon("Fast team", "We deliver quickly")
	con, _ := domain.NewProCon("No ISO", "Missing certification")
	req, _ := domain.NewRequirement("Go experience", domain.ReqMet)

	ta.SetExtras(
		[]domain.ProCon{pro},
		[]domain.ProCon{con},
		[]domain.Requirement{req},
		"~40 hours",
	)

	if len(ta.Pros()) != 1 {
		t.Errorf("Pros() len = %d, want 1", len(ta.Pros()))
	}
	if ta.Pros()[0].Title() != "Fast team" {
		t.Errorf("Pros()[0].Title() = %q, want Fast team", ta.Pros()[0].Title())
	}
	if len(ta.Cons()) != 1 {
		t.Errorf("Cons() len = %d, want 1", len(ta.Cons()))
	}
	if ta.Cons()[0].Desc() != "Missing certification" {
		t.Errorf("Cons()[0].Desc() = %q", ta.Cons()[0].Desc())
	}
	if len(ta.Requirements()) != 1 {
		t.Errorf("Requirements() len = %d, want 1", len(ta.Requirements()))
	}
	if ta.Requirements()[0].Label() != "Go experience" {
		t.Errorf("Requirements()[0].Label() = %q", ta.Requirements()[0].Label())
	}
	if ta.Effort() != "~40 hours" {
		t.Errorf("Effort() = %q, want ~40 hours", ta.Effort())
	}
}

func TestTenderAnalysis_SetExtras_Nil(t *testing.T) {
	doc, _ := domain.NewDocument("spec.pdf", domain.FileTypePDF, "content")
	ta, _ := domain.NewTenderAnalysis([]domain.Document{*doc}, "Acme")
	ta.SetExtras(nil, nil, nil, "")
	if ta.Pros() == nil {
		t.Error("SetExtras(nil pros) should set empty slice")
	}
	if ta.Cons() == nil {
		t.Error("SetExtras(nil cons) should set empty slice")
	}
	if ta.Requirements() == nil {
		t.Error("SetExtras(nil reqs) should set empty slice")
	}
}

func TestParseRequirementStatus(t *testing.T) {
	tests := []struct {
		input string
		want  domain.RequirementStatus
		err   error
	}{
		{"met", domain.ReqMet, nil},
		{"partial", domain.ReqPartial, nil},
		{"miss", domain.ReqMiss, nil},
		{"invalid", "", domain.ErrInvalidReqStatus},
		{"", "", domain.ErrInvalidReqStatus},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := domain.ParseRequirementStatus(tt.input)
			if got != tt.want { t.Errorf("got %q, want %q", got, tt.want) }
			if !errors.Is(err, tt.err) { t.Errorf("err = %v, want %v", err, tt.err) }
		})
	}
}

func TestNewProCon(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		desc    string
		wantErr error
	}{
		{name: "valid with desc", title: "Strong team", desc: "10 engineers"},
		{name: "valid empty desc", title: "Fast delivery", desc: ""},
		{name: "valid long title", title: "Experienced in government tenders", desc: "5+ years"},
		{name: "empty title", title: "", desc: "desc", wantErr: domain.ErrEmptyTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc, err := domain.NewProCon(tt.title, tt.desc)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewProCon() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewProCon() unexpected error: %v", err)
			}
			if pc.Title() != tt.title {
				t.Errorf("Title() = %q, want %q", pc.Title(), tt.title)
			}
			if pc.Desc() != tt.desc {
				t.Errorf("Desc() = %q, want %q", pc.Desc(), tt.desc)
			}
		})
	}
}

func TestNewRequirement(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		status  domain.RequirementStatus
		wantErr error
	}{
		{name: "valid met", label: "Go experience", status: domain.ReqMet},
		{name: "valid partial", label: "ISO 27001", status: domain.ReqPartial},
		{name: "valid miss", label: "5 year track record", status: domain.ReqMiss},
		{name: "empty label", label: "", status: domain.ReqMet, wantErr: domain.ErrEmptyLabel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := domain.NewRequirement(tt.label, tt.status)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewRequirement() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewRequirement() unexpected error: %v", err)
			}
			if r.Label() != tt.label {
				t.Errorf("Label() = %q, want %q", r.Label(), tt.label)
			}
			if r.Status() != tt.status {
				t.Errorf("Status() = %v, want %v", r.Status(), tt.status)
			}
		})
	}
}
