package mcp

import (
	"fmt"
	"strings"
)

// scopeTools narrows the global server-mode catalog to the tools a virtual key's
// executeOnlyTools patterns admit (PAR-BF-MCP-017/019, D4). It is PURE: clientOf
// reports the owning MCP client of a (bare) tool name so the "<client>-*" prefix
// wildcard and the exact "<client>-<tool>" form can be matched without the filter
// re-deriving prefix logic — it REUSES the shipped stripServerPrefix primitive
// (toolpolicy.go) to normalize each pattern against its owning client.
//
// Pattern vocabulary (the three matrix-evidenced forms only):
//   - len(patterns)==0 (nil or empty) -> deny-all (017: nil/[] = deny).
//   - "*"                              -> allow-all (017: ["*"] = all).
//   - "<client>-*"                     -> every tool whose owning client == <client>.
//   - "<client>-<tool>" or bare "<tool>" -> that exact tool, matched via the
//     bare-vs-prefixed dual shape stripServerPrefix normalizes (019).
func scopeTools(global []ServerTool, patterns []string, clientOf func(string) string) []ServerTool {
	if len(patterns) == 0 {
		return []ServerTool{}
	}
	for _, p := range patterns {
		if p == "*" {
			out := make([]ServerTool, len(global))
			copy(out, global)
			return out
		}
	}

	// Pre-split the "<client>-*" prefix wildcards from the exact patterns.
	prefixClients := map[string]bool{}
	exact := map[string]bool{}
	for _, p := range patterns {
		if client, ok := strings.CutSuffix(p, "-*"); ok && client != "" {
			prefixClients[client] = true
			continue
		}
		exact[p] = true
	}

	out := make([]ServerTool, 0, len(global))
	for _, tool := range global {
		owner := clientOf(tool.Name)
		if prefixClients[owner] {
			out = append(out, tool)
			continue
		}
		// Exact match against either the bare tool name or the "<client>-<tool>"
		// prefixed form, reusing stripServerPrefix to normalize the prefixed key
		// back to the bare name the catalog stores (no re-derived prefix logic).
		if exact[tool.Name] || exact[owner+"-"+tool.Name] || matchesExactPrefixed(exact, owner, tool.Name) {
			out = append(out, tool)
		}
	}
	return out
}

// matchesExactPrefixed reports whether any exact pattern, once its "<owner>-"
// prefix is stripped via the shipped stripServerPrefix, resolves to the bare
// tool name. This lets a pattern carrying a redundant prefix (e.g. repeated
// "<client>-<client>-<tool>") still match, mirroring the idempotent strip the
// CLIENT-mode policy uses.
func matchesExactPrefixed(exact map[string]bool, owner, bare string) bool {
	for p := range exact {
		if stripServerPrefix(owner, p) == bare {
			return true
		}
	}
	return false
}

// ScopeTools is the exported form of scopeTools for the admin transport's per-VK
// scoped CatalogSource (bf-mcp-2 D3/D4).
func ScopeTools(global []ServerTool, patterns []string, clientOf func(string) string) []ServerTool {
	return scopeTools(global, patterns, clientOf)
}

// ValidateAutoExecuteSubset is the exported form of validateAutoExecuteSubset for
// the admin transport's live assignment write path (bf-mcp-2 D5/049).
func ValidateAutoExecuteSubset(execute, autoExecute []string) error {
	return validateAutoExecuteSubset(execute, autoExecute)
}

// validateAutoExecuteSubset returns a non-nil error when any autoExecute entry is
// not admitted by the execute patterns (PAR-BF-MCP-018/049, D5). The two lists
// share the D4 pattern vocabulary: a "*" execute admits any auto-execute; an
// auto-execute entry otherwise requires a literal match in execute. PURE.
func validateAutoExecuteSubset(execute, autoExecute []string) error {
	if len(autoExecute) == 0 {
		return nil
	}
	allow := map[string]bool{}
	star := false
	for _, e := range execute {
		if e == "*" {
			star = true
		}
		allow[e] = true
	}
	if star {
		return nil
	}
	for _, a := range autoExecute {
		if !allow[a] {
			return fmt.Errorf("auto-execute tool %q is not a subset of the execute list", a)
		}
	}
	return nil
}
