package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLauncherBuildsCommandSpecWithoutShell(t *testing.T) {
	spec, err := BuildLaunchSpec(InstanceConfig{
		LaunchType: LaunchCommand,
		Transport:  TransportStdio,
		Command:    "mcp-files",
		Args:       []string{"--root", "/tmp/docs; rm -rf /"},
	})
	if err != nil {
		t.Fatalf("BuildLaunchSpec: %v", err)
	}

	if spec.Command != "mcp-files" {
		t.Fatalf("command = %q, want mcp-files", spec.Command)
	}
	if len(spec.Args) != 2 || spec.Args[1] != "/tmp/docs; rm -rf /" {
		t.Fatalf("args = %#v, want argv preservation", spec.Args)
	}
}

func TestLauncherBuildsNPXSpecWithoutShellInterpolation(t *testing.T) {
	spec, err := BuildLaunchSpec(InstanceConfig{
		LaunchType: LaunchNPX,
		Transport:  TransportStdio,
		Command:    "@modelcontextprotocol/server-filesystem",
		Args:       []string{"--root", "/tmp/docs"},
	})
	if err != nil {
		t.Fatalf("BuildLaunchSpec: %v", err)
	}

	want := []string{"--yes", "@modelcontextprotocol/server-filesystem", "--root", "/tmp/docs"}
	if spec.Command != "npx" {
		t.Fatalf("command = %q, want npx", spec.Command)
	}
	if !stringSlicesEqual(spec.Args, want) {
		t.Fatalf("args = %#v, want %#v", spec.Args, want)
	}
}

func TestLauncherBuildsDockerSpec(t *testing.T) {
	spec, err := BuildLaunchSpec(InstanceConfig{
		LaunchType: LaunchDocker,
		Transport:  TransportStdio,
		Command:    "mcp/server:latest",
		Args:       []string{"--debug"},
		Env:        map[string]string{"TOKEN": "secret"},
	})
	if err != nil {
		t.Fatalf("BuildLaunchSpec: %v", err)
	}

	want := []string{"run", "--rm", "-i", "-e", "TOKEN", "mcp/server:latest", "--debug"}
	if spec.Command != "docker" {
		t.Fatalf("command = %q, want docker", spec.Command)
	}
	if !stringSlicesEqual(spec.Args, want) {
		t.Fatalf("args = %#v, want %#v", spec.Args, want)
	}
}

func TestLauncherCapturesStderrDiagnostics(t *testing.T) {
	runner := &fakeProcessRunner{stderr: "warming cache\nready\n"}
	launcher := NewLauncher(runner, nil)

	result, err := launcher.Launch(context.Background(), InstanceConfig{
		LaunchType: LaunchCommand,
		Transport:  TransportStdio,
		Command:    "mcp-files",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if result.Diagnostics != "warming cache\nready\n" {
		t.Fatalf("diagnostics = %q, want stderr", result.Diagnostics)
	}
}

func TestHTTPLauncherStoresStreamableSessionID(t *testing.T) {
	var gotProtocol string
	var initializeRequest map[string]any
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		method, _ := req["method"].(string)
		methods = append(methods, method)
		if method == "notifications/initialized" {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		gotProtocol = r.Header.Get("MCP-Protocol-Version")
		initializeRequest = req
		w.Header().Set("Mcp-Session-Id", "session-123")
		writeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": protocolVersion})
	}))
	defer server.Close()

	launcher := NewLauncher(nil, server.Client())
	result, err := launcher.Launch(context.Background(), InstanceConfig{
		LaunchType: LaunchHTTP,
		Transport:  TransportStreamableHTTP,
		URL:        server.URL,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	if result.SessionID != "session-123" {
		t.Fatalf("session = %q, want session-123", result.SessionID)
	}
	if gotProtocol == "" {
		t.Fatal("MCP protocol version header should be sent")
	}
	if initializeRequest["method"] != "initialize" {
		t.Fatalf("method = %q, want initialize", initializeRequest["method"])
	}
	params, ok := initializeRequest["params"].(map[string]any)
	if !ok {
		t.Fatalf("params = %#v, want object", initializeRequest["params"])
	}
	if params["protocolVersion"] != protocolVersion {
		t.Fatalf("params.protocolVersion = %q, want %q", params["protocolVersion"], protocolVersion)
	}
	if _, ok := params["capabilities"].(map[string]any); !ok {
		t.Fatalf("params.capabilities = %#v, want object", params["capabilities"])
	}
	clientInfo, ok := params["clientInfo"].(map[string]any)
	if !ok {
		t.Fatalf("params.clientInfo = %#v, want object", params["clientInfo"])
	}
	if clientInfo["name"] != "g0router" {
		t.Fatalf("params.clientInfo.name = %q, want g0router", clientInfo["name"])
	}
	wantMethods := []string{"initialize", "notifications/initialized"}
	if !stringSlicesEqual(methods, wantMethods) {
		t.Fatalf("methods = %#v, want %#v", methods, wantMethods)
	}
}

func TestHTTPLauncherFallsBackToSSEOnlyForDocumentedStatuses(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != "/sse" {
			t.Fatalf("fallback path = %q, want /sse", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("event: endpoint\ndata: /message\n\n"))
	}))
	defer server.Close()

	launcher := NewLauncher(nil, server.Client())
	result, err := launcher.Launch(context.Background(), InstanceConfig{
		LaunchType: LaunchHTTP,
		Transport:  TransportStreamableHTTP,
		URL:        server.URL,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.Transport != TransportSSE {
		t.Fatalf("transport = %q, want sse fallback", result.Transport)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want streamable + sse", requests)
	}
}

func TestHTTPLauncherDoesNotFallbackForServerErrors(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	launcher := NewLauncher(nil, server.Client())
	_, err := launcher.Launch(context.Background(), InstanceConfig{
		LaunchType: LaunchHTTP,
		Transport:  TransportStreamableHTTP,
		URL:        server.URL,
	})
	if err == nil {
		t.Fatal("Launch error is nil")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want no fallback", requests)
	}
}

type fakeProcessRunner struct {
	stderr string
}

func (r *fakeProcessRunner) Start(ctx context.Context, spec ProcessSpec) (Process, error) {
	return fakeProcess{stderr: r.stderr}, nil
}

type fakeProcess struct {
	stderr string
}

func (p fakeProcess) Stdin() io.WriteCloser {
	return nopWriteCloser{Writer: &bytes.Buffer{}}
}

func (p fakeProcess) Stdout() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(nil))
}

func (p fakeProcess) Stderr() *bytes.Buffer {
	return bytes.NewBufferString(p.stderr)
}

func (p fakeProcess) Close() error {
	return nil
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !strings.EqualFold(a[i], b[i]) {
			return false
		}
	}
	return true
}

type nopWriteCloser struct {
	io.Writer
}

func (w nopWriteCloser) Close() error {
	return nil
}
