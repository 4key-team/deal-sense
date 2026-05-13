package usecase

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/daniil/deal-sense/backend/internal/domain"
)

// MdSection is a parsed `##` heading with the lines that followed it
// in the source until the next `##`. RawBody empty signals the section
// should be LLM-filled; non-empty signals it must be passed through
// verbatim.
type MdSection struct {
	Title   string
	RawBody string
}

// MdTemplate is the parsed shape of a user-supplied markdown template.
// Title comes from the first `#` heading (optional). Meta captures
// `- **key:** value` lines that appear before any `##`. Sections is
// the ordered list of `##` headings with their bodies.
type MdTemplate struct {
	Title    string
	Meta     map[string]string
	Sections []MdSection
}

var (
	mdHeading1   = regexp.MustCompile(`^#\s+(.+?)\s*$`)
	mdHeading2   = regexp.MustCompile(`^##\s+(.+?)\s*$`)
	mdMetaLine   = regexp.MustCompile(`^\s*[-*]\s+\*\*([^:]+):\*\*\s+(.+?)\s*$`)
	mdHeadingAny = regexp.MustCompile(`^#{3,}\s+`)
)

// ParseMarkdownTemplate splits a markdown document into Title/Meta/Sections.
// It does not interpret bold/italic or any inline markdown — only the
// structural elements relevant to template-driven generation.
func ParseMarkdownTemplate(data []byte) (MdTemplate, error) {
	if len(data) == 0 {
		return MdTemplate{}, fmt.Errorf("parse markdown template: %w", domain.ErrEmptyTemplate)
	}

	var (
		tmpl       MdTemplate
		curBody    strings.Builder
		curTitle   string
		inSection  bool
		seenTitle  bool
		metaParsed bool
	)

	commitSection := func() {
		if !inSection {
			return
		}
		tmpl.Sections = append(tmpl.Sections, MdSection{
			Title:   curTitle,
			RawBody: strings.TrimRight(curBody.String(), "\n"),
		})
		curBody.Reset()
		curTitle = ""
		inSection = false
	}

	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")

		if m := mdHeading2.FindStringSubmatch(line); m != nil {
			commitSection()
			curTitle = m[1]
			inSection = true
			metaParsed = true // meta is only collected before the first ##
			continue
		}

		if !inSection && !seenTitle {
			if m := mdHeading1.FindStringSubmatch(line); m != nil {
				tmpl.Title = m[1]
				seenTitle = true
				continue
			}
		}

		if !inSection && !metaParsed {
			if m := mdMetaLine.FindStringSubmatch(line); m != nil {
				if tmpl.Meta == nil {
					tmpl.Meta = map[string]string{}
				}
				tmpl.Meta[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
				continue
			}
		}

		if inSection {
			// Skip the immediate blank line right after the heading so
			// "## H\n\nbody" keeps RawBody = "body" (cleaner downstream).
			if curBody.Len() == 0 && strings.TrimSpace(line) == "" {
				continue
			}
			// Pass-through every other line, including deeper headings.
			if mdHeadingAny.MatchString(line) {
				curBody.WriteString(line)
				curBody.WriteByte('\n')
				continue
			}
			curBody.WriteString(line)
			curBody.WriteByte('\n')
		}
	}
	commitSection()

	return tmpl, nil
}

// IsMarkdownTemplate returns true when the filename's extension marks
// it as a markdown template input. Used by the proposal handler to
// route into the markdown-mode branch before invoking the parser.
func IsMarkdownTemplate(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}
