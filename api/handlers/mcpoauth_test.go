package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/valyala/fasthttp"
)

func TestMCPOAuthCallbackCompletesPendingFlow(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/oauth/callback?instance_id=inst-1&code=callback-code&state=state-1")

	MCPOAuthCallback(ctx, completer, nil, nil)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if completer.instanceID != "inst-1" {
		t.Fatalf("instance = %q, want inst-1", completer.instanceID)
	}
	if !strings.Contains(completer.callbackURL, "code=callback-code") {
		t.Fatalf("callbackURL = %q, want code", completer.callbackURL)
	}
	if strings.Contains(string(ctx.Response.Body()), "callback-code") {
		t.Fatalf("response leaked code: %s", ctx.Response.Body())
	}
}

func TestMCPOAuthCompleteAcceptsPastedCallbackURL(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?code=pasted-code&state=state-1"}`)

	MCPOAuthComplete(ctx, completer, nil, nil, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if completer.instanceID != "inst-1" {
		t.Fatalf("instance = %q, want inst-1", completer.instanceID)
	}
	if !strings.Contains(completer.callbackURL, "pasted-code") {
		t.Fatalf("callbackURL = %q, want pasted code", completer.callbackURL)
	}
	if strings.Contains(string(ctx.Response.Body()), "pasted-code") {
		t.Fatalf("response leaked code: %s", ctx.Response.Body())
	}
}

func TestMCPOAuthCompleteReappliesLiveInstanceCredentials(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := createHandlerMCPInstance(t, s, "linear-a", "default")
	completer := &fakeMCPOAuthCompleter{}
	runtime := &fakeMCPInstanceRuntime{manifest: mcp.Manifest{ClientID: instance.ID, Tools: []mcp.Tool{{Name: "search", Description: "Search"}}}}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/"+instance.ID+"/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?code=pasted-code&state=state-1"}`)

	MCPOAuthComplete(ctx, completer, runtime, s, instance.ID)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if len(runtime.reapplied) != 1 || runtime.reapplied[0] != instance.ID {
		t.Fatalf("reapplied = %+v, want instance", runtime.reapplied)
	}
	stored, err := s.GetMCPInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if stored.HealthStatus != "healthy" {
		t.Fatalf("health = %q, want healthy", stored.HealthStatus)
	}
	if stored.ToolManifest == nil || len(stored.ToolManifest.Tools) != 1 {
		t.Fatalf("stored manifest = %+v, want one tool", stored.ToolManifest)
	}
	var response struct {
		InstanceID string `json:"instance_id"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.InstanceID != instance.ID {
		t.Fatalf("response instance = %q, want %q", response.InstanceID, instance.ID)
	}
}

func TestMCPOAuthCallbackDecodesInstanceID(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	encoded := "b64:" + base64.RawURLEncoding.EncodeToString([]byte("inst-1"))
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/oauth/callback?instance_id="+encoded+"&code=callback-code&state=state-1")

	MCPOAuthCallback(ctx, completer, nil, nil)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if completer.instanceID != "inst-1" {
		t.Fatalf("instance = %q, want decoded inst-1", completer.instanceID)
	}
}

func TestMCPOAuthCompleteRejectsMissingCode(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?state=state-1"}`)

	MCPOAuthComplete(ctx, completer, nil, nil, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestMCPOAuthCompleteRejectsMismatchedState(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{err: mcp.ErrOAuthFlowNotFound}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?code=ok&state=wrong"}`)

	MCPOAuthComplete(ctx, completer, nil, nil, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestMCPOAuthCompleteSanitizesCompletionErrors(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{err: errors.New(`token endpoint returned 400 {"access_token":"leaked-access","refresh_token":"leaked-refresh","code":"pasted-code"}`)}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?code=pasted-code&state=state-1"}`)

	MCPOAuthComplete(ctx, completer, nil, nil, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "mcp oauth exchange failed") {
		t.Fatalf("body = %s, want sanitized exchange error", body)
	}
	for _, secret := range []string{"leaked-access", "leaked-refresh", "pasted-code", "access_token", "refresh_token"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked %q: %s", secret, body)
		}
	}
}

func TestMCPOAuthCompleteRejectsNonHTTPScheme(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"javascript:alert(1)"}`)

	MCPOAuthComplete(ctx, completer, nil, nil, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestMCPOAuthCompleteAcceptsHTTPSScheme(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"https://app.example.com/callback?code=c1&state=s1"}`)

	MCPOAuthComplete(ctx, completer, nil, nil, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

type fakeMCPOAuthCompleter struct {
	instanceID  string
	callbackURL string
	err         error
}

func (f *fakeMCPOAuthCompleter) CompleteCallback(ctx context.Context, instanceID, callbackURL string) (mcp.OAuthAccount, error) {
	f.instanceID = instanceID
	f.callbackURL = callbackURL
	if f.err != nil {
		return mcp.OAuthAccount{}, f.err
	}
	return mcp.OAuthAccount{InstanceID: instanceID, AccountLabel: "default", AccessToken: "token"}, nil
}
