package usecase_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

func TestParseMarkdownTemplate(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantErr      error
		wantTitle    string
		wantMeta     map[string]string
		wantSections []usecase.MdSection
	}{
		{
			name:    "empty input",
			input:   "",
			wantErr: domain.ErrEmptyTemplate,
		},
		{
			name:      "top-level heading only",
			input:     "# Title document",
			wantTitle: "Title document",
		},
		{
			name:  "single empty section — LLM filler",
			input: "## Section A",
			wantSections: []usecase.MdSection{
				{Title: "Section A", RawBody: ""},
			},
		},
		{
			name:  "single filled section — raw body preserved",
			input: "## Section A\nBody text line 1\nBody text line 2",
			wantSections: []usecase.MdSection{
				{Title: "Section A", RawBody: "Body text line 1\nBody text line 2"},
			},
		},
		{
			name:  "meta key-value pairs",
			input: "# Title\n\n- **client:** ACME Inc.\n- **price:** 1 200 000\n",
			wantMeta: map[string]string{
				"client": "ACME Inc.",
				"price":  "1 200 000",
			},
			wantTitle: "Title",
		},
		{
			name: "hybrid: mix of empty and filled sections",
			input: strings.Join([]string{
				"# Главный заголовок",
				"",
				"- **client:** Acme",
				"",
				"## Введение",
				"",
				"## Стоимость",
				"Конкретная цифра уже от заказчика: 1 200 000 рублей.",
				"",
				"## Условия",
			}, "\n"),
			wantTitle: "Главный заголовок",
			wantMeta:  map[string]string{"client": "Acme"},
			wantSections: []usecase.MdSection{
				{Title: "Введение", RawBody: ""},
				{Title: "Стоимость", RawBody: "Конкретная цифра уже от заказчика: 1 200 000 рублей."},
				{Title: "Условия", RawBody: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := usecase.ParseMarkdownTemplate([]byte(tt.input))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want wraps %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", got.Title, tt.wantTitle)
			}
			if len(tt.wantMeta) > 0 {
				for k, v := range tt.wantMeta {
					if got.Meta[k] != v {
						t.Errorf("Meta[%q] = %q, want %q", k, got.Meta[k], v)
					}
				}
			}
			if len(got.Sections) != len(tt.wantSections) {
				t.Fatalf("Sections len = %d, want %d (got: %+v)", len(got.Sections), len(tt.wantSections), got.Sections)
			}
			for i, want := range tt.wantSections {
				if got.Sections[i].Title != want.Title {
					t.Errorf("Sections[%d].Title = %q, want %q", i, got.Sections[i].Title, want.Title)
				}
				if strings.TrimSpace(got.Sections[i].RawBody) != strings.TrimSpace(want.RawBody) {
					t.Errorf("Sections[%d].RawBody = %q, want %q", i, got.Sections[i].RawBody, want.RawBody)
				}
			}
		})
	}
}
