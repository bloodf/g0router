package store

// Additive per-MCP-client config-blob flags (bf-mcp-2). They ride the existing
// mcp_clients.config_json map[string]any blob — additive JSON keys, never a
// struct-shape break — so an old blob without the keys still unmarshals and the
// flags default false. The accessors are the operator-set + live-read surface for
// PAR-BF-MCP-020 (allow-on-all-VK bypass), 057 (disable-auto-inject), and 078
// (code-mode client; execution ESC).
const (
	mcpClientAllowOnAllVKsKey     = "allow_on_all_virtual_keys"
	mcpClientDisableAutoInjectKey = "disable_auto_tool_inject"
	mcpClientCodeModeKey          = "is_code_mode_client"
)

// configBool reads a boolean config-blob flag, defaulting false when the key is
// absent or not a bool (defensive against a heterogeneous JSON decode).
func configBool(c *MCPClient, key string) bool {
	if c == nil || c.Config == nil {
		return false
	}
	v, ok := c.Config[key].(bool)
	return ok && v
}

// MCPClientAllowOnAllVKs reports whether the client's tools bypass per-VK
// executeOnlyTools scoping and are visible to every VK (PAR-BF-MCP-020, D6).
func MCPClientAllowOnAllVKs(c *MCPClient) bool { return configBool(c, mcpClientAllowOnAllVKsKey) }

// MCPClientDisableAutoToolInject reports whether the client's tools are omitted
// from the auto-injected server-mode surface (PAR-BF-MCP-057, D7).
func MCPClientDisableAutoToolInject(c *MCPClient) bool {
	return configBool(c, mcpClientDisableAutoInjectKey)
}

// MCPClientIsCodeMode reports whether the client is a code-mode client
// (PAR-BF-MCP-078, D8 — flag stored only; code-mode VFS execution is ESC).
func MCPClientIsCodeMode(c *MCPClient) bool { return configBool(c, mcpClientCodeModeKey) }
