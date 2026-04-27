package parser

import (
	"context"
	"fmt"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// MarkdownRenderer generates Markdown output from proposal data.
type MarkdownRenderer struct{}

func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

func (r *MarkdownRenderer) Render(_ context.Context, input usecase.ContentInput) ([]byte, error) {
	var b strings.Builder

	title := "Коммерческое предложение"
	if input.Summary != "" {
		title = input.Summary
	}
	fmt.Fprintf(&b, "# %s\n\n", title)

	// Meta table
	meta := input.Meta
	if meta == nil {
		meta = map[string]string{}
	}
	if len(meta) > 0 {
		for _, key := range []string{"client", "project", "price", "timeline", "date"} {
			if v, ok := meta[key]; ok {
				fmt.Fprintf(&b, "- **%s:** %s\n", key, v)
			}
		}
		b.WriteString("\n---\n\n")
	}

	// Sections
	for _, sec := range input.Sections {
		fmt.Fprintf(&b, "## %s\n\n%s\n\n", sec.Title, sec.Content)
	}

	return []byte(b.String()), nil
}

// Ensure MarkdownRenderer implements MDGenerator at compile time.
var _ usecase.MDGenerator = (*MarkdownRenderer)(nil)
