package domain_test

import (
	"errors"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

func TestNewProposal(t *testing.T) {
	validContent := []byte("template content")
	validParams := map[string]string{"company": "Acme"}

	tests := []struct {
		name     string
		tmplName string
		content  []byte
		params   map[string]string
		wantErr  error
	}{
		{name: "valid", tmplName: "offer.docx", content: validContent, params: validParams},
		{name: "nil params is ok", tmplName: "offer.docx", content: validContent, params: nil},
		{name: "empty content", tmplName: "offer.docx", content: []byte{}, params: validParams, wantErr: domain.ErrEmptyTemplate},
		{name: "nil content", tmplName: "offer.docx", content: nil, params: validParams, wantErr: domain.ErrEmptyTemplate},
		{name: "empty name", tmplName: "", content: validContent, params: validParams, wantErr: domain.ErrEmptyTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NewProposal(tt.tmplName, tt.content, tt.params)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewProposal() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewProposal() unexpected error: %v", err)
			}
			if got.TemplateName() != tt.tmplName {
				t.Errorf("TemplateName() = %q, want %q", got.TemplateName(), tt.tmplName)
			}
		})
	}
}

func TestProposal_Getters(t *testing.T) {
	params := map[string]string{"company": "Acme", "project": "Portal"}
	content := []byte("template content")
	p, err := domain.NewProposal("offer.docx", content, params)
	if err != nil {
		t.Fatal(err)
	}

	if string(p.TemplateContent()) != "template content" {
		t.Errorf("TemplateContent() = %q", p.TemplateContent())
	}
	if len(p.Parameters()) != 2 {
		t.Errorf("Parameters() len = %d, want 2", len(p.Parameters()))
	}
	if p.Parameters()["company"] != "Acme" {
		t.Errorf("Parameters()[company] = %q, want Acme", p.Parameters()["company"])
	}
}

func TestProposal_SetResult(t *testing.T) {
	p, err := domain.NewProposal("offer.docx", []byte("tmpl"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Result() != nil {
		t.Error("Result() should be nil before SetResult")
	}
	p.SetResult([]byte("generated"))
	if string(p.Result()) != "generated" {
		t.Errorf("Result() = %q, want %q", p.Result(), "generated")
	}
}

func TestParseSectionStatus(t *testing.T) {
	tests := []struct {
		input string
		want  domain.SectionStatus
		err   error
	}{
		{"ai", domain.SectionAI, nil},
		{"filled", domain.SectionFilled, nil},
		{"review", domain.SectionReview, nil},
		{"bad", "", domain.ErrInvalidSectionStatus},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := domain.ParseSectionStatus(tt.input)
			if got != tt.want { t.Errorf("got %q, want %q", got, tt.want) }
			if !errors.Is(err, tt.err) { t.Errorf("err = %v, want %v", err, tt.err) }
		})
	}
}

func TestNewLogEntry(t *testing.T) {
	tests := []struct {
		name    string
		time    string
		msg     string
		wantErr error
	}{
		{name: "valid", time: "14:00", msg: "parsed template"},
		{name: "empty time is ok", time: "", msg: "step completed"},
		{name: "empty msg", time: "14:00", msg: "", wantErr: domain.ErrEmptyMsg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := domain.NewLogEntry(tt.time, tt.msg)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewLogEntry() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewLogEntry() unexpected error: %v", err)
			}
			if e.Time() != tt.time {
				t.Errorf("Time() = %q, want %q", e.Time(), tt.time)
			}
			if e.Msg() != tt.msg {
				t.Errorf("Msg() = %q, want %q", e.Msg(), tt.msg)
			}
		})
	}
}

func TestNewProposalSection(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		status  domain.SectionStatus
		tokens  int
		wantErr error
	}{
		{name: "valid ai", title: "Summary", status: domain.SectionAI, tokens: 420},
		{name: "valid filled", title: "Intro", status: domain.SectionFilled, tokens: 0},
		{name: "valid review", title: "Pricing", status: domain.SectionReview, tokens: 200},
		{name: "empty title", title: "", status: domain.SectionAI, tokens: 100, wantErr: domain.ErrEmptyTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := domain.NewProposalSection(tt.title, tt.status, tt.tokens)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewProposalSection() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewProposalSection() unexpected error: %v", err)
			}
			if s.Title() != tt.title {
				t.Errorf("Title() = %q, want %q", s.Title(), tt.title)
			}
			if s.Status() != tt.status {
				t.Errorf("Status() = %v, want %v", s.Status(), tt.status)
			}
			if s.Tokens() != tt.tokens {
				t.Errorf("Tokens() = %d, want %d", s.Tokens(), tt.tokens)
			}
		})
	}
}
