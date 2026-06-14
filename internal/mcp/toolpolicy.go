package mcp

import (
	"regexp"
	"strings"
)

// pluginNameRe matches the characters NOT permitted in a sanitized plugin name.
var pluginNameRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// maxPluginNameLen caps a sanitized plugin name (cowork-settings:339).
const maxPluginNameLen = 64

// managedServerInput is the per-instance input to buildManagedServers.
type managedServerInput struct {
	Name      string
	URL       string
	Transport string
	OAuth     bool
	ToolNames []string
}

// ManagedServer is the assembled {name,url,transport,oauth,toolPolicy} record
// (PAR-MCP-018; cowork-settings:262). PURE data — no I/O.
type ManagedServer struct {
	Name       string
	URL        string
	Transport  string
	OAuth      bool
	ToolPolicy map[string]string
}

// stripServerPrefix idempotently removes repeated "<server>-" prefixes from a
// tool name (PAR-MCP-045/046; mirrors coworkPlugins.js:48 while-loop). PURE.
func stripServerPrefix(server, tool string) string {
	if server == "" {
		return tool
	}
	prefix := server + "-"
	for strings.HasPrefix(tool, prefix) {
		tool = tool[len(prefix):]
	}
	return tool
}

// buildToolPolicy emits, for each tool, BOTH the bare name and "<server>-<tool>"
// as "allow" (PAR-MCP-019; cowork-settings:171). PURE.
func buildToolPolicy(server string, toolNames []string) map[string]string {
	policy := make(map[string]string, len(toolNames)*2)
	for _, t := range toolNames {
		bare := stripServerPrefix(server, t)
		policy[bare] = "allow"
		if server != "" {
			policy[server+"-"+bare] = "allow"
		}
	}
	return policy
}

// buildManagedServers assembles the managed-server list from instance inputs
// (PAR-MCP-018; cowork-settings:262). PURE over its inputs.
func buildManagedServers(inputs []managedServerInput) []ManagedServer {
	out := make([]ManagedServer, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, ManagedServer{
			Name:       in.Name,
			URL:        in.URL,
			Transport:  in.Transport,
			OAuth:      in.OAuth,
			ToolPolicy: buildToolPolicy(in.Name, in.ToolNames),
		})
	}
	return out
}

// sanitizePluginName strips characters outside [a-zA-Z0-9_-] and truncates the
// result to maxPluginNameLen (PAR-MCP-048; cowork-settings:339). PURE.
func sanitizePluginName(s string) string {
	s = pluginNameRe.ReplaceAllString(s, "")
	if len(s) > maxPluginNameLen {
		s = s[:maxPluginNameLen]
	}
	return s
}
