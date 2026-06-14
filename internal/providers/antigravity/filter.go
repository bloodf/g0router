package antigravity

// agToolSuffix is appended to client tool names before they are sent to
// Antigravity (executors/antigravity.js AG_TOOL_SUFFIX = "_ide").
const agToolSuffix = "_ide"

// unavailableToolMarker is the description Antigravity uses for the decoy tools
// it injects so the upstream treats them as hardcoded-unavailable
// (PAR-MCP-060 ride-along; executors/antigravity.js AG_DECOY_TOOLS).
const unavailableToolMarker = "This tool is currently unavailable."

// agDefaultTools is the set of native Antigravity tool names. Client tools whose
// name collides with a native name are preserved WITHOUT the _ide suffix
// (executors/antigravity.js AG_DEFAULT_TOOLS, appConstants.js:104-125).
var agDefaultTools = map[string]bool{
	"browser_subagent":           true,
	"command_status":             true,
	"find_by_name":               true,
	"generate_image":             true,
	"grep_search":                true,
	"list_dir":                   true,
	"list_resources":             true,
	"multi_replace_file_content": true,
	"notify_user":                true,
	"read_resource":              true,
	"read_terminal":              true,
	"read_url_content":           true,
	"replace_file_content":       true,
	"run_command":                true,
	"search_web":                 true,
	"send_command_input":         true,
	"task_boundary":              true,
	"view_content_chunk":         true,
	"view_file":                  true,
	"write_to_file":              true,
}

// agDecoyToolNames is the ordered list of decoy tool names injected by
// Antigravity, each marked unavailable (executors/antigravity.js AG_DECOY_TOOLS:
// the AG default tool names plus mcp_sequential-thinking_sequentialthinking).
var agDecoyToolNames = []string{
	"browser_subagent",
	"command_status",
	"find_by_name",
	"generate_image",
	"grep_search",
	"list_dir",
	"list_resources",
	"mcp_sequential-thinking_sequentialthinking",
	"multi_replace_file_content",
	"notify_user",
	"read_resource",
	"read_terminal",
	"read_url_content",
	"replace_file_content",
	"run_command",
	"search_web",
	"send_command_input",
	"task_boundary",
	"view_content_chunk",
	"view_file",
	"write_to_file",
}

// cloakTools implements the PAR-MCP-060 unavailable-tool ride-along
// (executors/antigravity.js cloakTools): client tools are renamed with the _ide
// suffix (native AG names preserved), then the AG decoy tools — each described as
// unavailable — are appended. It returns the cloaked tool list and a map from
// suffixed name -> original name so the response path can restore the real names.
// An empty input yields (nil, nil).
func cloakTools(tools []map[string]any) ([]map[string]any, map[string]string) {
	if len(tools) == 0 {
		return nil, nil
	}

	nameMap := make(map[string]string)
	var cloaked []map[string]any
	seen := make(map[string]bool)

	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == "" {
			continue
		}
		if agDefaultTools[name] {
			// Native AG name — preserve unchanged.
			if !seen[name] {
				seen[name] = true
				cloaked = append(cloaked, tool)
			}
			continue
		}
		suffixed := name + agToolSuffix
		nameMap[suffixed] = name
		if seen[suffixed] {
			continue
		}
		seen[suffixed] = true
		renamed := make(map[string]any, len(tool))
		for k, v := range tool {
			renamed[k] = v
		}
		renamed["name"] = suffixed
		cloaked = append(cloaked, renamed)
	}

	// Append the decoy tools, each marked unavailable.
	for _, name := range agDecoyToolNames {
		if seen[name] {
			continue
		}
		seen[name] = true
		cloaked = append(cloaked, map[string]any{
			"name":        name,
			"description": unavailableToolMarker,
		})
	}

	return cloaked, nameMap
}
