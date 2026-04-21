package parser

import (
	"regexp"
	"strings"
)

// runRe matches a single <w:r>...</w:r> element (non-greedy).
var runRe = regexp.MustCompile(`<w:r\b[^>]*>[\s\S]*?</w:r>`)

// textRe extracts the text content from <w:t ...>...</w:t>.
var textRe = regexp.MustCompile(`<w:t[^>]*>([\s\S]*?)</w:t>`)

// mergePlaceholderRuns finds {{placeholder}} tokens split across
// consecutive <w:r> runs in OOXML and merges them into one run.
//
// Word/LibreOffice may arbitrarily split text like "{{name}}" into
// multiple runs: "<w:r><w:t>{{</w:t></w:r><w:r><w:t>name}}</w:t></w:r>".
// go-docx cannot match these — this function normalizes them first.
func mergePlaceholderRuns(xml string) string {
	runs := runRe.FindAllStringIndex(xml, -1)
	if len(runs) < 2 {
		return xml
	}

	type runInfo struct {
		start, end int
		text       string
		full       string
	}

	infos := make([]runInfo, len(runs))
	for i, loc := range runs {
		full := xml[loc[0]:loc[1]]
		text := ""
		if m := textRe.FindStringSubmatch(full); len(m) > 1 {
			text = m[1]
		}
		infos[i] = runInfo{start: loc[0], end: loc[1], text: text, full: full}
	}

	// Walk through runs looking for incomplete {{ without closing }}.
	var result strings.Builder
	prev := 0

	for i := 0; i < len(infos); {
		ri := infos[i]

		// Check if this run's text contains an unmatched "{{".
		openIdx := strings.LastIndex(ri.text, "{{")
		closeIdx := strings.LastIndex(ri.text, "}}")

		if openIdx == -1 || (closeIdx != -1 && closeIdx > openIdx) {
			// No unmatched open — emit as-is.
			result.WriteString(xml[prev:ri.end])
			prev = ri.end
			i++
			continue
		}

		// Found unmatched {{ — accumulate subsequent runs until }}.
		merged := ri.text
		lastMerged := i
		for j := i + 1; j < len(infos); j++ {
			merged += infos[j].text
			lastMerged = j
			if strings.Contains(infos[j].text, "}}") {
				break
			}
		}

		// Emit everything before this run group unchanged.
		result.WriteString(xml[prev:ri.start])

		// Build merged run: keep first run's structure, replace text.
		firstRun := ri.full
		newRun := textRe.ReplaceAllStringFunc(firstRun, func(_ string) string {
			// Preserve xml:space attribute if present.
			orig := textRe.FindString(firstRun)
			tagEnd := strings.Index(orig, ">")
			openTag := orig[:tagEnd+1]
			return openTag + merged + "</w:t>"
		})
		result.WriteString(newRun)

		prev = infos[lastMerged].end
		i = lastMerged + 1
	}

	result.WriteString(xml[prev:])
	return result.String()
}
