package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
	"github.com/daniil/deal-sense/backend/internal/usecase"
)

var paragraphRe = regexp.MustCompile(`<w:p\b[^>]*>[\s\S]*?</w:p>`)
var textContentRe = regexp.MustCompile(`<w:t[^>]*>([\s\S]*?)</w:t>`)

// DocxGenerative fills DOCX templates in generative mode — replacing paragraph content
// by matching section titles to headings, and appending unmatched sections at the end.
type DocxGenerative struct{}

func NewDocxGenerative() *DocxGenerative {
	return &DocxGenerative{}
}

func (g *DocxGenerative) GenerativeFill(_ context.Context, template []byte, sections []usecase.ContentSection) ([]byte, error) {
	if len(template) == 0 {
		return nil, fmt.Errorf("generative fill: %w", domain.ErrEmptyTemplate)
	}

	r, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, fmt.Errorf("generative fill: %w", err)
	}

	needsBulletStyle := hasBulletContent(sections)

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		content := mustReadZipEntry(f)

		if f.Name == "word/document.xml" {
			content = g.injectSections(string(content), sections)
		}
		if f.Name == "word/styles.xml" && needsBulletStyle {
			content = ensureListBulletStyle(content)
		}

		header := f.FileHeader
		header.UncompressedSize64 = uint64(len(content))
		fw, _ := w.CreateHeader(&header)
		fw.Write(content)
	}

	w.Close()
	return buf.Bytes(), nil
}

// injectSections finds headings in the document XML and replaces following paragraphs
// with section content. Unmatched sections are appended before </w:body>.
func (g *DocxGenerative) injectSections(xml string, sections []usecase.ContentSection) []byte {
	if len(sections) == 0 {
		return []byte(xml)
	}

	remaining := make(map[string]usecase.ContentSection, len(sections))
	for _, s := range sections {
		remaining[strings.ToLower(s.Title)] = s
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
		titleLower := strings.ToLower(strings.TrimSpace(sec.Title))
		for i, pi := range infos {
			if strings.ToLower(strings.TrimSpace(pi.text)) != titleLower {
				continue
			}
			delete(remaining, titleLower)
			if i+1 < len(infos) {
				replaced[i+1] = buildParagraphs(sec.Content)
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
			if _, ok := remaining[strings.ToLower(sec.Title)]; !ok {
				continue
			}
			appended.WriteString(buildHeadingParagraph(sec.Title))
			appended.WriteString(buildParagraphs(sec.Content))
		}
		insertAt := strings.LastIndex(out, "<w:sectPr")
		if insertAt < 0 {
			insertAt = strings.LastIndex(out, "</w:body>")
		}
		if insertAt >= 0 {
			out = out[:insertAt] + appended.String() + out[insertAt:]
		}
	}

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

func buildParagraphs(content string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
			text := strings.TrimSpace(trimmed[2:])
			b.WriteString(`<w:p><w:pPr><w:pStyle w:val="ListBullet"/></w:pPr>`)
			b.WriteString(`<w:r><w:t xml:space="preserve">` + escapeXML(text) + `</w:t></w:r></w:p>`)
		default:
			b.WriteString(`<w:p><w:r><w:t xml:space="preserve">` + escapeXML(trimmed) + `</w:t></w:r></w:p>`)
		}
	}
	return b.String()
}

func buildHeadingParagraph(title string) string {
	return `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:rPr><w:b/></w:rPr><w:t>` +
		escapeXML(title) + `</w:t></w:r></w:p>`
}

const listBulletStyleDef = `<w:style w:type="paragraph" w:styleId="ListBullet">` +
	`<w:name w:val="List Bullet"/>` +
	`<w:basedOn w:val="Normal"/>` +
	`<w:pPr><w:ind w:left="720" w:hanging="360"/></w:pPr>` +
	`</w:style>`

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

func ensureListBulletStyle(stylesXML []byte) []byte {
	s := string(stylesXML)
	if strings.Contains(s, `w:styleId="ListBullet"`) {
		return stylesXML
	}
	closeTag := strings.LastIndex(s, "</w:styles>")
	if closeTag < 0 {
		return stylesXML
	}
	return []byte(s[:closeTag] + listBulletStyleDef + s[closeTag:])
}

func (g *DocxGenerative) GenerateClean(_ context.Context, _ usecase.ContentInput) ([]byte, error) {
	return nil, fmt.Errorf("generate clean: not yet implemented")
}

// Ensure DocxGenerative implements GenerativeEngine at compile time.
var _ usecase.GenerativeEngine = (*DocxGenerative)(nil)

