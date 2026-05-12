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

// headingKey is the canonical comparable form of a heading string. Two
// headings match iff their headingKey values are equal — never compare
// raw strings directly. The type is unexported so callers must go
// through newHeadingKey, which prevents accidental raw-string comparisons
// in future code.
type headingKey string

// newHeadingKey produces a canonical comparison key for a heading by:
//
//  1. stripping a leading numeric prefix (`1.`, `2)`, `10. `…)
//  2. collapsing internal whitespace runs to a single space
//  3. lower-casing
//  4. trimming leading/trailing space
//
// Substring matches are intentionally NOT supported — that would create
// false positives when one section title contains another ("Цели" vs
// "Цели проекта").
func newHeadingKey(s string) headingKey {
	s = leadingNumberRe.ReplaceAllString(s, "")
	s = multiSpaceRe.ReplaceAllString(s, " ")
	return headingKey(strings.ToLower(strings.TrimSpace(s)))
}
