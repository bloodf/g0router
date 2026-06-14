package mcp

import "testing"

// TestDefaultPluginsCount: the three default plugin definitions (Exa, Tavily,
// browsermcp) are present (coworkPlugins.js:3,26 — PAR-MCP-043/044).
func TestDefaultPluginsCount(t *testing.T) {
	defs := DefaultPlugins()
	if len(defs) != 3 {
		t.Fatalf("DefaultPlugins len = %d, want 3", len(defs))
	}
}

// TestDefaultPluginExa: Exa is HTTP transport with no OAuth (PAR-MCP-043).
func TestDefaultPluginExa(t *testing.T) {
	d := findPlugin(t, "exa")
	if d.Transport != "http" {
		t.Errorf("exa transport = %q, want http", d.Transport)
	}
	if d.OAuth {
		t.Errorf("exa OAuth = true, want false")
	}
}

// TestDefaultPluginTavily: Tavily is HTTP transport with OAuth (PAR-MCP-043).
func TestDefaultPluginTavily(t *testing.T) {
	d := findPlugin(t, "tavily")
	if d.Transport != "http" {
		t.Errorf("tavily transport = %q, want http", d.Transport)
	}
	if !d.OAuth {
		t.Errorf("tavily OAuth = false, want true")
	}
}

// TestDefaultPluginBrowsermcp: browsermcp is a local stdio plugin launched via
// npx with the canonical args + 10 tool names, and its command is allowlisted
// (PAR-MCP-044).
func TestDefaultPluginBrowsermcp(t *testing.T) {
	d := findPlugin(t, "browsermcp")
	if d.Transport != "stdio" {
		t.Errorf("browsermcp transport = %q, want stdio", d.Transport)
	}
	if d.Command != "npx" {
		t.Errorf("browsermcp command = %q, want npx", d.Command)
	}
	if !isAllowedCommand(d.Command) {
		t.Errorf("browsermcp command %q is not allowlisted", d.Command)
	}
	wantArgs := []string{"-y", "@browsermcp/mcp@latest"}
	if len(d.Args) != len(wantArgs) {
		t.Fatalf("browsermcp args = %v, want %v", d.Args, wantArgs)
	}
	for i := range wantArgs {
		if d.Args[i] != wantArgs[i] {
			t.Fatalf("browsermcp args[%d] = %q, want %q", i, d.Args[i], wantArgs[i])
		}
	}
	if len(d.ToolNames) != 10 {
		t.Errorf("browsermcp tool names = %d, want 10", len(d.ToolNames))
	}
}

func findPlugin(t *testing.T, name string) PluginDefinition {
	t.Helper()
	for _, d := range DefaultPlugins() {
		if d.Name == name {
			return d
		}
	}
	t.Fatalf("default plugin %q not found", name)
	return PluginDefinition{}
}
