package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestStdioClientInitializesAndListsTools(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		switch req["method"] {
		case "initialize":
			return rpcResult(req["id"], map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]any{},
				"serverInfo":      map[string]any{"name": "fake", "version": "1.0"},
			})
		case "tools/list":
			return rpcResult(req["id"], map[string]any{
				"tools": []map[string]any{
					{
						"name":        "search",
						"description": "Search docs",
						"inputSchema": map[string]any{
							"type":       "object",
							"properties": map[string]any{"query": map[string]any{"type": "string"}},
						},
					},
				},
			})
		default:
			return rpcError(req["id"], -32601, "method not found")
		}
	})
	defer server.Close()

	client := NewStdioClient(process)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	if tools[0].Name != "search" || tools[0].Description != "Search docs" {
		t.Fatalf("tool = %+v", tools[0])
	}
	if !bytes.Contains(tools[0].InputSchema, []byte(`"query"`)) {
		t.Fatalf("input schema = %s, want full schema", tools[0].InputSchema)
	}

	methods := server.Methods()
	want := []string{"initialize", "notifications/initialized", "tools/list"}
	if !stringSlicesEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestStdioClientCallsTool(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		switch req["method"] {
		case "initialize":
			return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/call":
			params := req["params"].(map[string]any)
			if params["name"] != "search" {
				t.Fatalf("tool name = %q, want search", params["name"])
			}
			arguments := params["arguments"].(map[string]any)
			if arguments["query"] != "mcp" {
				t.Fatalf("arguments = %+v, want query=mcp", arguments)
			}
			return rpcResult(req["id"], map[string]any{
				"content": []map[string]any{{"type": "text", "text": "found"}},
			})
		default:
			return rpcError(req["id"], -32601, "method not found")
		}
	})
	defer server.Close()

	client := NewStdioClient(process)
	result, err := client.CallTool(context.Background(), CallRequest{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"mcp"}`),
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	content, ok := result.Content.([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content = %#v, want one content item", result.Content)
	}
}

func TestStdioClientReturnsJSONRPCError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		if req["method"] == "initialize" {
			return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
		}
		return rpcError(req["id"], -32000, "tool failed")
	})
	defer server.Close()

	client := NewStdioClient(process)
	_, err := client.CallTool(context.Background(), CallRequest{Name: "search", Arguments: json.RawMessage(`{}`)})
	if err == nil || !strings.Contains(err.Error(), "tool failed") {
		t.Fatalf("CallTool error = %v, want json-rpc error", err)
	}
}

type fakeStdioServer struct {
	t       *testing.T
	input   io.WriteCloser
	output  io.ReadCloser
	methods []string
	mu      sync.Mutex
	done    chan struct{}
}

type fakeStdioProcess struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *bytes.Buffer
}

func newFakeStdioServer(t *testing.T, handler func(map[string]any) map[string]any) (*fakeStdioServer, *fakeStdioProcess) {
	t.Helper()
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	server := &fakeStdioServer{t: t, input: serverWrite, output: serverRead, done: make(chan struct{})}
	process := &fakeStdioProcess{stdin: clientWrite, stdout: clientRead, stderr: &bytes.Buffer{}}
	go server.run(handler)
	return server, process
}

func (s *fakeStdioServer) run(handler func(map[string]any) map[string]any) {
	defer close(s.done)
	scanner := bufio.NewScanner(s.output)
	for scanner.Scan() {
		var req map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			s.t.Errorf("decode client request: %v", err)
			return
		}
		method, _ := req["method"].(string)
		s.mu.Lock()
		s.methods = append(s.methods, method)
		s.mu.Unlock()
		if _, hasID := req["id"]; !hasID {
			continue
		}
		resp := handler(req)
		encoded, err := json.Marshal(resp)
		if err != nil {
			s.t.Errorf("encode response: %v", err)
			return
		}
		if _, err := s.input.Write(append(encoded, '\n')); err != nil {
			return
		}
	}
}

func (s *fakeStdioServer) Methods() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.methods...)
}

func (s *fakeStdioServer) Close() {
	_ = s.input.Close()
	_ = s.output.Close()
	<-s.done
}

func (p *fakeStdioProcess) Stdin() io.WriteCloser {
	return p.stdin
}

func (p *fakeStdioProcess) Stdout() io.ReadCloser {
	return p.stdout
}

func (p *fakeStdioProcess) Stderr() *bytes.Buffer {
	return p.stderr
}

func (p *fakeStdioProcess) Close() error {
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	return nil
}

func rpcResult(id any, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func rpcError(id any, code int, message string) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": code, "message": message}}
}
