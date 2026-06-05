package mcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// --- Launcher branches ---

func TestBuildLaunchSpecMissingCommandAndDefault(t *testing.T) {
	for _, lt := range []LaunchType{LaunchCommand, LaunchNPX, LaunchDocker} {
		if _, err := BuildLaunchSpec(InstanceConfig{LaunchType: lt}); !errors.Is(err, ErrInvalidInstanceConfig) {
			t.Fatalf("%s missing command: err = %v", lt, err)
		}
	}
	if _, err := BuildLaunchSpec(InstanceConfig{LaunchType: "bogus"}); !errors.Is(err, ErrInvalidInstanceConfig) {
		t.Fatalf("default launch type: err = %v", err)
	}
}

func TestLauncherLaunchInvalidType(t *testing.T) {
	l := NewLauncher(nil, nil)
	if _, err := l.Launch(context.Background(), InstanceConfig{LaunchType: "bogus"}); !errors.Is(err, ErrInvalidInstanceConfig) {
		t.Fatalf("invalid launch type: err = %v", err)
	}
}

func TestLaunchProcessNilRunner(t *testing.T) {
	l := NewLauncher(nil, nil)
	if _, err := l.Launch(context.Background(), InstanceConfig{LaunchType: LaunchCommand, Command: "x"}); err == nil {
		t.Fatal("nil runner: want error")
	}
}

func TestLaunchProcessBuildSpecError(t *testing.T) {
	l := NewLauncher(&fakeProcessRunner{}, nil)
	// LaunchCommand with empty command fails in BuildLaunchSpec.
	if _, err := l.Launch(context.Background(), InstanceConfig{LaunchType: LaunchCommand}); !errors.Is(err, ErrInvalidInstanceConfig) {
		t.Fatalf("build spec error: err = %v", err)
	}
}

type errProcessRunner struct{}

func (errProcessRunner) Start(context.Context, ProcessSpec) (Process, error) {
	return nil, errors.New("start failed")
}

func TestLaunchProcessStartError(t *testing.T) {
	l := NewLauncher(errProcessRunner{}, nil)
	if _, err := l.Launch(context.Background(), InstanceConfig{LaunchType: LaunchCommand, Command: "x"}); err == nil {
		t.Fatal("start error: want error")
	}
}

func TestLaunchHTTPMissingURL(t *testing.T) {
	l := NewLauncher(nil, nil)
	if _, err := l.Launch(context.Background(), InstanceConfig{LaunchType: LaunchHTTP}); !errors.Is(err, ErrInvalidInstanceConfig) {
		t.Fatalf("missing url: err = %v", err)
	}
}

func TestLaunchHTTPSSEFallbackError(t *testing.T) {
	// First streamable POST returns a fallback-eligible status, then /sse fails.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sse" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound) // streamable fallback trigger
	}))
	defer server.Close()
	l := NewLauncher(nil, server.Client())
	if _, err := l.Launch(context.Background(), InstanceConfig{
		LaunchType: LaunchHTTP, Transport: TransportStreamableHTTP, URL: server.URL,
	}); err == nil {
		t.Fatal("sse fallback failure: want error")
	}
}

// --- Discovery branches ---

func TestDiscoverNilGuards(t *testing.T) {
	var nilDisc *Discovery
	if _, err := nilDisc.Discover(context.Background(), ClientConfig{}); !errors.Is(err, ErrInvalidDiscovery) {
		t.Fatalf("nil discovery: err = %v", err)
	}
	d := &Discovery{}
	if _, err := d.Discover(context.Background(), ClientConfig{}); !errors.Is(err, ErrInvalidDiscovery) {
		t.Fatalf("empty discovery: err = %v", err)
	}
}

// --- ClientManager.Close error path (client.Close returns error) ---

func TestClientManagerCloseClientError(t *testing.T) {
	closeErr := errors.New("close boom")
	manager := &ClientManager{clients: map[string]Client{"docs": &fakeClient{err: closeErr}}}
	if err := manager.Close("docs"); !errors.Is(err, closeErr) {
		t.Fatalf("Close wrapped err = %v, want close boom", err)
	}
}

// --- Agent: exceeds max turns ---

func TestAgentExceedsMaxTurns(t *testing.T) {
	tools := NewToolManager()
	client := &fakeClient{callResult: CallResult{Content: "again"}}
	tools.RegisterClient("docs", client)
	if err := tools.RegisterManifest(Manifest{ClientID: "docs", Tools: []Tool{{Name: "loop"}}}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	// Provider always returns a tool call -> never terminates -> hits max turns.
	loopResp := &providers.ChatResponse{Choices: []providers.Choice{{Message: providers.Message{
		ToolCalls: []providers.ToolCall{{ID: "1", Function: providers.ToolCallFunc{Name: "docs__loop", Arguments: "{}"}}},
	}}}}
	responses := make([]*providers.ChatResponse, defaultAgentMaxTurns+2)
	for i := range responses {
		responses[i] = loopResp
	}
	provider := &fakeAgentProvider{responses: responses}
	agent := NewAgent(provider, providers.Key{}, tools)
	if _, err := agent.Run(context.Background(), &providers.ChatRequest{}); err == nil {
		t.Fatal("want max turns error")
	}
}

// --- ToolManager.RefreshManifest invalid manifest ---

func TestToolManagerRefreshManifestErrors(t *testing.T) {
	tm := NewToolManager()
	if err := tm.RefreshManifest(Manifest{}); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("empty client id: err = %v", err)
	}
	if err := tm.RefreshManifest(Manifest{ClientID: "c", Tools: []Tool{{Name: ""}}}); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("empty tool name: err = %v", err)
	}
}

// --- instances redactSecretMap: nil + secret/non-secret keys ---

func TestRedactSecretMapBranches(t *testing.T) {
	if redactSecretMap(nil) != nil {
		t.Fatal("nil map should stay nil")
	}
	got := redactSecretMap(map[string]string{"API_TOKEN": "s", "PLAIN": "v"})
	if got["API_TOKEN"] != RedactedValue {
		t.Fatalf("secret key not redacted: %v", got)
	}
	if got["PLAIN"] != "v" {
		t.Fatalf("plain key altered: %v", got)
	}
}

// --- toolResultContent default branch already; cover nil path explicitly ---

func TestToolResultContentNilAndDefault(t *testing.T) {
	if toolResultContent(nil) != "" {
		t.Fatal("nil -> empty")
	}
	if toolResultContent(42) != 42 {
		t.Fatal("default passthrough")
	}
}
