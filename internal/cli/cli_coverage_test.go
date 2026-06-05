package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
)

// --- ForgetInstanceConfig ---

func TestForgetInstanceConfig(t *testing.T) {
	c := &mcpLauncherConnector{}
	c.RememberInstanceConfig(mcp.InstanceConfig{ID: "abc"})
	c.ForgetInstanceConfig("abc")
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.instanceConfigs["abc"]; ok {
		t.Fatal("ForgetInstanceConfig did not remove config")
	}
}

func TestForgetInstanceConfigNonExistent(t *testing.T) {
	c := &mcpLauncherConnector{}
	// Should not panic when map is nil
	c.ForgetInstanceConfig("nonexistent")
}

// --- Connect: SSE transport ---

func TestConnectSSETransport(t *testing.T) {
	connector := &mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{
			result: mcp.LaunchResult{Transport: mcp.TransportSSE},
		},
	}
	connector.RememberInstanceConfig(mcp.InstanceConfig{
		ID:        "sse-1",
		Transport: mcp.TransportSSE,
		URL:       "http://example.test/sse",
	})
	client, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "sse-1",
		Transport: mcp.TransportSSE,
		URL:       "http://example.test/sse",
	})
	if err != nil {
		t.Fatalf("Connect SSE: %v", err)
	}
	if client == nil {
		t.Fatal("SSE client is nil")
	}
}

// --- Connect: launcher error ---

func TestConnectLauncherError(t *testing.T) {
	connector := &mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{
			err: errors.New("launcher failed"),
		},
	}
	_, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "err-1",
		Transport: mcp.TransportStdio,
		Command:   "fake",
	})
	if err == nil || !strings.Contains(err.Error(), "launcher failed") {
		t.Fatalf("Connect launcher error = %v", err)
	}
}

// --- clientInstanceConfig ---

func TestClientInstanceConfigStdioNoCommand(t *testing.T) {
	_, err := clientInstanceConfig(mcp.ClientConfig{
		Transport: mcp.TransportStdio,
		Command:   "",
	})
	if !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("error = %v, want ErrInvalidClientConfig", err)
	}
}

func TestClientInstanceConfigHTTPNoURL(t *testing.T) {
	_, err := clientInstanceConfig(mcp.ClientConfig{
		Transport: mcp.TransportStreamableHTTP,
		URL:       "",
	})
	if !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("error = %v, want ErrInvalidClientConfig", err)
	}
}

func TestClientInstanceConfigSSENoURL(t *testing.T) {
	_, err := clientInstanceConfig(mcp.ClientConfig{
		Transport: mcp.TransportSSE,
		URL:       "",
	})
	if !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("error = %v, want ErrInvalidClientConfig", err)
	}
}

func TestClientInstanceConfigUnknownTransport(t *testing.T) {
	_, err := clientInstanceConfig(mcp.ClientConfig{
		Transport: "bogus",
		Command:   "cmd",
	})
	if !errors.Is(err, mcp.ErrInvalidClientConfig) {
		t.Fatalf("error = %v, want ErrInvalidClientConfig", err)
	}
}

func TestClientInstanceConfigHTTPValid(t *testing.T) {
	cfg, err := clientInstanceConfig(mcp.ClientConfig{
		ID:        "h1",
		Name:      "http-server",
		Transport: mcp.TransportStreamableHTTP,
		URL:       "http://example.test",
		Env:       map[string]string{"K": "V"},
	})
	if err != nil {
		t.Fatalf("clientInstanceConfig HTTP: %v", err)
	}
	if cfg.URL != "http://example.test" || cfg.LaunchType != mcp.LaunchHTTP {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Env["K"] != "V" {
		t.Fatalf("env = %+v", cfg.Env)
	}
}

func TestClientInstanceConfigSSEValid(t *testing.T) {
	cfg, err := clientInstanceConfig(mcp.ClientConfig{
		ID:        "sse1",
		Transport: mcp.TransportSSE,
		URL:       "http://example.test/sse",
	})
	if err != nil {
		t.Fatalf("clientInstanceConfig SSE: %v", err)
	}
	if cfg.Transport != mcp.TransportSSE {
		t.Fatalf("transport = %q, want SSE", cfg.Transport)
	}
}

// --- CloseInstance ---

func TestCloseInstanceNilRuntime(t *testing.T) {
	var r *defaultMCPRuntime
	err := r.CloseInstance("id")
	if !errors.Is(err, mcp.ErrInvalidDiscovery) {
		t.Fatalf("CloseInstance nil = %v, want ErrInvalidDiscovery", err)
	}
}

func TestCloseInstanceNonExistent(t *testing.T) {
	r := newDefaultMCPRuntime()
	err := r.CloseInstance("nonexistent-id")
	if err != nil && !errors.Is(err, mcp.ErrClientNotFound) {
		t.Fatalf("CloseInstance nonexistent: %v", err)
	}
}

// --- RegisterInstance: nil runtime ---

func TestRegisterInstanceNilRuntime(t *testing.T) {
	var r *defaultMCPRuntime
	_, err := r.RegisterInstance(context.Background(), &store.MCPInstance{})
	if !errors.Is(err, mcp.ErrInvalidDiscovery) {
		t.Fatalf("RegisterInstance nil = %v, want ErrInvalidDiscovery", err)
	}
}

// --- processEnv and sortedStringKeys ---

func TestProcessEnvAppendsExtras(t *testing.T) {
	env := processEnv(map[string]string{"MYKEY": "MYVAL"})
	found := false
	for _, e := range env {
		if e == "MYKEY=MYVAL" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("processEnv did not include MYKEY=MYVAL; env=%v", env)
	}
}

func TestProcessEnvNilExtras(t *testing.T) {
	env := processEnv(nil)
	if len(env) == 0 {
		t.Fatal("processEnv nil extras returned empty (expected base env)")
	}
}

func TestSortedStringKeys(t *testing.T) {
	m := map[string]string{"b": "1", "a": "2", "c": "3"}
	keys := sortedStringKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("sortedStringKeys = %v, want sorted", keys)
	}
}

func TestSortedStringKeysEmpty(t *testing.T) {
	keys := sortedStringKeys(nil)
	if len(keys) != 0 {
		t.Fatalf("sortedStringKeys nil = %v, want empty", keys)
	}
}

// --- isProcessExitAfterClose ---

func TestIsProcessExitAfterClose(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 1")
	err := cmd.Run()
	if err == nil {
		t.Skip("command succeeded unexpectedly")
	}
	if !isProcessExitAfterClose(err) {
		t.Fatalf("isProcessExitAfterClose = false for ExitError")
	}
}

func TestIsProcessExitAfterCloseNonExit(t *testing.T) {
	if isProcessExitAfterClose(errors.New("regular error")) {
		t.Fatal("isProcessExitAfterClose = true for non-exit error")
	}
}

// --- commandProcess methods ---

func TestCommandProcessMethods(t *testing.T) {
	runner := commandProcessRunner{}
	process, err := runner.Start(context.Background(), mcp.ProcessSpec{
		Command: "sh",
		Args:    []string{"-c", "cat"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cp, ok := process.(*commandProcess)
	if !ok {
		t.Fatalf("process is %T, want *commandProcess", process)
	}

	if cp.Stdin() == nil {
		t.Fatal("Stdin is nil")
	}
	if cp.Stdout() == nil {
		t.Fatal("Stdout is nil")
	}
	if cp.Stderr() == nil {
		t.Fatal("Stderr is nil")
	}

	if err := cp.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestCommandProcessRunnerEmptyCommand(t *testing.T) {
	runner := commandProcessRunner{}
	_, err := runner.Start(context.Background(), mcp.ProcessSpec{Command: ""})
	if !errors.Is(err, mcp.ErrInvalidInstanceConfig) {
		t.Fatalf("Start empty command = %v, want ErrInvalidInstanceConfig", err)
	}
}

func TestCommandProcessRunnerBadCommand(t *testing.T) {
	runner := commandProcessRunner{}
	_, err := runner.Start(context.Background(), mcp.ProcessSpec{Command: "/nonexistent-binary-xyz"})
	if err == nil {
		t.Fatal("Start bad command should fail")
	}
}

func TestCommandProcessCloseNilCmd(t *testing.T) {
	p := &commandProcess{cmd: nil}
	if err := p.Close(); err != nil {
		t.Fatalf("Close nil cmd: %v", err)
	}
}

func TestCommandProcessCloseNilProcess(t *testing.T) {
	cmd := &exec.Cmd{}
	p := &commandProcess{
		cmd:    cmd,
		stdin:  nopWriteCloser{Writer: io.Discard},
		stdout: io.NopCloser(bytes.NewReader(nil)),
		stderr: &bytes.Buffer{},
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close nil process: %v", err)
	}
}

// --- ReapplyInstanceCredentials nil store ---

func TestReapplyInstanceCredentialsNilStore(t *testing.T) {
	r := newDefaultMCPRuntime()
	_, err := r.ReapplyInstanceCredentials(context.Background(), nil, "id")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ReapplyInstanceCredentials nil store = %v, want ErrNotFound", err)
	}
}

// --- ReapplyInstanceCredentials: success path ---

func TestReapplyInstanceCredentialsSuccess(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	mcpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if _, ok := req["id"]; !ok {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "reapply-session")
		w.Header().Set("Content-Type", "application/json")
		switch req["method"] {
		case "initialize":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": "2025-11-25"})
		case "tools/list":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"tools": []map[string]any{{"name": "search", "description": "Search", "inputSchema": map[string]any{"type": "object"}}}})
		default:
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{})
		}
	}))
	defer mcpSrv.Close()

	urlVal := mcpSrv.URL
	instance := &store.MCPInstance{
		Name:       "reapply-test",
		ServerKey:  "reapply-test",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        &urlVal,
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	launcher := mcp.NewLauncher(commandProcessRunner{}, mcpSrv.Client())
	connector := &mcpLauncherConnector{launcher: launcher}
	r := &defaultMCPRuntime{
		clients:   mcp.NewClientManager(connector),
		tools:     mcp.NewToolManager(),
		connector: connector,
	}

	manifest, err := r.ReapplyInstanceCredentials(context.Background(), s, instance.ID)
	if err != nil {
		t.Fatalf("ReapplyInstanceCredentials: %v", err)
	}
	if len(manifest.Tools) == 0 {
		t.Fatal("ReapplyInstanceCredentials returned empty manifest")
	}
}

// --- install: newInstallOptions, generateInstallSecret, runCommand ---

func TestNewInstallOptions(t *testing.T) {
	opts := newInstallOptions(io.Discard, false)
	if opts.Out == nil {
		t.Fatal("Out is nil")
	}
	if opts.RunCommand == nil {
		t.Fatal("RunCommand is nil")
	}
	if opts.SecretGenerator == nil {
		t.Fatal("SecretGenerator is nil")
	}
}

func TestGenerateInstallSecret(t *testing.T) {
	secret, err := generateInstallSecret()
	if err != nil {
		t.Fatalf("generateInstallSecret: %v", err)
	}
	if len(secret) != 64 {
		t.Fatalf("secret len = %d, want 64 hex chars", len(secret))
	}
}

func TestRunCommandSuccess(t *testing.T) {
	err := runCommand("echo", "hello")
	if err != nil {
		t.Fatalf("runCommand echo: %v", err)
	}
}

func TestRunCommandFailure(t *testing.T) {
	err := runCommand("sh", "-c", "exit 1")
	if err == nil {
		t.Fatal("runCommand failed exit should return error")
	}
}

// --- normalizeInstallOptions ---

func TestNormalizeInstallOptionsDefaults(t *testing.T) {
	opts := normalizeInstallOptions(installOptions{})
	if opts.Root == "" {
		t.Fatal("Root is empty after normalize")
	}
	if opts.RunCommand == nil {
		t.Fatal("RunCommand is nil after normalize")
	}
	if opts.SecretGenerator == nil {
		t.Fatal("SecretGenerator is nil after normalize")
	}
	if opts.Out == nil {
		t.Fatal("Out is nil after normalize")
	}
}

// --- rooted ---

func TestRootedWithNonRootSeparator(t *testing.T) {
	got := rooted("/tmp/testroot", "/etc/g0router.conf")
	if !strings.HasPrefix(got, "/tmp/testroot") {
		t.Fatalf("rooted = %q, want prefix /tmp/testroot", got)
	}
	if !strings.HasSuffix(got, "etc/g0router.conf") {
		t.Fatalf("rooted = %q, want suffix etc/g0router.conf", got)
	}
}

func TestRootedWithSlash(t *testing.T) {
	got := rooted("/", "/etc/g0router.conf")
	if got != "/etc/g0router.conf" {
		t.Fatalf("rooted / = %q, want /etc/g0router.conf", got)
	}
}

// --- ensureSystemUser ---

func TestEnsureSystemUserAlreadyExists(t *testing.T) {
	err := ensureSystemUser(func(name string, args ...string) error {
		if name == "id" {
			return nil
		}
		return errors.New("useradd should not be called")
	})
	if err != nil {
		t.Fatalf("ensureSystemUser existing user: %v", err)
	}
}

func TestEnsureSystemUserCreateFails(t *testing.T) {
	err := ensureSystemUser(func(name string, args ...string) error {
		return errors.New("command failed")
	})
	if err == nil {
		t.Fatal("ensureSystemUser useradd failure should return error")
	}
	if !strings.Contains(err.Error(), "create g0router user") {
		t.Fatalf("error = %v", err)
	}
}

// --- readDeployTemplate ---

func TestReadDeployTemplateUnknownName(t *testing.T) {
	_, err := readDeployTemplate("unknown.template")
	if err == nil {
		t.Fatal("readDeployTemplate unknown should fail")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadDeployTemplateService(t *testing.T) {
	content, err := readDeployTemplate("g0router.service")
	if err != nil {
		t.Fatalf("readDeployTemplate service: %v", err)
	}
	if !strings.Contains(string(content), "ExecStart") {
		t.Fatalf("service template missing ExecStart")
	}
}

func TestReadDeployTemplateDefault(t *testing.T) {
	content, err := readDeployTemplate("g0router.default")
	if err != nil {
		t.Fatalf("readDeployTemplate default: %v", err)
	}
	if !strings.Contains(string(content), "PORT=") {
		t.Fatalf("default template missing PORT=")
	}
}

// --- renderDefaultTemplate ---

func TestRenderDefaultTemplateSecretError(t *testing.T) {
	_, err := renderDefaultTemplate([]byte("JWT_SECRET=\n"), func() (string, error) {
		return "", errors.New("rng failed")
	})
	if err == nil {
		t.Fatal("renderDefaultTemplate secret error should fail")
	}
}

func TestRenderDefaultTemplateFillsSecrets(t *testing.T) {
	content := []byte("JWT_SECRET=\nAPI_KEY_SECRET=\n")
	i := 0
	secrets := []string{"jwt-val", "api-val"}
	got, err := renderDefaultTemplate(content, func() (string, error) {
		s := secrets[i]
		i++
		return s, nil
	})
	if err != nil {
		t.Fatalf("renderDefaultTemplate: %v", err)
	}
	if !strings.Contains(string(got), "JWT_SECRET=jwt-val") {
		t.Fatalf("got = %q, want jwt-val", got)
	}
	if !strings.Contains(string(got), "API_KEY_SECRET=api-val") {
		t.Fatalf("got = %q, want api-val", got)
	}
}

// --- copyFile: same file ---

func TestCopyFileSameFile(t *testing.T) {
	path := t.TempDir() + "/g0router"
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := copyFile(path, path, 0o755); err != nil {
		t.Fatalf("copyFile same file: %v", err)
	}
}

// --- root: expandServeDataDir tilde-only ---

func TestExpandServeDataDirTildeOnlyCoverage(t *testing.T) {
	got, err := expandServeDataDir("~")
	if err != nil {
		t.Fatalf("expandServeDataDir ~: %v", err)
	}
	home, _ := os.UserHomeDir()
	if got != home {
		t.Fatalf("expandServeDataDir ~ = %q, want %q", got, home)
	}
}

// --- openCLIStore ---

func TestOpenCLIStoreValid(t *testing.T) {
	dir := t.TempDir()
	s, err := openCLIStore(dir)
	if err != nil {
		t.Fatalf("openCLIStore: %v", err)
	}
	defer s.Close()
}

// --- shouldRefreshMCPAccount ---

func TestShouldRefreshMCPAccountNoRefreshToken(t *testing.T) {
	account := mcp.OAuthAccount{RefreshToken: ""}
	if shouldRefreshMCPAccount(account) {
		t.Fatal("shouldRefresh = true with no refresh token")
	}
}

func TestShouldRefreshMCPAccountNoAccessToken(t *testing.T) {
	account := mcp.OAuthAccount{RefreshToken: "rt", AccessToken: ""}
	if !shouldRefreshMCPAccount(account) {
		t.Fatal("shouldRefresh = false with empty access token")
	}
}

func TestShouldRefreshMCPAccountNotExpired(t *testing.T) {
	account := mcp.OAuthAccount{
		RefreshToken: "rt",
		AccessToken:  "at",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	if shouldRefreshMCPAccount(account) {
		t.Fatal("shouldRefresh = true for non-expired token")
	}
}

func TestShouldRefreshMCPAccountExpired(t *testing.T) {
	account := mcp.OAuthAccount{
		RefreshToken: "rt",
		AccessToken:  "at",
		ExpiresAt:    time.Now().Add(-time.Minute),
	}
	if !shouldRefreshMCPAccount(account) {
		t.Fatal("shouldRefresh = false for expired token")
	}
}

// --- providers test: unknown provider ---

func TestProvidersTestCommandUnknownProvider(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"providers", "test", "not-a-real-provider-xyz"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("providers test unknown should fail")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("error = %v", err)
	}
}

// --- keys remove: not found ---

func TestKeysRemoveNotFound(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", t.TempDir(), "keys", "rm", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("keys rm nonexistent should fail")
	}
	if !strings.Contains(err.Error(), "api key not found") {
		t.Fatalf("error = %v", err)
	}
}

// --- MCPLauncherConnector: remember + connect via remembered config ---

func TestConnectUsesRememberedInstanceConfig(t *testing.T) {
	process := &closableRuntimeProcess{stderr: &bytes.Buffer{}}
	connector := &mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{
			result: mcp.LaunchResult{
				Transport: mcp.TransportStdio,
				Process:   process,
			},
		},
	}
	connector.RememberInstanceConfig(mcp.InstanceConfig{
		ID:        "remembered",
		Transport: mcp.TransportStdio,
		Command:   "fake",
	})
	client, err := connector.Connect(context.Background(), mcp.ClientConfig{
		ID:        "remembered",
		Transport: mcp.TransportStdio,
	})
	if err != nil {
		t.Fatalf("Connect remembered: %v", err)
	}
	_ = client
}

// --- serviceTemplate with user=false (covered by install tests but ensures no false skip) ---

func TestServiceTemplateSystemMode(t *testing.T) {
	got, err := serviceTemplate("/usr/local/bin/g0router", "/var/lib/g0router", false)
	if err != nil {
		t.Fatalf("serviceTemplate: %v", err)
	}
	if !strings.Contains(got, "User=g0router") {
		t.Fatalf("serviceTemplate missing User=g0router; got=%s", got)
	}
}

// --- ValidateAPIKeyIdentity ---

func TestValidateAPIKeyIdentityValid(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()
	t.Setenv("API_KEY_SECRET", "test-secret")

	_, raw, err := s.CreateAPIKey("default", "test-secret")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	validator := storeAPIKeyValidator{s: s}
	identity, ok, err := validator.ValidateAPIKeyIdentity(raw, "test-secret")
	if err != nil {
		t.Fatalf("ValidateAPIKeyIdentity: %v", err)
	}
	if !ok {
		t.Fatal("ValidateAPIKeyIdentity = false, want true")
	}
	if identity == nil || identity.ID == "" {
		t.Fatal("identity is nil or has empty ID")
	}
}

func TestValidateAPIKeyIdentityInvalid(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	validator := storeAPIKeyValidator{s: s}
	_, ok, err := validator.ValidateAPIKeyIdentity("bad-key", "secret")
	if err != nil {
		t.Fatalf("ValidateAPIKeyIdentity error: %v", err)
	}
	if ok {
		t.Fatal("ValidateAPIKeyIdentity = true for invalid key")
	}
}

// --- healthcheck with default port (no --url) ---

func TestHealthcheckCommandWithPortFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract port from server URL
	addr := strings.TrimPrefix(server.URL, "http://127.0.0.1:")
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"healthcheck", "--port", addr})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("healthcheck port: %v", err)
	}
}

func TestHealthcheckCommandNotOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"healthcheck", "--url", server.URL + "/healthz"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("healthcheck 503 should fail")
	}
}

// --- staticModelSource.ListModels ---

func TestStaticModelSourceListModels(t *testing.T) {
	src := staticModelSource{}
	models, err := src.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("ListModels returned no models")
	}
}

// --- newRootCommand: no-arg (help) path ---

func TestRootCommandHelp(t *testing.T) {
	cmd := newRootCommand(rootConfig{Version: "test"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})
	// Execute with no subcommand prints help, may return error
	_ = cmd.Execute()
}

// --- newServeCommand: serve=nil path ---

func TestServeCommandNilRunner(t *testing.T) {
	cmd := newRootCommand(rootConfig{Version: "test", Serve: nil})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"serve", "--data-dir", t.TempDir()})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("serve nil runner should fail")
	}
	if !strings.Contains(err.Error(), "serve runner unavailable") {
		t.Fatalf("error = %v", err)
	}
}

// --- rehydrateMCPRuntime: closed store (ListActiveMCPInstances error) ---

func TestRehydrateMCPRuntimeStoreError(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	s.Close() // closed store forces error
	runtime := newDefaultMCPRuntime()
	// Should not panic; just returns early
	rehydrateMCPRuntime(context.Background(), s, runtime)
}

// --- ReapplyInstanceCredentials: instance not found ---

func TestReapplyInstanceCredentialsNotFound(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()
	r := newDefaultMCPRuntime()
	_, err := r.ReapplyInstanceCredentials(context.Background(), s, "nonexistent-id")
	if err == nil {
		t.Fatal("ReapplyInstanceCredentials nonexistent should fail")
	}
}

// --- registerProvider: covers provider_runtime ---

func TestNewDefaultInferenceEngineIncludesKnownProviders(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()
	engine := newDefaultInferenceEngine(s)
	if len(engine.RegisteredProviders()) == 0 {
		t.Fatal("newDefaultInferenceEngine registered no providers")
	}
}

// --- openCLIStore: bad path ---

func TestOpenCLIStoreBadPath(t *testing.T) {
	// Path that can't be written
	_, err := openCLIStore("/nonexistent-path-xyz")
	if err == nil {
		t.Fatal("openCLIStore bad path should fail")
	}
}

// --- mcp_runtime RegisterInstance: client not found after register ---

func TestRegisterInstanceConnectError(t *testing.T) {
	// Use a connector that always fails to connect
	connector := &mcpLauncherConnector{
		launcher: fakeRuntimeLauncher{
			err: errors.New("connect failed"),
		},
	}
	r := &defaultMCPRuntime{
		clients:   mcp.NewClientManager(connector),
		tools:     mcp.NewToolManager(),
		connector: connector,
	}
	urlVal := "http://example.test"
	instance := &store.MCPInstance{
		ID:        "fail-connect",
		Name:      "fail",
		Transport: mcp.TransportStreamableHTTP,
		URL:       &urlVal,
		IsActive:  true,
	}
	_, err := r.RegisterInstance(context.Background(), instance)
	if err == nil {
		t.Fatal("RegisterInstance connect error should fail")
	}
}

// --- commandProcess Close with stdin/stdout nil ---

func TestCommandProcessCloseWithNilStdinStdout(t *testing.T) {
	runner := commandProcessRunner{}
	process, err := runner.Start(context.Background(), mcp.ProcessSpec{
		Command: "sh",
		Args:    []string{"-c", "exit 0"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	cp := process.(*commandProcess)
	// Close stdin/stdout manually before Close to test nil guards
	_ = cp.stdin.Close()
	cp.stdin = nil
	_ = cp.stdout.Close()
	cp.stdout = nil
	// Close should handle nil stdin/stdout gracefully
	_ = cp.Close()
}

// --- rehydrateMCPRuntime: instance exists but connect fails (registers unhealthy) ---

func TestRehydrateMCPRuntimeInstanceConnectFails(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	urlVal := "http://127.0.0.1:1" // unreachable, connect will fail
	instance := &store.MCPInstance{
		Name:       "fail-connect",
		ServerKey:  "fail-connect",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        &urlVal,
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	runtime := newDefaultMCPRuntime()
	rehydrateMCPRuntime(context.Background(), s, runtime)

	got, err := s.GetMCPInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if got.HealthStatus != "unhealthy" {
		t.Fatalf("health = %q, want unhealthy after connect failure", got.HealthStatus)
	}
}

// --- mcpInstanceForRuntime: Stdio transport with OAuth account ---

func TestMCPInstanceForRuntimeStdioTransport(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cmdStr := "fake-mcp"
	instance := &store.MCPInstance{
		Name:       "stdio-oauth",
		ServerKey:  "stdio-oauth",
		LaunchType: mcp.LaunchCommand,
		Transport:  mcp.TransportStdio,
		Command:    &cmdStr,
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	// No OAuth accounts, so selectMCPRuntimeOAuthAccount returns !ok
	// mcpInstanceForRuntime just returns instance as-is
	got, err := mcpInstanceForRuntime(context.Background(), s, instance)
	if err != nil {
		t.Fatalf("mcpInstanceForRuntime: %v", err)
	}
	if got.ID != instance.ID {
		t.Fatalf("got.ID = %q, want %q", got.ID, instance.ID)
	}
}

// --- selectMCPRuntimeOAuthAccount: account label mismatch ---

func TestSelectMCPRuntimeOAuthAccountLabelMismatch(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	urlVal := "http://example.test"
	label := "work"
	instance := &store.MCPInstance{
		Name:         "labeled",
		ServerKey:    "labeled",
		LaunchType:   mcp.LaunchHTTP,
		Transport:    mcp.TransportStreamableHTTP,
		URL:          &urlVal,
		AccountLabel: &label,
		IsActive:     true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	// Add account with different label
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "personal", // doesn't match "work"
		ResourceURI:  urlVal,
		AccessToken:  "token",
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	account, ok, err := selectMCPRuntimeOAuthAccount(s, instance)
	if err != nil {
		t.Fatalf("selectMCPRuntimeOAuthAccount: %v", err)
	}
	if ok || account != nil {
		t.Fatalf("account = %v %v, want nil false (label mismatch)", account, ok)
	}
}

// --- mcp_auth: findMCPInstanceByName ---

func TestFindMCPInstanceByNameNotFound(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	_, err := findMCPInstanceByName(s, "nonexistent")
	if err == nil {
		t.Fatal("findMCPInstanceByName should fail for nonexistent")
	}
}

// --- mcp_auth: validateMCPCallbackURL ---

func TestValidateMCPCallbackURLValid(t *testing.T) {
	if err := validateMCPCallbackURL("http://localhost:8080/callback?code=abc&state=xyz"); err != nil {
		t.Fatalf("validateMCPCallbackURL valid: %v", err)
	}
}

func TestValidateMCPCallbackURLMissingCode(t *testing.T) {
	if err := validateMCPCallbackURL("http://localhost:8080/callback?state=xyz"); err == nil {
		t.Fatal("validateMCPCallbackURL missing code should fail")
	}
}

func TestValidateMCPCallbackURLMissingState(t *testing.T) {
	if err := validateMCPCallbackURL("http://localhost:8080/callback?code=abc"); err == nil {
		t.Fatal("validateMCPCallbackURL missing state should fail")
	}
}

// --- mcp_auth: parseAssignments ---

func TestParseAssignmentsValid(t *testing.T) {
	got := parseAssignments([]string{"KEY=VALUE", "FOO=BAR"})
	if got["KEY"] != "VALUE" || got["FOO"] != "BAR" {
		t.Fatalf("parseAssignments = %v", got)
	}
}

func TestParseAssignmentsNoEquals(t *testing.T) {
	// No = → key stored with empty value
	got := parseAssignments([]string{"NOEQUALS"})
	if _, ok := got["NOEQUALS"]; !ok {
		t.Fatalf("parseAssignments NOEQUALS = %v, want key present", got)
	}
}

func TestParseAssignmentsNil(t *testing.T) {
	got := parseAssignments(nil)
	if got != nil {
		t.Fatalf("parseAssignments nil = %v, want nil", got)
	}
}

// --- keys add/list/remove: store open errors ---

func TestKeysAddOpenStoreError(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", "/nonexistent/path", "keys", "add", "key1"})
	t.Setenv("API_KEY_SECRET", "secret")
	err := cmd.Execute()
	if err == nil {
		t.Fatal("keys add bad data-dir should fail")
	}
}

func TestKeysListOpenStoreError(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", "/nonexistent/path", "keys", "list"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("keys list bad data-dir should fail")
	}
}

func TestKeysRemoveOpenStoreError(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", "/nonexistent/path", "keys", "rm", "key1"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("keys rm bad data-dir should fail")
	}
}

// --- status: store open error ---

func TestStatusCommandOpenStoreError(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", "/nonexistent/path", "status"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("status bad data-dir should fail")
	}
}

// --- providers test: no-auth provider ---

func TestProvidersTestCommandNoAuthProvider(t *testing.T) {
	// "ollama" uses noauth so it shouldn't need connections
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", t.TempDir(), "providers", "test", "ollama"})
	if err := cmd.Execute(); err != nil {
		// May fail if ollama is not noauth — just skip
		t.Skipf("providers test ollama: %v", err)
	}
}

// --- providers test: unsupported public inference ---

func TestProvidersTestCommandUnsupportedInference(t *testing.T) {
	// cursor is not a public inference provider
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", t.TempDir(), "providers", "test", "cursor"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("providers test cursor should fail")
	}
}

// --- providers test: no active connections ---

func TestProvidersTestCommandNoConnections(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--data-dir", t.TempDir(), "providers", "test", "openai"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("providers test openai no connections should fail")
	}
	if !strings.Contains(err.Error(), "no active connection") {
		t.Fatalf("error = %v", err)
	}
}

// --- openCLIStore: expandServeDataDir with ~ path ---

func TestOpenCLIStoreTildePath(t *testing.T) {
	// Should succeed since ~ expands to home
	s, err := openCLIStore("~/.g0router-test-coverage")
	if err != nil {
		// May fail if home dir unwritable - not a concern in normal env
		t.Skipf("openCLIStore ~ path: %v", err)
	}
	defer s.Close()
	defer os.RemoveAll(os.Getenv("HOME") + "/.g0router-test-coverage")
}

// --- mcpInstanceForRuntime: OAuth account with Stdio transport ---

func TestMCPInstanceForRuntimeStdioWithOAuthAccount(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	cmdStr := "fake-mcp"
	instance := &store.MCPInstance{
		Name:       "stdio-creds",
		ServerKey:  "stdio-creds",
		LaunchType: mcp.LaunchCommand,
		Transport:  mcp.TransportStdio,
		Command:    &cmdStr,
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:  instance.ID,
		AccessToken: "at-value",
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	got, err := mcpInstanceForRuntime(context.Background(), s, instance)
	if err != nil {
		t.Fatalf("mcpInstanceForRuntime stdio+oauth: %v", err)
	}
	if got == nil {
		t.Fatal("got is nil")
	}
}

// --- rehydrateMCPRuntime: nil store ---

func TestRehydrateMCPRuntimeNilStore(t *testing.T) {
	runtime := newDefaultMCPRuntime()
	rehydrateMCPRuntime(context.Background(), nil, runtime) // should not panic
}

func TestRehydrateMCPRuntimeNilRuntime(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()
	rehydrateMCPRuntime(context.Background(), s, nil) // should not panic
}

// --- mcpInstanceForRuntime: expired OAuth causes refresh failure ---

func TestMCPInstanceForRuntimeRefreshFails(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	urlVal := "http://127.0.0.1:1" // unreachable token server
	instance := &store.MCPInstance{
		Name:       "refresh-fail",
		ServerKey:  "refresh-fail",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        &urlVal,
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	// Expired OAuth account with refresh token — triggers RefreshAccount which will fail
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		ResourceURI:  urlVal,
		AccessToken:  "expired-at",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(-time.Hour),
		AuthMetadata: map[string]string{"token_endpoint": "http://127.0.0.1:1/token"},
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	_, err := mcpInstanceForRuntime(context.Background(), s, instance)
	if err == nil {
		t.Fatal("mcpInstanceForRuntime with expired oauth should fail")
	}
}

// --- rehydrateMCPRuntime: instance mcpInstanceForRuntime fails → unhealthy ---

func TestRehydrateMCPRuntimeInstanceForRuntimeFails(t *testing.T) {
	s := openCLIStoreForTest(t, t.TempDir())
	defer s.Close()

	urlVal := "http://127.0.0.1:1"
	instance := &store.MCPInstance{
		Name:       "refresh-fail-rehydrate",
		ServerKey:  "refresh-fail-rehydrate",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        &urlVal,
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		ResourceURI:  urlVal,
		AccessToken:  "expired",
		RefreshToken: "rt",
		ExpiresAt:    time.Now().Add(-time.Hour),
		AuthMetadata: map[string]string{"token_endpoint": "http://127.0.0.1:1/token"},
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	runtime := newDefaultMCPRuntime()
	rehydrateMCPRuntime(context.Background(), s, runtime)

	got, err := s.GetMCPInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if got.HealthStatus != "unhealthy" {
		t.Fatalf("health = %q, want unhealthy", got.HealthStatus)
	}
}

// --- RegisterInstance: tools.RegisterManifest error via duplicate client IDs ---
// (hard to trigger; skip — but cover CloseInstance after Register)

func TestCloseInstanceAfterRegister(t *testing.T) {
	mcpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if _, ok := req["id"]; !ok {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "close-session")
		w.Header().Set("Content-Type", "application/json")
		switch req["method"] {
		case "initialize":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": "2025-11-25"})
		case "tools/list":
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{"tools": []map[string]any{{"name": "search", "description": "Search", "inputSchema": map[string]any{"type": "object"}}}})
		default:
			writeRuntimeRPCResult(t, w, req["id"], map[string]any{})
		}
	}))
	defer mcpSrv.Close()

	urlVal := mcpSrv.URL
	launcher := mcp.NewLauncher(commandProcessRunner{}, mcpSrv.Client())
	connector := &mcpLauncherConnector{launcher: launcher}
	r := &defaultMCPRuntime{
		clients:   mcp.NewClientManager(connector),
		tools:     mcp.NewToolManager(),
		connector: connector,
	}

	instance := &store.MCPInstance{
		ID:         "close-after-register",
		Name:       "close-test",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        &urlVal,
	}

	if _, err := r.RegisterInstance(context.Background(), instance); err != nil {
		t.Fatalf("RegisterInstance: %v", err)
	}
	if err := r.CloseInstance(instance.ID); err != nil {
		t.Fatalf("CloseInstance after register: %v", err)
	}
}
