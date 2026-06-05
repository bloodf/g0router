package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

func TestRegisterInstanceNilGuard(t *testing.T) {
	var r *defaultMCPRuntime
	if _, err := r.RegisterInstance(context.Background(), &store.MCPInstance{}); !errors.Is(err, mcp.ErrInvalidDiscovery) {
		t.Fatalf("nil runtime err = %v", err)
	}

	partial := &defaultMCPRuntime{}
	if _, err := partial.RegisterInstance(context.Background(), &store.MCPInstance{}); !errors.Is(err, mcp.ErrInvalidDiscovery) {
		t.Fatalf("partial runtime err = %v", err)
	}
}

func TestCloseInstanceNilGuard(t *testing.T) {
	var r *defaultMCPRuntime
	if err := r.CloseInstance("x"); !errors.Is(err, mcp.ErrInvalidDiscovery) {
		t.Fatalf("nil runtime close err = %v", err)
	}
}

func TestReapplyInstanceCredentialsMissingInstance(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer s.Close()
	r := newDefaultMCPRuntime()
	if _, err := r.ReapplyInstanceCredentials(context.Background(), s, "missing-id"); err == nil {
		t.Fatal("expected error for missing instance")
	}
}

func TestConnectLaunchError(t *testing.T) {
	connector := mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{err: errors.New("launch failed")},
	}
	if _, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "x",
		Name:      "x",
		Transport: mcp.TransportStdio,
		Command:   "fake",
	}); err == nil {
		t.Fatal("expected launch error")
	}
}

func TestConnectStreamableHTTP(t *testing.T) {
	connector := mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{result: mcp.LaunchResult{Transport: mcp.TransportStreamableHTTP, SessionID: "s1"}},
	}
	client, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "h1",
		Name:      "http",
		Transport: mcp.TransportStreamableHTTP,
		URL:       "https://mcp.example/mcp",
	})
	if err != nil || client == nil {
		t.Fatalf("streamable http connect: client=%v err=%v", client, err)
	}
}

func TestConnectSSE(t *testing.T) {
	connector := mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{result: mcp.LaunchResult{Transport: mcp.TransportSSE}},
	}
	client, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "s1",
		Name:      "sse",
		Transport: mcp.TransportSSE,
		URL:       "https://mcp.example/sse",
	})
	if err != nil || client == nil {
		t.Fatalf("sse connect: client=%v err=%v", client, err)
	}
}

func TestConnectInvalidConfig(t *testing.T) {
	connector := mcpLauncherConnector{launcher: fakeRuntimeLauncher{}}
	// stdio transport without command -> clientInstanceConfig rejects.
	if _, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "bad",
		Transport: mcp.TransportStdio,
	}); !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("err = %v, want ErrInvalidClientConfig", err)
	}
}

func TestCommandProcessCloseNil(t *testing.T) {
	p := &commandProcess{stderr: &bytes.Buffer{}}
	if err := p.Close(); err != nil {
		t.Fatalf("Close nil cmd: %v", err)
	}
}

func TestClientInstanceConfigHTTPMissingURL(t *testing.T) {
	if _, err := clientInstanceConfig(mcp.ClientConfig{Transport: mcp.TransportStreamableHTTP}); !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("err = %v", err)
	}
}

// newStdioRuntime wires a defaultMCPRuntime whose launcher returns the given
// in-process stdio process, so RegisterInstance / CloseInstance success paths
// run without spawning a real subprocess.
func newStdioRuntime(process mcp.Process) *defaultMCPRuntime {
	connector := &mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{result: mcp.LaunchResult{Transport: mcp.TransportStdio, Process: process}},
	}
	return &defaultMCPRuntime{
		clients:   mcp.NewClientManager(connector),
		tools:     mcp.NewToolManager(),
		connector: connector,
	}
}

func TestRegisterAndCloseInstanceSuccess(t *testing.T) {
	server, process := newRuntimeMCPServer(t)
	defer server.Close()

	runtime := newStdioRuntime(process)
	command := "fake-mcp"
	instance := &store.MCPInstance{
		ID:         "inst-1",
		Name:       "expo",
		ServerKey:  "expo",
		LaunchType: mcp.LaunchCommand,
		Transport:  mcp.TransportStdio,
		Command:    &command,
		IsActive:   true,
	}

	manifest, err := runtime.RegisterInstance(context.Background(), instance)
	if err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}
	if len(manifest.Tools) == 0 {
		t.Fatalf("manifest tools empty: %+v", manifest)
	}

	if err := runtime.CloseInstance(instance.ID); err != nil {
		t.Fatalf("CloseInstance: %v", err)
	}
}
