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
