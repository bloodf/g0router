package mcp

import (
	"strings"
	"sync"
)

// toolsCache caches a probe's tools/list result keyed by instance/url so repeated
// agent injections reuse the discovered tools without re-probing (PAR-MCP-039).
type toolsCache struct {
	mu      sync.RWMutex
	entries map[string][]ProbeTool
}

// newToolsCache builds an empty cache.
func newToolsCache() *toolsCache {
	return &toolsCache{entries: make(map[string][]ProbeTool)}
}

// get returns the cached tools for key, if present.
func (c *toolsCache) get(key string) ([]ProbeTool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tools, ok := c.entries[key]
	return tools, ok
}

// set stores tools under key.
func (c *toolsCache) set(key string, tools []ProbeTool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = tools
}

// buildCompactManifest assembles a token-frugal one-line-per-tool listing for
// agent injection. PURE — "name: description" per line (description omitted when
// empty). An empty tool list yields an empty string.
func buildCompactManifest(tools []ProbeTool) string {
	if len(tools) == 0 {
		return ""
	}
	var b strings.Builder
	for i, t := range tools {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(t.Name)
		if t.Description != "" {
			b.WriteString(": ")
			b.WriteString(t.Description)
		}
	}
	return b.String()
}
