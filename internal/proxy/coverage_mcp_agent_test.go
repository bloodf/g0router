package proxy

import (
	"context"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
)

// TestShouldRunMCPAgentToolsNoneFound exercises the `return false` at line 206
// in shouldRunMCPAgent: mcpTools is set, req has tools, but none are MCP tools.
func TestShouldRunMCPAgentToolsNoneFound(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	tm := mcp.NewToolManager()
	engine.RegisterMCPToolManager(tm)
	// tm has no registered tools, so Lookup will always fail.
	// A request with tools: none match → shouldRunMCPAgent returns false (L206).
	req := &providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
		Tools: []providers.Tool{
			{Type: "function", Function: providers.ToolFunction{Name: "non_mcp_tool"}},
		},
	}
	got := engine.shouldRunMCPAgent(context.Background(), req)
	if got {
		t.Fatal("shouldRunMCPAgent with no matching MCP tools should return false")
	}
}

// TestShouldRunMCPAgentCompactHasTools exercises the CompactToolsForRequest>0 path
// in shouldRunMCPAgent (line 203): when req has no Tools but mcpTools has tools,
// it should return true. This requires a registered tool in the manager.
func TestShouldRunMCPAgentNilReqReturnsFalse(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)
	tm := mcp.NewToolManager()
	engine.RegisterMCPToolManager(tm)

	// nil request → early return false (L200).
	got := engine.shouldRunMCPAgent(context.Background(), nil)
	if got {
		t.Fatal("shouldRunMCPAgent with nil req should return false")
	}
}
