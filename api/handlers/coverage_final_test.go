package handlers

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestCopyStringSliceNil(t *testing.T) {
	if got := copyStringSlice(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil slice = %+v, want empty non-nil", got)
	}
	got := copyStringSlice([]string{"a", "b"})
	if len(got) != 2 || got[0] != "a" {
		t.Fatalf("copy = %+v", got)
	}
}

// MCPTools GET aggregating registered MCP clients (covers the client branch of
// compactToolList plus the allowed-tools filter).
func TestMCPToolsListAggregatesClientManifests(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &store.MCPClient{
		Name:      "docs",
		Transport: mcp.TransportStdio,
		IsActive:  true,
	}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	manifest := mcp.Manifest{ClientID: client.ID, Tools: []mcp.Tool{
		{Name: "search", Description: "Search"},
		{Name: "fetch", Description: "Fetch"},
	}}
	if err := s.UpdateMCPClientManifest(client.ID, manifest); err != nil {
		t.Fatalf("UpdateMCPClientManifest: %v", err)
	}

	// No tool manager passed, so compactToolList walks the store clients branch.
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
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
		t.Fatalf("unmarshal: %v", err)
	}
	if len(listed.Data) != 2 {
		t.Fatalf("tools = %+v, want 2 client tools", listed.Data)
	}

	// allowed_tools filter narrows the aggregated client tools.
	names := []string{listed.Data[0].Function.Name, listed.Data[1].Function.Name}
	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools?allowed_tools="+names[0])
	MCPTools(ctx, s, nil, "")
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal filtered: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Function.Name != names[0] {
		t.Fatalf("filtered tools = %+v, want only %s", listed.Data, names[0])
	}
}

func TestMCPToolsListStoreErrorSanitized(t *testing.T) {
	s := openMCPHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, nil, "")
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
}

// OAuthPoll with a session whose stored provider mismatches the path flow.
func TestOAuthPollProviderMismatch(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:    "session-xyz",
		Provider: "openai",
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/poll?session_id=session-xyz")
	OAuthPoll(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// OAuthCallback where the session cannot be consumed (none stored) -> 500.
func TestOAuthCallbackConsumeFailure(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/callback?code=c&state=missing-state")
	OAuthCallback(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
}

// OAuthExchange where consuming the stored session fails -> 500.
func TestOAuthExchangeConsumeFailure(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"state":"missing-state","code":"c"}`)
	OAuthExchange(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
}

// getOAuthSession returns a non-not-found error when the store is closed.
func TestGetOAuthSessionStoreError(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := getOAuthSession(s, "state"); err == nil {
		t.Fatal("closed store should return error")
	}
}

// MCPOAuthStart success path persists a flow and returns the authorization URL.
func TestMCPOAuthStartPersistsFlow(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := createHandlerMCPInstance(t, s, "atlassian-a", "account-a")
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/"+instance.ID+"/auth/start")
	ctx.Request.SetBodyString(`{"authorization_url":"https://auth.example/authorize","resource_uri":"https://mcp.example","redirect_uri":"http://localhost:3000/cb","client_id":"client-123"}`)
	MCPOAuthStart(ctx, s, instance.ID)
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var resp mcpOAuthStartResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AuthorizationURL == "" || resp.ExpiresAt == "" {
		t.Fatalf("response = %+v, want authorization url + expiry", resp)
	}
}
