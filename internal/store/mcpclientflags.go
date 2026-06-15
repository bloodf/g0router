package store

import "strings"

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
	mcpClientExtraHeadersKey      = "allowed_extra_headers"
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

// MCPClientAllowedExtraHeaders reads the canonicalized allowed_extra_headers
// whitelist from the client config blob (PAR-BF-MCP-071). The stored value is a
// JSON array, so it round-trips as []any after unmarshal; this normalizes back to
// []string and re-canonicalizes defensively.
func MCPClientAllowedExtraHeaders(c *MCPClient) []string {
	if c == nil || c.Config == nil {
		return nil
	}
	raw, ok := c.Config[mcpClientExtraHeadersKey]
	if !ok {
		return nil
	}
	var headers []string
	switch v := raw.(type) {
	case []string:
		headers = v
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				headers = append(headers, s)
			}
		}
	default:
		return nil
	}
	return CanonicalizeExtraHeaders(headers)
}

// normalizeMCPClientConfig canonicalizes the operator-set config-blob values that
// have a validated shape before persistence (PAR-BF-MCP-071): the
// allowed_extra_headers whitelist is canonicalized in place so a stored client
// always round-trips a clean whitelist. It is called from the live client write
// path (Create/Upsert) — the operator-set consumer of CanonicalizeExtraHeaders.
func normalizeMCPClientConfig(c *MCPClient) {
	if c == nil || c.Config == nil {
		return
	}
	if hdrs := MCPClientAllowedExtraHeaders(c); hdrs != nil {
		c.Config[mcpClientExtraHeadersKey] = hdrs
	}
}

// CanonicalizeExtraHeaders canonicalizes an AllowedExtraHeaders whitelist
// (PAR-BF-MCP-071, D8 config-only): lowercase, trim, drop empties, de-duplicate,
// preserving first-seen order. There is NO server-mode upstream header-forwarding
// path to gate (forwarding ESC); this is the stored + validated config surface
// only. PURE.
func CanonicalizeExtraHeaders(headers []string) []string {
	out := make([]string, 0, len(headers))
	seen := map[string]bool{}
	for _, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" || seen[h] {
			continue
		}
		seen[h] = true
		out = append(out, h)
	}
	return out
}
