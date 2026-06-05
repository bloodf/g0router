package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
)

func TestMCPLauncherConnectorReturnsWorkingStreamableHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			// Streamable HTTP session termination on client Close; no body.
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, ok := req["id"]; !ok {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "session-runtime")
		switch req["method"] {
		case "initialize":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": "2025-11-25"})
		case "tools/list":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"tools": []map[string]any{{"name": "search", "description": "Search docs", "inputSchema": map[string]any{"type": "object"}}}})
		case "tools/call":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"content": []map[string]any{{"type": "text", "text": "found"}}})
		default:
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{})
		}
	}))
	defer server.Close()

	connector := mcpLauncherConnector{launcher: mcp.NewLauncher(runtimeProcessRunner{}, server.Client())}
	client, err := connector.Connect(t.Context(), mcp.ClientConfig{
		ID:        "http-1",
		Name:      "http",
		Transport: mcp.TransportStreamableHTTP,
		URL:       server.URL,
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(t.Context())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "search" {
		t.Fatalf("tools = %+v, want search tool", tools)
	}
	if _, err := client.CallTool(t.Context(), mcp.CallRequest{Name: "search", Arguments: json.RawMessage(`{"query":"mcp"}`)}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
}

func writeRuntimeRPCResult(t *testing.T, w http.ResponseWriter, id any, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result}); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
