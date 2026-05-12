package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	docx "github.com/mmonterroca/docxgo/v2"
	docxdomain "github.com/mmonterroca/docxgo/v2/domain"

	"github.com/daniil/deal-sense/backend/internal/adapter/mdstrip"
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
		// docxgo can't handle some templates (themes, complex drawings).
		// Fall back to zip-based XML manipulation.
		return g.generativeFillZip(template, sections)
	}

	if len(sections) == 0 {
		var buf bytes.Buffer
		if _, err := doc.WriteTo(&buf); err != nil {
			return nil, fmt.Errorf("generative fill: write: %w", err)
		}
		return buf.Bytes(), nil
	}

	remaining := make(map[headingKey]usecase.ContentSection, len(sections))
	for _, s := range sections {
		remaining[newHeadingKey(s.Title)] = s
	}

	paras := doc.Paragraphs()

	for i, para := range paras {
		text := strings.TrimSpace(para.Text())
		if text == "" {
			continue
		}
		key := newHeadingKey(text)

		sec, ok := remaining[key]
		if !ok {
			continue
		}
		delete(remaining, key)

		// Replace the next paragraph's content (the one after the heading).
		if i+1 < len(paras) {
			contentPara := paras[i+1]
			contentPara.ClearRuns()
			g.fillParagraphWithContent(contentPara, sec.Content)
		}
	}

	// Append unmatched sections at end.
	for _, sec := range sections {
		if _, ok := remaining[newHeadingKey(sec.Title)]; !ok {
			continue
		}
		heading, _ := doc.AddParagraph()
		heading.SetStyle(docxdomain.StyleIDHeading1)
		hr, _ := heading.AddRun()
		_ = hr.SetText(sec.Title) //nolint:errcheck // docxgo SetText returns nil on detached runs only

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
		isBullet := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
		cleaned := mdstrip.Strip(trimmed)
		if cleaned == "" {
			continue
		}
		if !first {
			br, _ := para.AddRun()
			br.AddBreak(docxdomain.BreakTypeLine)
		}
		r, _ := para.AddRun()
		if isBullet {
			r.SetText("• " + bulletText(cleaned))
		} else {
			r.SetText(cleaned)
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
		isBullet := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
		cleaned := mdstrip.Strip(trimmed)
		if cleaned == "" {
			continue
		}

		p, _ := doc.AddParagraph()

		switch {
		case isBullet:
			p.SetStyle(docxdomain.StyleIDListParagraph)
			r, _ := p.AddRun()
			r.SetText(bulletText(cleaned))
		default:
			r, _ := p.AddRun()
			r.SetText(cleaned)
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
		_ = hr.SetText(sec.Title) //nolint:errcheck // docxgo SetText returns nil on detached runs only

		g.addContentParagraphs(doc, sec.Content)
	}

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("generate clean: write: %w", err)
	}
	return buf.Bytes(), nil
}

// --- Zip-based fallback for templates that docxgo cannot open ---

var paragraphRe = regexp.MustCompile(`<w:p\b[^>]*>[\s\S]*?</w:p>`)
var textContentRe = regexp.MustCompile(`<w:t[^>]*>([\s\S]*?)</w:t>`)

func (g *DocxGenerative) generativeFillZip(template []byte, sections []usecase.ContentSection) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, fmt.Errorf("generative fill: %w", err)
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		content := mustReadZipEntry(f)

		if f.Name == "word/document.xml" {
			content = g.injectSectionsXML(string(content), sections)
		}
		if f.Name == "word/styles.xml" && hasBulletContent(sections) {
			content = ensureListBulletStyle(content)
		}

		header := f.FileHeader
		header.UncompressedSize64 = uint64(len(content))
		fw, _ := w.CreateHeader(&header)
		_, _ = fw.Write(content) //nolint:errcheck // write to in-memory bytes.Buffer
	}

	_ = w.Close() //nolint:errcheck // close of in-memory zip writer
	return buf.Bytes(), nil
}

func (g *DocxGenerative) injectSectionsXML(xml string, sections []usecase.ContentSection) []byte {
	if len(sections) == 0 {
		return []byte(xml)
	}

	remaining := make(map[headingKey]usecase.ContentSection, len(sections))
	for _, s := range sections {
		remaining[newHeadingKey(s.Title)] = s
	}

	paras := paragraphRe.FindAllStringIndex(xml, -1)

	type paraInfo struct {
		start, end int
		text       string
	}
	infos := make([]paraInfo, len(paras))
	for i, loc := range paras {
		raw := xml[loc[0]:loc[1]]
		infos[i] = paraInfo{start: loc[0], end: loc[1], text: extractParagraphText(raw)}
	}

	replaced := make(map[int]string)
	for _, sec := range sections {
		key := newHeadingKey(sec.Title)
		for i, pi := range infos {
			if newHeadingKey(pi.text) != key {
				continue
			}
			delete(remaining, key)
			if i+1 < len(infos) {
				replaced[i+1] = buildParagraphsXML(sec.Content)
			}
			break
		}
	}

	var result strings.Builder
	prev := 0
	for i, pi := range infos {
		result.WriteString(xml[prev:pi.start])
		if repl, ok := replaced[i]; ok {
			result.WriteString(repl)
		} else {
			result.WriteString(xml[pi.start:pi.end])
		}
		prev = pi.end
	}
	result.WriteString(xml[prev:])

	out := result.String()

	if len(remaining) > 0 {
		var appended strings.Builder
		for _, sec := range sections {
			if _, ok := remaining[newHeadingKey(sec.Title)]; !ok {
				continue
			}
			appended.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>` +
				escapeXML(sec.Title) + `</w:t></w:r></w:p>`)
			appended.WriteString(buildParagraphsXML(sec.Content))
		}
		insertAt := findFooterInsertPoint(out)
		if insertAt >= 0 {
			out = out[:insertAt] + appended.String() + out[insertAt:]
		}
	}

	out = stripEmptyBodyParagraphs(out)

	return []byte(out)
}

func extractParagraphText(paraXML string) string {
	matches := textContentRe.FindAllStringSubmatch(paraXML, -1)
	var b strings.Builder
	for _, m := range matches {
		b.WriteString(m[1])
	}
	return b.String()
}

func buildParagraphsXML(content string) string {
	var b strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		isBullet := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
		cleaned := mdstrip.Strip(trimmed)
		if cleaned == "" {
			continue
		}
		switch {
		case isBullet:
			text := bulletText(cleaned)
			b.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListBullet"/></w:pPr>`)
			b.WriteString(`<w:r><w:t xml:space="preserve">` + escapeXML(text) + `</w:t></w:r></w:p>`)
		default:
			b.WriteString(`<w:p><w:r><w:t xml:space="preserve">` + escapeXML(cleaned) + `</w:t></w:r></w:p>`)
		}
	}
	return b.String()
}

func hasBulletContent(sections []usecase.ContentSection) bool {
	for _, s := range sections {
		for _, line := range strings.Split(s.Content, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
				return true
			}
		}
	}
	return false
}

func bulletText(cleaned string) string {
	if len(cleaned) > 2 && (strings.HasPrefix(cleaned, "- ") || strings.HasPrefix(cleaned, "* ")) {
		return strings.TrimSpace(cleaned[2:])
	}
	return strings.TrimSpace(cleaned)
}

var footerMarkerRe = regexp.MustCompile(`(?i)(с\s+уважением|действительно\s+\d|regards|sincerely)`)

func findFooterInsertPoint(xml string) int {
	paras := paragraphRe.FindAllStringIndex(xml, -1)
	const tailCount = 10
	start := 0
	if len(paras) > tailCount {
		start = len(paras) - tailCount
	}
	for _, loc := range paras[start:] {
		text := extractParagraphText(xml[loc[0]:loc[1]])
		if footerMarkerRe.MatchString(text) {
			return loc[0]
		}
	}
	insertAt := strings.LastIndex(xml, "<w:sectPr")
	if insertAt < 0 {
		insertAt = strings.LastIndex(xml, "</w:body>")
	}
	return insertAt
}

func stripEmptyBodyParagraphs(xml string) string {
	return paragraphRe.ReplaceAllStringFunc(xml, func(para string) string {
		if textContentRe.MatchString(para) {
			return para
		}
		if strings.Contains(para, "<w:drawing") || strings.Contains(para, "<mc:") ||
			strings.Contains(para, "<wps:") || strings.Contains(para, "<wpg:") ||
			strings.Contains(para, "<w:spacing") {
			return para
		}
		return ""
	})
}

func ensureListBulletStyle(stylesXML []byte) []byte {
	s := string(stylesXML)
	if strings.Contains(s, `w:styleId="ListBullet"`) {
		return stylesXML
	}
	closeTag := strings.LastIndex(s, "</w:styles>")
	if closeTag < 0 {
		return stylesXML
	}
	return []byte(s[:closeTag] +
		`<w:style w:type="paragraph" w:styleId="ListBullet">` +
		`<w:name w:val="List Bullet"/>` +
		`<w:basedOn w:val="Normal"/>` +
		`<w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr>` +
		`</w:style>` + s[closeTag:])
}

// Ensure DocxGenerative implements GenerativeEngine at compile time.
var _ usecase.GenerativeEngine = (*DocxGenerative)(nil)
