package mcp

import (
	"strings"
	"testing"
)

// TestStripServerPrefixIdempotent: repeated "<server>-" prefixes are removed
// down to the bare tool name (PAR-MCP-045/046; mirrors coworkPlugins.js:48
// while-loop).
func TestStripServerPrefixIdempotent(t *testing.T) {
	cases := []struct {
		server, tool, want string
	}{
		{"github", "github-create_issue", "create_issue"},
		{"github", "github-github-create_issue", "create_issue"},
		{"github", "create_issue", "create_issue"},
		{"fs", "read_file", "read_file"},
		{"fs", "fs-fs-fs-read_file", "read_file"},
		{"", "read_file", "read_file"},
	}
	for _, c := range cases {
		got := stripServerPrefix(c.server, c.tool)
		if got != c.want {
			t.Fatalf("stripServerPrefix(%q,%q) = %q, want %q", c.server, c.tool, got, c.want)
		}
	}
}

// TestBuildToolPolicyEmitsBareAndPrefixed: for each tool, both the bare name and
// "<server>-<tool>" are present as "allow" (PAR-MCP-019; cowork-settings:171).
func TestBuildToolPolicyEmitsBareAndPrefixed(t *testing.T) {
	policy := buildToolPolicy("github", []string{"create_issue", "list_repos"})
	want := map[string]string{
		"create_issue":         "allow",
		"github-create_issue":  "allow",
		"list_repos":           "allow",
		"github-list_repos":    "allow",
	}
	if len(policy) != len(want) {
		t.Fatalf("policy = %v, want %v", policy, want)
	}
	for k, v := range want {
		if policy[k] != v {
			t.Fatalf("policy[%q] = %q, want %q", k, policy[k], v)
		}
	}
}

// TestSanitizePluginNameRegexAnd64Cap: non [a-zA-Z0-9_-] chars are stripped and
// the result is truncated to 64 chars (PAR-MCP-048; cowork-settings:339).
func TestSanitizePluginNameRegexAnd64Cap(t *testing.T) {
	if got := sanitizePluginName("my plugin@v1.0!"); got != "mypluginv10" {
		t.Fatalf("sanitize = %q, want %q", got, "mypluginv10")
	}
	if got := sanitizePluginName("keep_this-name99"); got != "keep_this-name99" {
		t.Fatalf("sanitize = %q, want unchanged", got)
	}
	long := strings.Repeat("a", 100)
	if got := sanitizePluginName(long); len(got) != 64 {
		t.Fatalf("len(sanitize(100 a's)) = %d, want 64", len(got))
	}
}

// TestBuildManagedServers: the {name,url,transport,oauth,toolPolicy} list is
// assembled from instance inputs (PAR-MCP-018; cowork-settings:262), with the
// toolPolicy emitting bare+prefixed allow entries.
func TestBuildManagedServers(t *testing.T) {
	servers := buildManagedServers([]managedServerInput{
		{Name: "github", URL: "https://api.github.com/mcp", Transport: "sse", OAuth: true, ToolNames: []string{"create_issue"}},
		{Name: "fs", Transport: "stdio", ToolNames: []string{"read_file"}},
	})
	if len(servers) != 2 {
		t.Fatalf("len(servers) = %d, want 2", len(servers))
	}
	gh := servers[0]
	if gh.Name != "github" || gh.URL != "https://api.github.com/mcp" || gh.Transport != "sse" || !gh.OAuth {
		t.Fatalf("github server = %+v", gh)
	}
	if gh.ToolPolicy["create_issue"] != "allow" || gh.ToolPolicy["github-create_issue"] != "allow" {
		t.Fatalf("github toolPolicy = %v", gh.ToolPolicy)
	}
	if servers[1].OAuth {
		t.Fatalf("fs server OAuth = true, want false")
	}
}
