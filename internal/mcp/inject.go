package mcp

import "context"

// InjectAllowedTools returns a context that restricts MCP tool injection to
// the provided tool names. An empty or nil slice returns ctx unchanged.
func InjectAllowedTools(ctx context.Context, allowed []string) context.Context {
	if len(allowed) == 0 {
		return ctx
	}
	return WithAllowedTools(ctx, allowed...)
}
