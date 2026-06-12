package translation

import (
	"regexp"

	"github.com/bloodf/g0router/internal/schemas"
)

// dedupRule describes one trigger-based stripping rule. When any tool name
// matches one of the triggers, every tool name matching one of the strip
// patterns is removed from the request.
type dedupRule struct {
	triggers []*regexp.Regexp
	strip    []*regexp.Regexp
}

// dedupRules is the Go port of DEDUP_RULES from toolDeduper.js:6-22.
var dedupRules = []dedupRule{
	{
		// Exa MCP present → drop built-in web tools (Exa is preferred).
		triggers: mustCompilePatterns("mcp__exa__web_search_exa", "mcp__exa__web_fetch_exa"),
		strip:    mustCompilePatterns("WebSearch", "WebFetch", "mcp__workspace__web_fetch"),
	},
	{
		// Tavily MCP present → drop built-in web tools.
		triggers: mustCompilePatterns("mcp__tavily__tavily_search", "mcp__tavily__tavily_extract"),
		strip:    mustCompilePatterns("WebSearch", "WebFetch", "mcp__workspace__web_fetch"),
	},
	{
		// Browser MCP present → drop Cowork's duplicate Claude_in_Chrome connector.
		triggers: []*regexp.Regexp{regexp.MustCompile(`^mcp__browsermcp__`)},
		strip:    []*regexp.Regexp{regexp.MustCompile(`^mcp__Claude_in_Chrome__`)},
	},
}

func mustCompilePatterns(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		out[i] = regexp.MustCompile(regexp.QuoteMeta(p))
	}
	return out
}

// toolName returns the callable name of a tool, preferring the top-level name
// and falling back to function.name.
func toolName(t schemas.Tool) string {
	if t.Function.Name != "" {
		return t.Function.Name
	}
	return ""
}

// matches reports whether name matches pattern.
func matches(name string, pattern *regexp.Regexp) bool {
	return pattern.MatchString(name)
}

// DedupeTools strips duplicate/built-in tools when equivalent MCP tools are
// present, and also removes later duplicate definitions that share the same
// name (keeping the last occurrence). It mutates req.Tools in place.
func DedupeTools(req *schemas.ChatRequest) {
	if len(req.Tools) == 0 {
		return
	}

	// First pass: deduplicate by name, keeping the last occurrence.
	seen := make(map[string]int)
	for i, t := range req.Tools {
		seen[toolName(t)] = i
	}
	if len(seen) < len(req.Tools) {
		unique := make([]schemas.Tool, 0, len(seen))
		// Preserve original order, but only keep the last occurrence of each name.
		keep := make(map[int]bool, len(seen))
		for _, idx := range seen {
			keep[idx] = true
		}
		for i, t := range req.Tools {
			if keep[i] {
				unique = append(unique, t)
			}
		}
		req.Tools = unique
	}

	// Second pass: apply trigger-based stripping rules.
	names := make([]string, len(req.Tools))
	for i, t := range req.Tools {
		names[i] = toolName(t)
	}

	toStrip := make(map[string]struct{})
	for _, rule := range dedupRules {
		hasTrigger := false
		for _, name := range names {
			for _, pat := range rule.triggers {
				if matches(name, pat) {
					hasTrigger = true
					break
				}
			}
			if hasTrigger {
				break
			}
		}
		if !hasTrigger {
			continue
		}
		for _, name := range names {
			for _, pat := range rule.strip {
				if matches(name, pat) {
					toStrip[name] = struct{}{}
					break
				}
			}
		}
	}

	if len(toStrip) == 0 {
		return
	}
	filtered := make([]schemas.Tool, 0, len(req.Tools))
	for _, t := range req.Tools {
		if _, drop := toStrip[toolName(t)]; !drop {
			filtered = append(filtered, t)
		}
	}
	req.Tools = filtered
}
