package parser

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	docx "github.com/mmonterroca/docxgo/v2"
	docxdomain "github.com/mmonterroca/docxgo/v2/domain"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

// DocxGenerative fills DOCX templates in generative mode — replacing paragraph content
// by matching section titles to headings, and appending unmatched sections at the end.
// Uses docxgo v2 for proper OOXML handling (preserves headers, footers, styles).
type DocxGenerative struct{}

func NewDocxGenerative() *DocxGenerative {
	return &DocxGenerative{}
}

func (g *DocxGenerative) GenerativeFill(_ context.Context, template []byte, sections []usecase.ContentSection) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("generative fill: %w", domain.ErrEmptyTemplate)
	}

	doc, err := docx.OpenDocumentFromBytes(template)
	if err != nil {
		return nil, fmt.Errorf("generative fill: %w", err)
	}

	if len(sections) == 0 {
		var buf bytes.Buffer
		if _, err := doc.WriteTo(&buf); err != nil {
			return nil, fmt.Errorf("generative fill: write: %w", err)
		}
		return buf.Bytes(), nil
	}

	remaining := make(map[string]usecase.ContentSection, len(sections))
	for _, s := range sections {
		remaining[strings.ToLower(strings.TrimSpace(s.Title))] = s
	}

	paras := doc.Paragraphs()

	for i, para := range paras {
		text := strings.TrimSpace(para.Text())
		if text == "" {
			continue
		}
		titleLower := strings.ToLower(text)

		sec, ok := remaining[titleLower]
		if !ok {
			continue
		}
		delete(remaining, titleLower)

		// Replace the next paragraph's content (the one after the heading).
		if i+1 < len(paras) {
			contentPara := paras[i+1]
			contentPara.ClearRuns()
			g.fillParagraphWithContent(contentPara, sec.Content)
		}
	}

	// Append unmatched sections at end.
	for _, sec := range sections {
		if _, ok := remaining[strings.ToLower(strings.TrimSpace(sec.Title))]; !ok {
			continue
		}
		heading, _ := doc.AddParagraph()
		heading.SetStyle(docxdomain.StyleIDHeading1)
		hr, _ := heading.AddRun()
		hr.SetText(sec.Title)

		g.addContentParagraphs(doc, sec.Content)
	}

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("generative fill: write: %w", err)
	}
	return buf.Bytes(), nil
}

// fillParagraphWithContent writes multiline content into a single paragraph using line breaks.
func (g *DocxGenerative) fillParagraphWithContent(para docxdomain.Paragraph, content string) {
	first := true
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !first {
			br, _ := para.AddRun()
			br.AddBreak(docxdomain.BreakTypeLine)
		}
		r, _ := para.AddRun()
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			r.SetText("• " + strings.TrimSpace(trimmed[2:]))
		} else {
			r.SetText(trimmed)
		}
		first = false
	}
}

func (g *DocxGenerative) addContentParagraphs(doc docxdomain.Document, content string) {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		p, _ := doc.AddParagraph()

		switch {
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
			p.SetStyle(docxdomain.StyleIDListParagraph)
			r, _ := p.AddRun()
			r.SetText(strings.TrimSpace(trimmed[2:]))
		default:
			r, _ := p.AddRun()
			r.SetText(trimmed)
		}
	}
}

func (g *DocxGenerative) GenerateClean(_ context.Context, input usecase.ContentInput) ([]byte, error) {
	doc := docx.NewDocument()

	if input.Summary != "" {
		p, _ := doc.AddParagraph()
		p.SetStyle(docxdomain.StyleIDTitle)
		r, _ := p.AddRun()
		r.SetText(input.Summary)
	}

	for _, sec := range input.Sections {
		heading, _ := doc.AddParagraph()
		heading.SetStyle(docxdomain.StyleIDHeading1)
		hr, _ := heading.AddRun()
		hr.SetText(sec.Title)

		g.addContentParagraphs(doc, sec.Content)
	}

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("generate clean: write: %w", err)
	}
	return buf.Bytes(), nil
}

// Ensure DocxGenerative implements GenerativeEngine at compile time.
var _ usecase.GenerativeEngine = (*DocxGenerative)(nil)

