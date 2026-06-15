package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakeCatalog is a canned CatalogSource for the server-mode dispatcher tests.
type fakeCatalog struct {
	tools []ServerTool
}

func (f fakeCatalog) ListServerTools() []ServerTool { return f.tools }

// fakeDispatcher is a canned ToolDispatcher: it records the last call and
// returns a scripted result/error so tools/call is exercised with no bridge.
type fakeDispatcher struct {
	lastName string
	lastArgs map[string]any
	result   string
	err      error
}

func (f *fakeDispatcher) Execute(_ context.Context, name string, args map[string]any) (string, error) {
	f.lastName = name
	f.lastArgs = args
	return f.result, f.err
}

func decodeRPC(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("dispatch result is not JSON: %v\nraw: %s", err, raw)
	}
	if m["jsonrpc"] != "2.0" {
		t.Fatalf("missing jsonrpc:2.0: %v", m)
	}
	return m
}

func TestServerInitialize(t *testing.T) {
	s := NewServer(fakeCatalog{}, &fakeDispatcher{})
	resp := s.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	m := decodeRPC(t, resp)
	if m["id"] != float64(1) {
		t.Fatalf("id = %v, want 1", m["id"])
	}
	result, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result object: %v", m)
	}
	if result["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("protocolVersion = %v, want %s", result["protocolVersion"], mcpProtocolVersion)
	}
	caps, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("no capabilities: %v", result)
	}
	if _, ok := caps["tools"]; !ok {
		t.Fatalf("capabilities missing tools: %v", caps)
	}
	info, ok := result["serverInfo"].(map[string]any)
	if !ok || info["name"] != "g0router" {
		t.Fatalf("serverInfo = %v, want name g0router", result["serverInfo"])
	}
}

func TestServerToolsList(t *testing.T) {
	cat := fakeCatalog{tools: []ServerTool{
		{Name: "search", Description: "find things", InputSchema: map[string]any{"type": "object"}},
		{Name: "fetch", Description: "get a url"},
	}}
	s := NewServer(cat, &fakeDispatcher{})
	resp := s.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`))
	m := decodeRPC(t, resp)
	result := m["result"].(map[string]any)
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 2 {
		t.Fatalf("tools = %v, want 2", result["tools"])
	}
	first := tools[0].(map[string]any)
	if first["name"] != "search" || first["description"] != "find things" {
		t.Fatalf("first tool = %v", first)
	}
	if _, ok := first["inputSchema"]; !ok {
		t.Fatalf("tool missing inputSchema: %v", first)
	}
}

// TestServerToolsListAnnotations proves the server-mode tools/list carries the
// PAR-BF-MCP-077 annotation fields when the catalog supplies them, and that the
// omitempty shape omits an absent annotation block (additive serialization —
// existing tools without annotations are unchanged).
func TestServerToolsListAnnotations(t *testing.T) {
	cat := fakeCatalog{tools: []ServerTool{
		{Name: "delete_file", Description: "remove a file", Annotations: &ToolAnnotations{
			Title:           "Delete File",
			ReadOnlyHint:    false,
			DestructiveHint: true,
			IdempotentHint:  false,
			OpenWorldHint:   false,
		}},
		{Name: "search", Description: "find things"},
	}}
	s := NewServer(cat, &fakeDispatcher{})
	resp := s.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","id":7,"method":"tools/list"}`))
	m := decodeRPC(t, resp)
	tools := m["result"].(map[string]any)["tools"].([]any)

	first := tools[0].(map[string]any)
	ann, ok := first["annotations"].(map[string]any)
	if !ok {
		t.Fatalf("annotated tool missing annotations: %v", first)
	}
	if ann["title"] != "Delete File" {
		t.Fatalf("annotation title = %v, want Delete File", ann["title"])
	}
	if ann["destructiveHint"] != true {
		t.Fatalf("annotation destructiveHint = %v, want true", ann["destructiveHint"])
	}

	// The un-annotated tool omits the annotations block (omitempty additive shape).
	second := tools[1].(map[string]any)
	if _, present := second["annotations"]; present {
		t.Fatalf("un-annotated tool should omit annotations: %v", second)
	}
}

func TestServerToolsCall(t *testing.T) {
	disp := &fakeDispatcher{result: "generic\nresult text\nresult text"}
	s := NewServer(fakeCatalog{}, disp)
	resp := s.Dispatch(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"q":"hi"}}}`))
	m := decodeRPC(t, resp)
	if disp.lastName != "search" {
		t.Fatalf("dispatcher name = %q, want search", disp.lastName)
	}
	if disp.lastArgs["q"] != "hi" {
		t.Fatalf("dispatcher args = %v", disp.lastArgs)
	}
	result := m["result"].(map[string]any)
	content, ok := result["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content = %v, want 1", result["content"])
	}
	block := content[0].(map[string]any)
	if block["type"] != "text" {
		t.Fatalf("block type = %v, want text", block["type"])
	}
	// smartFilterText must have dropped the bare "generic" line and collapsed
	// the duplicate sibling.
	text, _ := block["text"].(string)
	if strings.Contains(text, "generic") {
		t.Fatalf("smartFilterText not applied: %q", text)
	}
	if strings.Count(text, "result text") != 1 {
		t.Fatalf("duplicate sibling not collapsed: %q", text)
	}
}

func TestServerToolsCallError(t *testing.T) {
	disp := &fakeDispatcher{err: errors.New("boom")}
	s := NewServer(fakeCatalog{}, disp)
	resp := s.Dispatch(context.Background(),
		[]byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"x"}}`))
	m := decodeRPC(t, resp)
	if _, ok := m["error"]; !ok {
		t.Fatalf("tools/call error not surfaced as JSON-RPC error: %v", m)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	s := NewServer(fakeCatalog{}, &fakeDispatcher{})
	resp := s.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","id":5,"method":"does/not/exist"}`))
	m := decodeRPC(t, resp)
	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error object: %v", m)
	}
	if errObj["code"] != float64(-32601) {
		t.Fatalf("error code = %v, want -32601 (method not found)", errObj["code"])
	}
}

func TestServerMalformedJSON(t *testing.T) {
	s := NewServer(fakeCatalog{}, &fakeDispatcher{})
	resp := s.Dispatch(context.Background(), []byte(`{not json`))
	m := decodeRPC(t, resp)
	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error object: %v", m)
	}
	if errObj["code"] != float64(-32700) {
		t.Fatalf("error code = %v, want -32700 (parse error)", errObj["code"])
	}
}
