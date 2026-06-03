package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

func TestAdvancedMCPOAuthFlowCompletesRedirectAndPastedCallback(t *testing.T) {
	s := openIntegrationStore(t)
	instance := createIntegrationInstance(t, s, "atlassian-a", "account-a")

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callback := "http://localhost:20128/api/mcp/oauth/callback?instance_id=" + instance.ID + "&code=redirect-code&state=" + r.URL.Query().Get("state")
		http.Redirect(w, r, callback, http.StatusFound)
	}))
	defer authServer.Close()

	state := "redirect-state"
	if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		InstanceID:         instance.ID,
		State:              state,
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost:20128/api/mcp/oauth/callback",
		AuthorizationURL:   authServer.URL + "/authorize?state=" + state,
		ResourceURI:        "https://mcp.atlassian.com",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateMCPOAuthFlow redirect: %v", err)
	}

	client := authServer.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Get(authServer.URL + "/authorize?state=" + state)
	if err != nil {
		t.Fatalf("GET auth endpoint: %v", err)
	}
	_ = resp.Body.Close()
	callbackURL := resp.Header.Get("Location")
	if callbackURL == "" {
		t.Fatalf("callback location is empty")
	}

	engine := mcp.NewOAuthEngine(s, authServer.Client())
	account, err := engine.CompleteCallback(context.Background(), instance.ID, callbackURL)
	if err != nil {
		t.Fatalf("CompleteCallback redirect: %v", err)
	}
	if account.AccessToken != "mcp_redirect-code" {
		t.Fatalf("access token = %q, want redirect token", account.AccessToken)
	}

	if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		InstanceID:         instance.ID,
		State:              "pasted-state",
		CodeVerifierSecret: "verifier",
		ResourceURI:        "https://mcp.atlassian.com",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateMCPOAuthFlow pasted: %v", err)
	}
	pasted, err := engine.CompleteCallback(context.Background(), instance.ID, "http://localhost:20128/api/mcp/oauth/callback?code=pasted-code&state=pasted-state")
	if err != nil {
		t.Fatalf("CompleteCallback pasted: %v", err)
	}
	if pasted.AccessToken != "mcp_pasted-code" || pasted.InstanceID != instance.ID {
		t.Fatalf("pasted account = %+v, want pasted token for instance", pasted)
	}
}

func TestAdvancedMCPCommandLaunchListsAndExecutesTools(t *testing.T) {
	runner := &recordingRunner{}
	launcher := mcp.NewLauncher(runner, nil)
	result, err := launcher.Launch(context.Background(), mcp.InstanceConfig{
		ID:         "stdio-1",
		Name:       "stdio",
		ServerKey:  "fake",
		LaunchType: mcp.LaunchCommand,
		Transport:  mcp.TransportStdio,
		Command:    "fake-mcp",
		Args:       []string{"--stdio"},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if result.Transport != mcp.TransportStdio || runner.spec.Command != "fake-mcp" {
		t.Fatalf("launch result = %+v spec=%+v, want stdio fake command", result, runner.spec)
	}

	client := &integrationMCPClient{tools: []mcp.Tool{{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	clients := mcp.NewClientManager(&integrationConnector{client: client})
	tools := mcp.NewToolManager()
	discovery := mcp.NewDiscovery(clients, tools)
	manifest, err := discovery.Discover(context.Background(), mcp.ClientConfig{ID: "stdio-1", Name: "stdio", Transport: mcp.TransportStdio, Command: "fake-mcp"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(manifest.Tools) != 1 || manifest.Tools[0].Function.Name != "stdio-1__search" {
		t.Fatalf("manifest = %+v, want stable search tool", manifest)
	}
	resultCall, err := tools.Call(context.Background(), "stdio-1__search", json.RawMessage(`{"query":"mcp"}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resultCall.Content.(map[string]any)["ok"] != true || len(client.calls) != 1 || client.calls[0].Name != "search" {
		t.Fatalf("result=%+v calls=%+v, want search call", resultCall, client.calls)
	}
}

func TestAdvancedMCPNPXLaunchSpecIsOffline(t *testing.T) {
	spec, err := mcp.BuildLaunchSpec(mcp.InstanceConfig{
		Name:       "expo",
		ServerKey:  "expo",
		LaunchType: mcp.LaunchNPX,
		Transport:  mcp.TransportStdio,
		Command:    "@expo/mcp",
		Args:       []string{"--stdio"},
	})
	if err != nil {
		t.Fatalf("BuildLaunchSpec: %v", err)
	}
	if spec.Command != "npx" || strings.Join(spec.Args, " ") != "--yes @expo/mcp --stdio" {
		t.Fatalf("spec = %+v, want offline npx argv", spec)
	}
}

func TestAdvancedMCPDockerSpecWhenDockerAvailable(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker unavailable; skipping docker launch verification: %v", err)
	}
	if err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Run(); err != nil {
		t.Skipf("docker daemon unavailable; skipping docker launch verification: %v", err)
	}

	spec, err := mcp.BuildLaunchSpec(mcp.InstanceConfig{
		Name:       "docker-mcp",
		ServerKey:  "docker-mcp",
		LaunchType: mcp.LaunchDocker,
		Transport:  mcp.TransportStdio,
		Command:    "mcp/server:latest",
		Env:        map[string]string{"TOKEN": "secret"},
	})
	if err != nil {
		t.Fatalf("BuildLaunchSpec: %v", err)
	}
	if spec.Command != "docker" || !containsAll(spec.Args, []string{"run", "--rm", "-i", "-e", "TOKEN", "mcp/server:latest"}) {
		t.Fatalf("spec = %+v, want docker run argv", spec)
	}
}

func TestAdvancedMCPTokenRefreshKeepsSelectedAccount(t *testing.T) {
	s := openIntegrationStore(t)
	instance := createIntegrationInstance(t, s, "atlassian-a", "account-a")
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "account-a",
		ResourceURI:  "https://mcp.atlassian.com",
		AccessToken:  "old-token",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("Upsert account old: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "account-a",
		ResourceURI:  "https://mcp.atlassian.com",
		AccessToken:  "new-token",
		RefreshToken: "new-refresh",
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}); err != nil {
		t.Fatalf("Upsert account new: %v", err)
	}

	accounts, err := s.ListMCPOAuthAccounts(instance.ID)
	if err != nil {
		t.Fatalf("ListMCPOAuthAccounts: %v", err)
	}
	got, err := s.GetMCPInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if len(accounts) != 1 || accounts[0].AccessToken != "new-token" || *got.AccountLabel != "account-a" {
		t.Fatalf("accounts=%+v selected=%+v, want refreshed same account", accounts, got.AccountLabel)
	}
}

type recordingRunner struct {
	spec mcp.ProcessSpec
}

func (r *recordingRunner) Start(ctx context.Context, spec mcp.ProcessSpec) (mcp.Process, error) {
	r.spec = spec
	return integrationProcess{}, nil
}

type integrationProcess struct{}

func (p integrationProcess) Stdin() io.WriteCloser {
	return nopWriteCloser{Writer: &bytes.Buffer{}}
}

func (p integrationProcess) Stdout() io.ReadCloser {
	return io.NopCloser(bytes.NewReader(nil))
}

func (p integrationProcess) Stderr() *bytes.Buffer {
	return bytes.NewBufferString("ready\n")
}

func (p integrationProcess) Close() error {
	return nil
}

type integrationConnector struct {
	client mcp.Client
}

func (c *integrationConnector) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	return c.client, nil
}

type integrationMCPClient struct {
	tools []mcp.Tool
	calls []mcp.CallRequest
}

func (c *integrationMCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return c.tools, nil
}

func (c *integrationMCPClient) CallTool(ctx context.Context, req mcp.CallRequest) (mcp.CallResult, error) {
	c.calls = append(c.calls, req)
	return mcp.CallResult{Content: map[string]any{"ok": true}}, nil
}

func (c *integrationMCPClient) Close() error {
	return nil
}

func openIntegrationStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return s
}

func createIntegrationInstance(t *testing.T, s *store.Store, name, accountLabel string) *store.MCPInstance {
	t.Helper()
	url := "https://mcp.atlassian.com/mcp"
	instance := &store.MCPInstance{
		Name:         name,
		ServerKey:    "atlassian",
		LaunchType:   mcp.LaunchHTTP,
		Transport:    mcp.TransportStreamableHTTP,
		URL:          &url,
		AccountLabel: &accountLabel,
		IsActive:     true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	return instance
}

func containsAll(values, wants []string) bool {
	joined := "\x00" + strings.Join(values, "\x00") + "\x00"
	for _, want := range wants {
		if !strings.Contains(joined, "\x00"+want+"\x00") {
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
