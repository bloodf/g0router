package mcp

import (
	"regexp"
	"strings"
)

// maxToolResultChars is 9router's hard cap on a filtered tool-result text
// (stdioSseBridge.js:20 / smartFilterText). Output is truncated to exactly this
// many characters.
const maxToolResultChars = 50_000

// genericLineRe matches a noise "generic" accessibility node — either a bare
// `generic` role line or a `generic "..."` line. These carry no signal and are
// dropped (mirrors 9router's smartFilterText regex over `generic`).
var genericLineRe = regexp.MustCompile(`^\s*generic(\s+".*")?\s*$`)

// emptyTextLineRe matches an empty `text ""` line (whitespace-only text), which
// is dropped as noise.
var emptyTextLineRe = regexp.MustCompile(`^\s*text\s+"\s*"\s*$`)

// smartFilterText drops noise nodes (generic roles, empty text lines), collapses
// consecutive identical role-prefixed sibling lines to a single occurrence, and
// hard-truncates the result at maxToolResultChars. PURE — no I/O. Mirrors
// 9router's smartFilterText + collapseRepeated (stdioSseBridge.js:20).
//
// It operates on the line-oriented accessibility-tree text 9router produces; the
// observable behavior (drop generic + empty text; collapse repeated siblings;
// 50K cap; clean short input unchanged) is pinned by filter_test.go.
func smartFilterText(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	var prev string
	havePrev := false
	for _, line := range lines {
		if genericLineRe.MatchString(line) || emptyTextLineRe.MatchString(line) {
			continue
		}
		// collapseRepeated: skip a line identical to the immediately preceding
		// kept line (consecutive duplicate sibling).
		if havePrev && line == prev {
			continue
		}
		out = append(out, line)
		prev = line
		havePrev = true
	}
	result := strings.Join(out, "\n")
	if len(result) > maxToolResultChars {
		result = result[:maxToolResultChars]
	}
	return result
}
