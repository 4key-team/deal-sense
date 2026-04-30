package mdstrip

import (
	"regexp"
	"strings"
)

var headingRe = regexp.MustCompile(`^#{1,6}\s+`)
var boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)
var italicRe = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
var tableSepRe = regexp.MustCompile(`^\|[-\s|:]+\|$`)
var tableRowRe = regexp.MustCompile(`^\|(.+)\|$`)
var linkRe = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
var refLinkRe = regexp.MustCompile(`\[([^\]]+)\]\[[^\]]*\]`)

func Strip(line string) string {
	trimmed := strings.TrimSpace(line)

	if tableSepRe.MatchString(trimmed) {
		return ""
	}

	if m := tableRowRe.FindStringSubmatch(trimmed); m != nil {
		cells := strings.Split(m[1], "|")
		var parts []string
		for _, c := range cells {
			c = strings.TrimSpace(c)
			if c != "" {
				parts = append(parts, c)
			}
		}
		return strings.Join(parts, " — ")
	}

	trimmed = headingRe.ReplaceAllString(trimmed, "")
	trimmed = boldRe.ReplaceAllString(trimmed, "$1")

	trimmed = italicRe.ReplaceAllStringFunc(trimmed, func(match string) string {
		sub := italicRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		prefix := ""
		suffix := ""
		if len(match) > 0 && match[0] != '*' {
			prefix = string(match[0])
		}
		if len(match) > 0 && match[len(match)-1] != '*' {
			suffix = string(match[len(match)-1])
		}
		return prefix + sub[1] + suffix
	})

	trimmed = linkRe.ReplaceAllString(trimmed, "$1")
	trimmed = refLinkRe.ReplaceAllString(trimmed, "$1")

	return trimmed
}
