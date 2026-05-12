package parser

import (
	"regexp"
	"strings"
)

var (
	// leadingNumberRe matches numbered-list prefixes like "1.", "2)",
	// "10. " — optionally surrounded by whitespace. Templates often carry
	// numeric prefixes that the LLM omits in its section titles.
	leadingNumberRe = regexp.MustCompile(`^\s*\d+\s*[.)]\s*`)
	// multiSpaceRe collapses runs of whitespace (including non-breaking
	// space inside Word documents) into a single ASCII space.
	multiSpaceRe = regexp.MustCompile(`\s+`)
)

// normalizeHeading prepares a heading string for fuzzy equality matching
// between a template paragraph and a section title from the LLM.
// Transformations, in order:
//
//  1. strip a leading numeric prefix (`1.`, `2)`, `10. `…)
//  2. collapse internal whitespace runs to a single space
//  3. lower-case
//  4. trim leading/trailing space
//
// Two strings that produce the same normalized form are treated as the
// same heading. Substring matches are intentionally NOT supported —
// that would introduce false positives when one section title contains
// another ("Цели" vs "Цели проекта").
func normalizeHeading(s string) string {
	s = leadingNumberRe.ReplaceAllString(s, "")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return strings.ToLower(strings.TrimSpace(s))
}
