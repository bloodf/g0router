package mcp

import (
	"strings"
	"testing"
)

func TestToolsCacheGetSet(t *testing.T) {
	c := newToolsCache()
	if _, ok := c.get("inst-1"); ok {
		t.Fatalf("empty cache should miss")
	}
	tools := []ProbeTool{{Name: "search", Description: "web search"}}
	c.set("inst-1", tools)
	got, ok := c.get("inst-1")
	if !ok {
		t.Fatalf("cache should hit after set")
	}
	if len(got) != 1 || got[0].Name != "search" {
		t.Fatalf("got %#v", got)
	}
	// A different key still misses.
	if _, ok := c.get("inst-2"); ok {
		t.Fatalf("unrelated key should miss")
	}
}

func TestBuildCompactManifest(t *testing.T) {
	tools := []ProbeTool{
		{Name: "search", Description: "web search"},
		{Name: "fetch", Description: "fetch a URL"},
		{Name: "noop"}, // no description
	}
	manifest := buildCompactManifest(tools)
	if !strings.Contains(manifest, "search") || !strings.Contains(manifest, "web search") {
		t.Fatalf("manifest missing search: %q", manifest)
	}
	if !strings.Contains(manifest, "fetch") {
		t.Fatalf("manifest missing fetch: %q", manifest)
	}
	if !strings.Contains(manifest, "noop") {
		t.Fatalf("manifest missing noop: %q", manifest)
	}
	// One line per tool.
	if got := strings.Count(strings.TrimSpace(manifest), "\n"); got != 2 {
		t.Fatalf("manifest line count = %d (newlines), want 2 for 3 tools", got)
	}
}

func TestBuildCompactManifestEmpty(t *testing.T) {
	if got := buildCompactManifest(nil); got != "" {
		t.Fatalf("empty manifest = %q, want empty", got)
	}
}
