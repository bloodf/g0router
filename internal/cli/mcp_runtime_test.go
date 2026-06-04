package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
)

func TestMCPLauncherConnectorReturnsWorkingStdioClient(t *testing.T) {
	server, process := newRuntimeMCPServer(t)
	defer server.Close()

	connector := mcpLauncherConnector{
		launcher: mcp.NewLauncher(runtimeProcessRunner{process: process}, nil),
	}
	client, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "stdio-1",
		Name:      "stdio",
		Transport: mcp.TransportStdio,
		Command:   "fake-mcp",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "search" {
		t.Fatalf("tools = %+v, want search tool", tools)
	}

	result, err := client.CallTool(context.Background(), mcp.CallRequest{Name: "search", Arguments: json.RawMessage(`{"query":"mcp"}`)})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.Content == nil {
		t.Fatal("call result content is nil")
	}
	if server.CallCount() != 1 {
		t.Fatalf("call count = %d, want 1", server.CallCount())
	}
}

func TestMCPLauncherConnectorRejectsUnsupportedLaunchTransport(t *testing.T) {
	process := &closableRuntimeProcess{stderr: &bytes.Buffer{}}
	connector := mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{
			result: mcp.LaunchResult{
				Transport: "bogus",
				Process:   process,
			},
		},
	}

	_, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "bogus-1",
		Name:      "bogus",
		Transport: mcp.TransportStdio,
		Command:   "fake-mcp",
	})
	if err == nil {
		t.Fatal("Connect error is nil")
	}
	if !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("Connect error = %v, want ErrInvalidClientConfig", err)
	}
	if !process.closed {
		t.Fatal("unsupported launch transport did not close process")
	}
}

type runtimeProcessRunner struct {
	process mcp.Process
}

func (r runtimeProcessRunner) Start(ctx context.Context, spec mcp.ProcessSpec) (mcp.Process, error) {
	return r.process, nil
}

type runtimeMCPServer struct {
	t      *testing.T
	input  io.WriteCloser
	output io.ReadCloser
	calls  int
	mu     sync.Mutex
	done   chan struct{}
}

type runtimeMCPProcess struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr *bytes.Buffer
}

type closableRuntimeProcess struct {
	closed bool
	stderr *bytes.Buffer
}

func (p *closableRuntimeProcess) Stdin() io.WriteCloser {
	return nopWriteCloser{Writer: io.Discard}
}

func (p *closableRuntimeProcess) Stdout() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(nil))
}

func (p *closableRuntimeProcess) Stderr() *bytes.Buffer {
	return p.stderr
}

func (p *closableRuntimeProcess) Close() error {
	p.closed = true
	return nil
}

type fakeRuntimeLauncher struct {
	result mcp.LaunchResult
	err    error
}

func (l fakeRuntimeLauncher) Launch(context.Context, mcp.InstanceConfig) (mcp.LaunchResult, error) {
	return l.result, l.err
}

func (l fakeRuntimeLauncher) HTTPClient() mcp.HTTPDoer {
	return nil
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func newRuntimeMCPServer(t *testing.T) (*runtimeMCPServer, *runtimeMCPProcess) {
	t.Helper()
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	server := &runtimeMCPServer{t: t, input: serverWrite, output: serverRead, done: make(chan struct{})}
	process := &runtimeMCPProcess{stdin: clientWrite, stdout: clientRead, stderr: &bytes.Buffer{}}
	go server.run()
	return server, process
}

func (s *runtimeMCPServer) run() {
	defer close(s.done)
	scanner := bufio.NewScanner(s.output)
	for scanner.Scan() {
		var req map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			s.t.Errorf("decode client request: %v", err)
			return
		}
		method, _ := req["method"].(string)
		if _, ok := req["id"]; !ok {
			continue
		}
		var result any
		switch method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2025-11-25"}
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{"name": "search", "description": "Search docs", "inputSchema": map[string]any{"type": "object"}}}}
		case "tools/call":
			s.mu.Lock()
			s.calls++
			s.mu.Unlock()
			result = map[string]any{"content": []map[string]any{{"type": "text", "text": "found"}}}
		default:
			result = map[string]any{}
		}
		encoded, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": result})
		if err != nil {
			s.t.Errorf("encode response: %v", err)
			return
		}
		if _, err := s.input.Write(append(encoded, '\n')); err != nil {
			return
		}
	}
}

func (s *runtimeMCPServer) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func (s *runtimeMCPServer) Close() {
	_ = s.input.Close()
	_ = s.output.Close()
	<-s.done
}

func (p *runtimeMCPProcess) Stdin() io.WriteCloser {
	return p.stdin
}

func (p *runtimeMCPProcess) Stdout() io.ReadCloser {
	return p.stdout
}

func (p *runtimeMCPProcess) Stderr() *bytes.Buffer {
	return p.stderr
}

func (p *runtimeMCPProcess) Close() error {
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	return nil
}
