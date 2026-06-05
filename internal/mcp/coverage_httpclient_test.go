package mcp

import (
	"encoding/json"
	"testing"
)

func TestDecodeJSONRPCResult(t *testing.T) {
	// Invalid JSON body.
	if err := decodeJSONRPCResult("test", []byte("notjson"), 1, nil); err == nil {
		t.Fatal("invalid body: want error")
	}
	// Mismatched id.
	body, _ := json.Marshal(jsonrpcResponse{JSONRPC: "2.0", ID: 99})
	if err := decodeJSONRPCResult("test", body, 1, nil); err == nil {
		t.Fatal("mismatched id: want error")
	}
	// JSON-RPC error.
	errBody, _ := json.Marshal(jsonrpcResponse{JSONRPC: "2.0", ID: 1, Error: &jsonrpcError{Code: -1, Message: "boom"}})
	if err := decodeJSONRPCResult("test", errBody, 1, nil); err == nil {
		t.Fatal("jsonrpc error: want error")
	}
	// Nil result target -> ok.
	okBody, _ := json.Marshal(jsonrpcResponse{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`{"x":1}`)})
	if err := decodeJSONRPCResult("test", okBody, 1, nil); err != nil {
		t.Fatalf("nil target: %v", err)
	}
	// Result decode into target.
	var out struct {
		X int `json:"x"`
	}
	if err := decodeJSONRPCResult("test", okBody, 1, &out); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if out.X != 1 {
		t.Fatalf("out.X = %d, want 1", out.X)
	}
	// Result decode error (type mismatch).
	badResult, _ := json.Marshal(jsonrpcResponse{JSONRPC: "2.0", ID: 1, Result: json.RawMessage(`"string"`)})
	if err := decodeJSONRPCResult("test", badResult, 1, &out); err == nil {
		t.Fatal("result type mismatch: want error")
	}
	// Empty result with non-nil target -> ok (no-op).
	emptyResult, _ := json.Marshal(jsonrpcResponse{JSONRPC: "2.0", ID: 1})
	if err := decodeJSONRPCResult("test", emptyResult, 1, &out); err != nil {
		t.Fatalf("empty result: %v", err)
	}
}

func TestToolsListResultPrefersSnakeCaseSchema(t *testing.T) {
	r := toolsListResult{}
	r.ToolsData = append(r.ToolsData, struct {
		Name             string          `json:"name"`
		Description      string          `json:"description"`
		InputSchema      json.RawMessage `json:"inputSchema"`
		InputSchemaSnake json.RawMessage `json:"input_schema"`
	}{Name: "t", InputSchemaSnake: json.RawMessage(`{"type":"object"}`)})
	tools := r.Tools()
	if len(tools) != 1 || string(tools[0].InputSchema) != `{"type":"object"}` {
		t.Fatalf("tools = %+v", tools)
	}
}
