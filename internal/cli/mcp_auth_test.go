package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestMCPOAuthCompleteCommandCompletesPastedCallback(t *testing.T) {
	dataDir := t.TempDir()
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Fatalf("path = %q, want /token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.PostForm.Get("code") != "secret-code" || r.PostForm.Get("code_verifier") != "verifier" {
			t.Fatalf("token form = %+v, want code and verifier", r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"account_label": "expo-work",
			"expires_in":    600,
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer tokenServer.Close()

	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	instance := createCLIMCPInstance(t, s, "expo")
	if err := s.CreateMCPOAuthFlow(&store.MCPOAuthFlow{
		InstanceID:         instance.ID,
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost:3000/callback?instance_id=" + instance.ID,
		AuthorizationURL:   tokenServer.URL + "/authorize",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--data-dir", dataDir, "mcp", "auth", "complete", "expo", "http://localhost:3000/callback?code=secret-code&state=state-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "completed mcp auth for expo") {
		t.Fatalf("output = %q, want completion", output)
	}
	if strings.Contains(output, "secret-code") {
		t.Fatalf("output leaked code: %q", output)
	}

	s, err = store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer s.Close()
	accounts, err := s.ListMCPOAuthAccounts(instance.ID)
	if err != nil {
		t.Fatalf("ListMCPOAuthAccounts: %v", err)
	}
	if len(accounts) != 1 || accounts[0].AccessToken != "access-token" || accounts[0].AccountLabel != "expo-work" {
		t.Fatalf("accounts = %+v, want exchanged expo-work token", accounts)
	}
}

func TestMCPOAuthCompleteCommandRejectsMissingCode(t *testing.T) {
	cmd := NewRootCommand("test")
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"mcp", "auth", "complete", "expo", "http://localhost:3000/callback?state=state-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute error is nil")
	}
	if !strings.Contains(err.Error(), "code is required") {
		t.Fatalf("err = %v, want missing code", err)
	}
}

func TestMCPOAuthStartCommandStoresPKCEVerifier(t *testing.T) {
	dataDir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	instance := createCLIMCPInstance(t, s, "linear")
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--data-dir", dataDir,
		"mcp", "auth", "start", "linear",
		"--authorization-url", "https://auth.example/authorize",
		"--resource", "https://mcp.example",
		"--redirect-url", "http://localhost:3000/api/mcp/oauth/callback",
		"--client-id", "client-123",
		"--client-secret", "client-secret",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	output := strings.TrimSpace(out.String())
	if strings.Contains(output, "client-secret") {
		t.Fatalf("output leaked client secret: %q", output)
	}
	authURL, err := url.Parse(output)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	query := authURL.Query()
	if query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") == "" {
		t.Fatalf("query = %s, want S256 PKCE", authURL.RawQuery)
	}
	if query.Get("client_id") != "client-123" {
		t.Fatalf("client_id query = %q, want client-123", query.Get("client_id"))
	}
	redirect, err := url.Parse(query.Get("redirect_uri"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if decodedInstanceIDForCLITest(t, redirect.Query().Get("instance_id")) != instance.ID {
		t.Fatalf("redirect instance_id = %q, want recoverable instance ID", redirect.Query().Get("instance_id"))
	}

	s, err = store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer s.Close()
	flow, err := s.ConsumeMCPOAuthFlow(instance.ID, query.Get("state"))
	if err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow: %v", err)
	}
	if flow.CodeVerifierSecret == "" || flow.CodeVerifierSecret == query.Get("state") {
		t.Fatalf("verifier = %q state = %q, want separate verifier", flow.CodeVerifierSecret, query.Get("state"))
	}
	if flow.ClientID != "client-123" || flow.ClientSecret != "client-secret" {
		t.Fatal("stored client credentials did not match flag credentials")
	}
	if pkceChallengeForCLITest(flow.CodeVerifierSecret) != query.Get("code_challenge") {
		t.Fatalf("stored verifier does not match challenge")
	}
}

func TestMCPOAuthStartCommandDiscoversAuthorizationURL(t *testing.T) {
	dataDir := t.TempDir()
	s, err := store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	instance := createCLIMCPInstance(t, s, "linear")
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var authorizationEndpoint string
	resourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+cliTestServerURL(r)+`/.well-known/oauth-protected-resource"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"authorization_servers": []string{cliTestServerURL(r)}}); err != nil {
				t.Fatalf("Encode protected resource metadata: %v", err)
			}
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{"authorization_endpoint": authorizationEndpoint}); err != nil {
				t.Fatalf("Encode authorization metadata: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer resourceServer.Close()
	authorizationEndpoint = resourceServer.URL + "/oauth/authorize"

	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--data-dir", dataDir,
		"mcp", "auth", "start", "linear",
		"--resource", resourceServer.URL + "/mcp",
		"--redirect-url", "http://localhost:3000/api/mcp/oauth/callback",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	authURL, err := url.Parse(strings.TrimSpace(out.String()))
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if authURL.Scheme+"://"+authURL.Host+authURL.Path != authorizationEndpoint {
		t.Fatalf("authorization URL = %q, want discovered %q", out.String(), authorizationEndpoint)
	}
	s, err = store.NewStore(filepath.Join(dataDir, "g0router.db"))
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer s.Close()
	query := authURL.Query()
	if query.Get("resource") != resourceServer.URL+"/mcp" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("auth query = %s, want resource and S256 PKCE challenge", authURL.RawQuery)
	}
	if decodedInstanceIDForCLITest(t, mustParseURLForCLITest(t, query.Get("redirect_uri")).Query().Get("instance_id")) != instance.ID {
		t.Fatalf("redirect_uri = %q, want instance id", query.Get("redirect_uri"))
	}
	if _, err := s.ConsumeMCPOAuthFlow(instance.ID, query.Get("state")); err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow: %v", err)
	}
}

func createCLIMCPInstance(t *testing.T, s *store.Store, name string) *store.MCPInstance {
	t.Helper()
	instance := &store.MCPInstance{
		Name:       name,
		ServerKey:  name,
		LaunchType: "http",
		Transport:  "streamable-http",
		URL:        stringPtr("https://mcp.example/mcp"),
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	return instance
}

func stringPtr(value string) *string {
	return &value
}

func pkceChallengeForCLITest(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func decodedInstanceIDForCLITest(t *testing.T, value string) string {
	t.Helper()
	if strings.HasPrefix(value, "b64:") {
		decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "b64:"))
		if err != nil {
			t.Fatalf("decode instance id: %v", err)
		}
		return string(decoded)
	}
	return value
}

func cliTestServerURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func mustParseURLForCLITest(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return parsed
}
