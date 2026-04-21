package parser

import (
	"testing"
)

func TestMergePlaceholderRuns(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single run placeholder unchanged",
			in:   `<w:r><w:t>{{name}}</w:t></w:r>`,
			want: `<w:r><w:t>{{name}}</w:t></w:r>`,
		},
		{
			name: "split across two runs",
			in:   `<w:r><w:t>{{</w:t></w:r><w:r><w:t>name}}</w:t></w:r>`,
			want: `<w:r><w:t>{{name}}</w:t></w:r>`,
		},
		{
			name: "split across three runs",
			in:   `<w:r><w:t>{{</w:t></w:r><w:r><w:t>company</w:t></w:r><w:r><w:t>_name}}</w:t></w:r>`,
			want: `<w:r><w:t>{{company_name}}</w:t></w:r>`,
		},
		{
			name: "multiple placeholders one split",
			in:   `<w:r><w:t>Hello {{ok}}</w:t></w:r><w:r><w:t>{{</w:t></w:r><w:r><w:t>split}}</w:t></w:r>`,
			want: `<w:r><w:t>Hello {{ok}}</w:t></w:r><w:r><w:t>{{split}}</w:t></w:r>`,
		},
		{
			name: "preserves rPr in first run",
			in:   `<w:r><w:rPr><w:b/></w:rPr><w:t>{{</w:t></w:r><w:r><w:t>bold}}</w:t></w:r>`,
			want: `<w:r><w:rPr><w:b/></w:rPr><w:t>{{bold}}</w:t></w:r>`,
		},
		{
			name: "no placeholders unchanged",
			in:   `<w:r><w:t>Hello world</w:t></w:r>`,
			want: `<w:r><w:t>Hello world</w:t></w:r>`,
		},
		{
			name: "xml:space preserved",
			in:   `<w:r><w:t xml:space="preserve">{{</w:t></w:r><w:r><w:t>x}}</w:t></w:r>`,
			want: `<w:r><w:t xml:space="preserve">{{x}}</w:t></w:r>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergePlaceholderRuns(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}
