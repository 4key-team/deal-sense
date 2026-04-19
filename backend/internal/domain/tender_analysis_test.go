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
		{name: "empty name", docName: "", ft: domain.FileTypePDF, content: "text", wantErr: domain.ErrEmptyContent},
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
