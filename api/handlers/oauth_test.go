package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/valyala/fasthttp"
)

type fakeOAuthFlow struct {
	provider oauth.ProviderID
	startErr error
	pollErr  error
	exErr    error

	started       bool
	polledSession oauth.AuthSession
	exSession     oauth.AuthSession
	exCode        string
}

func (f *fakeOAuthFlow) ProviderID() oauth.ProviderID {
	return f.provider
}

func (f *fakeOAuthFlow) Start(ctx context.Context) (oauth.AuthSession, error) {
	f.started = true
	if f.startErr != nil {
		return oauth.AuthSession{}, f.startErr
	}
	return oauth.AuthSession{
		Provider:     f.provider,
		AuthURL:      "https://auth.example/start",
		SessionID:    "session-123",
		UserCode:     "ABCD-EFGH",
		Verification: "https://auth.example/device",
		ExpiresIn:    600,
		PollInterval: 5,
	}, nil
}

func (f *fakeOAuthFlow) Exchange(ctx context.Context, session oauth.AuthSession, code string) (oauth.TokenResult, error) {
	f.exSession = session
	f.exCode = code
	if f.exErr != nil {
		return oauth.TokenResult{}, f.exErr
	}
	return oauth.TokenResult{
		Provider:     f.provider,
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "bearer",
		Scopes:       []string{"scope-a", "scope-b"},
	}, nil
}

func (f *fakeOAuthFlow) Poll(ctx context.Context, session oauth.AuthSession) (oauth.PollResult, error) {
	f.polledSession = session
	if f.pollErr != nil {
		return oauth.PollResult{}, f.pollErr
	}
	return oauth.PollResult{
		Status: oauth.PollStatusComplete,
		Token: &oauth.TokenResult{
			Provider:    f.provider,
			AccessToken: "access-token",
			TokenType:   "bearer",
		},
	}, nil
}

func TestOAuthStartReturnsSession(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("codex")}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/codex/authorize", nil)

	OAuthStart(ctx, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !flow.started {
		t.Fatal("flow was not started")
	}
	var session oauth.AuthSession
	decodeOAuthBody(t, ctx, &session)
	if session.Provider != flow.ProviderID() {
		t.Fatalf("provider = %q, want %q", session.Provider, flow.ProviderID())
	}
	if session.SessionID != "session-123" {
		t.Fatalf("session id = %q, want session-123", session.SessionID)
	}
}

func TestOAuthStartRejectsUnknownProvider(t *testing.T) {
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/missing/authorize", nil)

	OAuthStart(ctx, OAuthFlows{})

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestOAuthPollUsesSessionFromQuery(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("github-copilot")}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/github-copilot/poll?session_id=device-123", nil)

	OAuthPoll(ctx, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.polledSession.Provider != flow.ProviderID() {
		t.Fatalf("polled provider = %q, want %q", flow.polledSession.Provider, flow.ProviderID())
	}
	if flow.polledSession.SessionID != "device-123" {
		t.Fatalf("polled session = %q, want device-123", flow.polledSession.SessionID)
	}
	var result oauth.PollResult
	decodeOAuthBody(t, ctx, &result)
	if result.Status != oauth.PollStatusComplete {
		t.Fatalf("status = %q, want complete", result.Status)
	}
}

func TestOAuthCallbackExchangesQueryCode(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/callback?provider=anthropic&session_id=state.verifier&code=callback-code", nil)

	OAuthCallback(ctx, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.exSession.Provider != flow.ProviderID() {
		t.Fatalf("exchange provider = %q, want %q", flow.exSession.Provider, flow.ProviderID())
	}
	if flow.exSession.SessionID != "state.verifier" {
		t.Fatalf("exchange session = %q, want state.verifier", flow.exSession.SessionID)
	}
	if flow.exCode != "callback-code" {
		t.Fatalf("exchange code = %q, want callback-code", flow.exCode)
	}
	var token oauth.TokenResult
	decodeOAuthBody(t, ctx, &token)
	if token.AccessToken != "access-token" {
		t.Fatalf("access token = %q, want access-token", token.AccessToken)
	}
}

func TestOAuthExchangeUsesJSONBody(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("xai")}
	body := []byte(`{"session":{"provider":"xai","session_id":"state.verifier"},"code":"manual-code"}`)
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/xai/exchange", body)

	OAuthExchange(ctx, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.exSession.Provider != flow.ProviderID() {
		t.Fatalf("exchange provider = %q, want %q", flow.exSession.Provider, flow.ProviderID())
	}
	if flow.exCode != "manual-code" {
		t.Fatalf("exchange code = %q, want manual-code", flow.exCode)
	}
}

func TestOAuthExchangeRejectsInvalidJSON(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("xai")}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/xai/exchange", []byte(`{"session":`))

	OAuthExchange(ctx, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestOAuthHandlersReturnFlowErrors(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("codex"), pollErr: errors.New("poll failed")}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/codex/poll?session_id=device-123", nil)

	OAuthPoll(ctx, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !strings.Contains(string(ctx.Response.Body()), "poll failed") {
		t.Fatalf("body = %s, want poll error", ctx.Response.Body())
	}
}

func oauthRequestCtx(t *testing.T, method string, uri string, body []byte) *fasthttp.RequestCtx {
	t.Helper()

	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)
	if body != nil {
		req.Header.SetContentType("application/json")
		req.SetBody(body)
	}

	var ctx fasthttp.RequestCtx
	ctx.Init(&req, nil, nil)
	return &ctx
}

func decodeOAuthBody(t *testing.T, ctx *fasthttp.RequestCtx, target any) {
	t.Helper()

	if err := json.Unmarshal(ctx.Response.Body(), target); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, ctx.Response.Body())
	}
}
