package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// --- Providers method / not-found / nil source ---

func TestProvidersMethodNotAllowed(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, handlerModelSource{}, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestProvidersUnknownProvider(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, handlerModelSource{}, "does-not-exist")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestProvidersNilSource(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, nil, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

// --- OAuthStart invalid JSON + session-store failure ---

func TestOAuthStartInvalidJSON(t *testing.T) {
	s := newHandlerStore(t)
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	ctx.Request.SetBodyString(`{`)
	OAuthStart(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestOAuthStartSessionStoreFailureSanitized(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/start")
	ctx.Request.SetBodyString(`{}`)
	OAuthStart(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
}

// --- createOAuthSession + persistOAuthConnection direct store-error paths ---

func TestCreateOAuthSessionStoreError(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	err := createOAuthSession(s, &oauth.AuthSession{
		Provider:  oauth.ProviderID("anthropic"),
		SessionID: "state.verifier",
		AuthURL:   "https://a/x?redirect_uri=https%3A%2F%2Fcb",
	}, "label")
	if err == nil {
		t.Fatal("closed store should return error")
	}
}

func TestPersistOAuthConnectionCreateError(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	_, err := persistOAuthConnection(s, oauth.TokenResult{
		Provider:    oauth.ProviderID("anthropic"),
		AccessToken: "tok",
	}, "label", "anthropic")
	if err == nil {
		t.Fatal("closed store should fail to create connection")
	}
}

// --- exchangeStoredOAuth flow-not-found branch ---

func TestExchangeStoredOAuthFlowNotFound(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:    "state-unknown-provider",
		Provider: "unregistered",
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	flows := OAuthFlows{} // no flows registered
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/callback?code=c&state=state-unknown-provider")
	OAuthCallback(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// --- OAuthPoll get-session store error ---

func TestOAuthPollGetSessionStoreErrorSanitized(t *testing.T) {
	s := newHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/poll?session_id=sess")
	OAuthPoll(ctx, s, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
}

// --- MCPOAuthComplete: runtime reapply failure marks unhealthy and returns 502 ---

type reapplyFailRuntime struct {
	reapplyErr error
}

func (r *reapplyFailRuntime) RegisterInstance(ctx context.Context, instance *store.MCPInstance) (mcp.Manifest, error) {
	return mcp.Manifest{}, nil
}

func (r *reapplyFailRuntime) CloseInstance(instanceID string) error { return nil }

func (r *reapplyFailRuntime) ReapplyInstanceCredentials(ctx context.Context, s MCPRuntimeStore, instanceID string) (mcp.Manifest, error) {
	return mcp.Manifest{}, r.reapplyErr
}

func TestMCPOAuthCompleteReapplyFailureMarksUnhealthy(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := createHandlerMCPInstance(t, s, "linear-b", "default")
	completer := &fakeMCPOAuthCompleter{}
	runtime := &reapplyFailRuntime{reapplyErr: errors.New("reapply boom")}

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/"+instance.ID+"/oauth/complete")
	ctx.Request.SetBodyString(`{"callback_url":"http://localhost:3000/callback?code=c&state=s"}`)
	MCPOAuthComplete(ctx, completer, runtime, s, instance.ID)

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	assertNoInternalDetail(t, ctx.Response.Body())
	stored, err := s.GetMCPInstance(instance.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if stored.HealthStatus != "unhealthy" {
		t.Fatalf("health = %q, want unhealthy", stored.HealthStatus)
	}
}

// --- MCPInstances POST inactive instance success (no runtime needed) ---

func TestMCPInstancesCreateInactiveSkipsRuntime(t *testing.T) {
	s := openMCPHandlerStore(t)
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances")
	ctx.Request.SetBodyString(`{"name":"i","server_key":"atlassian","launch_type":"http","transport":"streamable-http","url":"https://mcp.example/mcp","is_active":false}`)
	MCPInstances(ctx, s, nil, "")
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// --- MCPInstances POST active instance registration failure rolls back (502) ---

func TestMCPInstancesCreateActiveRuntimeFailure(t *testing.T) {
	s := openMCPHandlerStore(t)
	runtime := &fakeMCPInstanceRuntime{err: errors.New("offline")}
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances")
	ctx.Request.SetBodyString(`{"name":"i","server_key":"atlassian","launch_type":"http","transport":"streamable-http","url":"https://mcp.example/mcp","is_active":true}`)
	MCPInstances(ctx, s, runtime, "")
	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// --- apikeys policy update store-error paths ---

func TestAPIKeysUpdatePolicyStoreError(t *testing.T) {
	s := newHandlerStore(t)
	ctx, _ := runHandler(t, fasthttp.MethodPost, `{"name":"k"}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create failed: %d", ctx.Response.StatusCode())
	}
	var created struct {
		Key struct{ ID string `json:"id"` } `json:"key"`
	}
	decodeJSON(t, ctx.Response.Body(), &created)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx, body := runHandler(t, fasthttp.MethodPut, `{"rate_limit_rpm":10}`, func(ctx *fasthttp.RequestCtx) {
		APIKeys(ctx, s, "test-secret", created.Key.ID)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), body)
	}
}
