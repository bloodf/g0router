package schemas

// MCPClient defines an MCP client registration.
type MCPClient struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

// MCPInstance is a running instance of an MCP client.
type MCPInstance struct {
	ID        string            `json:"id"`
	ClientID  string            `json:"client_id"`
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	URL       string            `json:"url,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Status    string            `json:"status"`
}

// MCPTool is a tool exposed by an MCP server.
type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// MCPToolGroup groups tools for assignment.
type MCPToolGroup struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	ToolNames []string `json:"tool_names"`
}
