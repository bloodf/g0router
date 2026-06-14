package mcp

// PluginDefinition is a default MCP plugin definition ported from 9router's
// coworkPlugins.js. These are consumed by w7-mcp-2 (registry/probe) and
// w7-mcp-3 (seed/UI); this package only declares them as Go values (no live
// HTTP dial here).
type PluginDefinition struct {
	Name      string
	Transport string // 'http' | 'sse' | 'stdio'
	URL       string // http/sse modes
	Command   string // stdio mode (allowlist-validated)
	Args      []string
	OAuth     bool
	ToolNames []string
}

// DefaultPlugins returns the default plugin definitions: Exa (HTTP, no auth),
// Tavily (HTTP, OAuth), and browsermcp (local stdio via npx). Mirrors
// coworkPlugins.js:3,26 (PAR-MCP-043/044).
func DefaultPlugins() []PluginDefinition {
	return []PluginDefinition{
		{
			Name:      "exa",
			Transport: "http",
			URL:       "https://mcp.exa.ai/mcp",
			OAuth:     false,
		},
		{
			Name:      "tavily",
			Transport: "http",
			URL:       "https://mcp.tavily.com/mcp",
			OAuth:     true,
		},
		{
			Name:      "browsermcp",
			Transport: "stdio",
			Command:   "npx",
			Args:      []string{"-y", "@browsermcp/mcp@latest"},
			OAuth:     false,
			ToolNames: []string{
				"browser_navigate",
				"browser_go_back",
				"browser_go_forward",
				"browser_snapshot",
				"browser_click",
				"browser_hover",
				"browser_type",
				"browser_select_option",
				"browser_press_key",
				"browser_get_console_logs",
			},
		},
	}
}
