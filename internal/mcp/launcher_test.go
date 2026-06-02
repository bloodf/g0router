package mcp

import (
	"bytes"
	"context"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotProtocol = r.Header.Get("MCP-Protocol-Version")
		w.Header().Set("Mcp-Session-Id", "session-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{}}`))
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
		w.WriteHeader(http.StatusOK)
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
