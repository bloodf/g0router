package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeMCPConnector struct {
	client *fakeMCPClient
	err    error
}

func (f *fakeMCPConnector) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.client.config = cfg
	return f.client, nil
}

type fakeMCPClient struct {
	config mcp.ClientConfig
	tools  []mcp.Tool
	calls  []mcp.CallRequest
	err    error
}

func (f *fakeMCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return f.tools, f.err
}

func (f *fakeMCPClient) CallTool(ctx context.Context, req mcp.CallRequest) (mcp.CallResult, error) {
	f.calls = append(f.calls, req)
	return mcp.CallResult{Content: map[string]any{"ok": true}}, nil
}

func (f *fakeMCPClient) Close() error {
	return nil
}

func TestMCPClientsCreateDiscoversAndPersistsManifest(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &fakeMCPClient{tools: []mcp.Tool{{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	manager := mcp.NewClientManager(&fakeMCPConnector{client: client})
	tools := mcp.NewToolManager()

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	ctx.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","args":["--stdio"],"env":{"TOKEN":"secret"},"is_active":true}`)
	MCPClients(ctx, s, manager, tools, "")

	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var created store.MCPClient
	if err := json.Unmarshal(ctx.Response.Body(), &created); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if created.ID == "" || created.Name != "docs" {
		t.Fatalf("created = %+v, want docs with ID", created)
	}
	if client.config.ID != created.ID || client.config.Command != "mcp-docs" {
		t.Fatalf("config = %+v, want created ID and command", client.config)
	}

	got, err := s.GetMCPClient(created.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	if got.ToolManifest == nil || len(got.ToolManifest.Tools) != 1 {
		t.Fatalf("stored manifest = %+v, want one tool", got.ToolManifest)
	}

	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/mcp/clients")
	MCPClients(ctx, s, manager, tools, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var listed struct {
		Data []store.MCPClient `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].ID != created.ID {
		t.Fatalf("listed = %+v, want created client", listed.Data)
	}
}

func TestMCPClientsCreateRollsBackStoreWhenDiscoveryFails(t *testing.T) {
	s := openMCPHandlerStore(t)
	manager := mcp.NewClientManager(&fakeMCPConnector{client: &fakeMCPClient{err: errors.New("offline")}})
	tools := mcp.NewToolManager()

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	ctx.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`)
	MCPClients(ctx, s, manager, tools, "")

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	clients, err := s.ListMCPClients()
	if err != nil {
		t.Fatalf("ListMCPClients: %v", err)
	}
	if len(clients) != 0 {
		t.Fatalf("clients = %+v, want rollback", clients)
	}
}

func TestMCPToolsListAndExecute(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &fakeMCPClient{tools: []mcp.Tool{{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	manager := mcp.NewClientManager(&fakeMCPConnector{client: client})
	tools := mcp.NewToolManager()

	createCtx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	createCtx.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`)
	MCPClients(createCtx, s, manager, tools, "")
	if createCtx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", createCtx.Response.StatusCode(), createCtx.Response.Body())
	}
	var created store.MCPClient
	if err := json.Unmarshal(createCtx.Response.Body(), &created); err != nil {
		t.Fatalf("unmarshal created client: %v", err)
	}
	toolName := created.ID + "__search"

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, tools, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var listed struct {
		Data []struct {
			Type     string `json:"type"`
			Function struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters,omitempty"`
			} `json:"function"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal tools: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Function.Name != toolName {
		t.Fatalf("tools = %+v, want %s", listed.Data, toolName)
	}
	if listed.Data[0].Function.Parameters != nil {
		t.Fatalf("compact tool includes parameters: %s", listed.Data[0].Function.Parameters)
	}

	ctx = newHandlerCtx(fasthttp.MethodPost, "/api/mcp/tools/"+toolName+"/execute")
	ctx.Request.SetBodyString(`{"arguments":{"query":"mcp"}}`)
	MCPTools(ctx, s, tools, toolName)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("execute status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if len(client.calls) != 1 || client.calls[0].Name != "search" || string(client.calls[0].Arguments) != `{"query":"mcp"}` {
		t.Fatalf("calls = %+v, want search with args", client.calls)
	}
}

func TestMCPToolsExecuteMissingTool(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/tools/missing/execute")
	ctx.Request.SetBodyString(`{"arguments":{}}`)

	MCPTools(ctx, openMCPHandlerStore(t), mcp.NewToolManager(), "missing")

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestMCPInstancesCreateListRedactsSecretsAndStartsAuth(t *testing.T) {
	s := openMCPHandlerStore(t)

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances")
	ctx.Request.SetBodyString(`{"name":"atlassian-a","server_key":"atlassian","launch_type":"http","transport":"streamable-http","url":"https://mcp.atlassian.com/mcp","headers":{"Authorization":"Bearer secret"},"env":{"API_TOKEN":"secret"},"account_label":"account-a","is_active":true}`)
	MCPInstances(ctx, s, "")

	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var created store.MCPInstance
	if err := json.Unmarshal(ctx.Response.Body(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}
	if created.Name != "atlassian-a" || created.AccountLabel == nil || *created.AccountLabel != "account-a" {
		t.Fatalf("created = %+v, want account-a instance", created)
	}

	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/mcp/instances")
	MCPInstances(ctx, s, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("list leaked secret: %s", ctx.Response.Body())
	}

	ctx = newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/"+created.ID+"/auth/start")
	ctx.Request.SetBodyString(`{"authorization_url":"https://auth.example/authorize","resource_uri":"https://mcp.atlassian.com","redirect_uri":"http://localhost:3000/api/mcp/oauth/callback"}`)
	MCPOAuthStart(ctx, s, created.ID)
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("auth status = %d, want 201; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if strings.Contains(string(ctx.Response.Body()), "secret") {
		t.Fatalf("auth response leaked secret: %s", ctx.Response.Body())
	}

	var started struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &started); err != nil {
		t.Fatalf("unmarshal auth start: %v", err)
	}
	authURL, err := url.Parse(started.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	query := authURL.Query()
	if query.Get("resource") != "https://mcp.atlassian.com" || query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") == "" {
		t.Fatalf("auth query = %s, want resource and S256 PKCE challenge", authURL.RawQuery)
	}
	redirect, err := url.Parse(query.Get("redirect_uri"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if decodedInstanceIDForHandlerTest(t, redirect.Query().Get("instance_id")) != created.ID {
		t.Fatalf("redirect instance_id = %q, want recoverable created ID", redirect.Query().Get("instance_id"))
	}
	flow, err := s.ConsumeMCPOAuthFlow(created.ID, query.Get("state"))
	if err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow: %v", err)
	}
	if flow.CodeVerifierSecret == "" || flow.CodeVerifierSecret == query.Get("state") {
		t.Fatalf("verifier = %q state = %q, want separate verifier", flow.CodeVerifierSecret, query.Get("state"))
	}
	if pkceChallengeForHandlerTest(flow.CodeVerifierSecret) != query.Get("code_challenge") {
		t.Fatalf("stored verifier does not match code challenge")
	}
}

func TestMCPInstanceAccountsRedactTokens(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := createHandlerMCPInstance(t, s, "atlassian-a", "account-a")
	if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   instance.ID,
		AccountLabel: "account-a",
		ResourceURI:  "https://mcp.atlassian.com",
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		ExpiresAt:    time.Now().Add(time.Hour),
		AuthMetadata: map[string]string{"token_endpoint": "https://auth.example/token"},
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/instances/"+instance.ID+"/accounts")
	MCPOAuthAccounts(ctx, s, instance.ID)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "account-a") {
		t.Fatalf("body = %s, want account label", body)
	}
	if strings.Contains(body, "access-secret") || strings.Contains(body, "refresh-secret") {
		t.Fatalf("account response leaked token: %s", body)
	}
}

func TestMCPInstanceToolsFilterByInstanceAndAccountWithStableDuplicateNames(t *testing.T) {
	s := openMCPHandlerStore(t)
	first := createHandlerMCPInstance(t, s, "atlassian-a", "account-a")
	second := createHandlerMCPInstance(t, s, "atlassian-b", "account-b")
	manifest := func(id string) mcp.Manifest {
		return mcp.Manifest{ClientID: id, Tools: []mcp.Tool{{Name: "search", Description: "Search"}}}
	}
	if err := s.UpdateMCPInstanceManifest(first.ID, manifest(first.ID)); err != nil {
		t.Fatalf("UpdateMCPInstanceManifest first: %v", err)
	}
	if err := s.UpdateMCPInstanceManifest(second.ID, manifest(second.ID)); err != nil {
		t.Fatalf("UpdateMCPInstanceManifest second: %v", err)
	}

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools?instance_id="+first.ID+"&account_label=account-a")
	MCPTools(ctx, s, nil, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var listed struct {
		Data []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal tools: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Function.Name != first.ID+"__search" {
		t.Fatalf("tools = %+v, want first stable tool name", listed.Data)
	}

	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, nil, "")
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal all tools: %v", err)
	}
	if len(listed.Data) != 2 || listed.Data[0].Function.Name == listed.Data[1].Function.Name {
		t.Fatalf("tools = %+v, want two distinct stable names", listed.Data)
	}
}

func TestMCPInstancesDeleteRemovesAccountsAndCachedToolsOnlyForOneInstance(t *testing.T) {
	s := openMCPHandlerStore(t)
	first := createHandlerMCPInstance(t, s, "atlassian-a", "account-a")
	second := createHandlerMCPInstance(t, s, "atlassian-b", "account-b")
	for _, instance := range []*store.MCPInstance{first, second} {
		if err := s.UpdateMCPInstanceManifest(instance.ID, mcp.Manifest{ClientID: instance.ID, Tools: []mcp.Tool{{Name: "search", Description: "Search"}}}); err != nil {
			t.Fatalf("UpdateMCPInstanceManifest: %v", err)
		}
		if err := s.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{InstanceID: instance.ID, AccountLabel: *instance.AccountLabel, AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatalf("UpsertMCPOAuthAccount: %v", err)
		}
	}

	ctx := newHandlerCtx(fasthttp.MethodDelete, "/api/mcp/instances/"+first.ID)
	MCPInstances(ctx, s, first.ID)
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if accounts, err := s.ListMCPOAuthAccounts(first.ID); err != nil || len(accounts) != 0 {
		t.Fatalf("first accounts = %+v err=%v, want none", accounts, err)
	}
	if accounts, err := s.ListMCPOAuthAccounts(second.ID); err != nil || len(accounts) != 1 {
		t.Fatalf("second accounts = %+v err=%v, want one", accounts, err)
	}

	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, nil, "")
	if strings.Contains(string(ctx.Response.Body()), first.ID+"__search") || !strings.Contains(string(ctx.Response.Body()), second.ID+"__search") {
		t.Fatalf("tools after delete = %s, want only sibling tools", ctx.Response.Body())
	}
}

func createHandlerMCPInstance(t *testing.T, s *store.Store, name, accountLabel string) *store.MCPInstance {
	t.Helper()
	instance := &store.MCPInstance{
		Name:         name,
		ServerKey:    "atlassian",
		LaunchType:   mcp.LaunchHTTP,
		Transport:    mcp.TransportStreamableHTTP,
		URL:          stringPtr(accountLabelURL()),
		AccountLabel: stringPtr(accountLabel),
		IsActive:     true,
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	return instance
}

func accountLabelURL() string {
	return "https://mcp.atlassian.com/mcp"
}

func pkceChallengeForHandlerTest(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func decodedInstanceIDForHandlerTest(t *testing.T, value string) string {
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

func openMCPHandlerStore(t *testing.T) *store.Store {
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
