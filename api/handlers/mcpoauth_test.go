package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/valyala/fasthttp"
)

func TestMCPOAuthCallbackCompletesPendingFlow(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/oauth/callback?instance_id=inst-1&code=callback-code&state=state-1")

	MCPOAuthCallback(ctx, completer)

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

	MCPOAuthComplete(ctx, completer, "inst-1")

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

func TestMCPOAuthCompleteRejectsMissingCode(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?state=state-1"}`)

	MCPOAuthComplete(ctx, completer, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestMCPOAuthCompleteRejectsMismatchedState(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{err: mcp.ErrOAuthFlowNotFound}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/inst-1/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?code=ok&state=wrong"}`)

	MCPOAuthComplete(ctx, completer, "inst-1")

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
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
