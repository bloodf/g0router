package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeOAuthFlow struct {
	provider     oauth.ProviderID
	startSession oauth.AuthSession
	token        oauth.TokenResult
	pollResult   oauth.PollResult
	startErr     error
	pollErr      error
	exErr        error

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
	if f.startSession.Provider != "" {
		return f.startSession, nil
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
	if f.token.Provider != "" {
		return f.token, nil
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
	if f.pollResult.Status != "" {
		return f.pollResult, nil
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

	OAuthStart(ctx, openHandlerTestStore(t), OAuthFlows{flow.ProviderID(): flow})

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

	OAuthStart(ctx, openHandlerTestStore(t), OAuthFlows{})

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestOAuthPollUsesSessionFromQuery(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("github-copilot")}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/github-copilot/poll?session_id=device-123", nil)

	OAuthPoll(ctx, openHandlerTestStore(t), OAuthFlows{flow.ProviderID(): flow})

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
	if strings.Contains(string(ctx.Response.Body()), "access-token") {
		t.Fatalf("poll response leaked token material: %s", ctx.Response.Body())
	}
}

func TestOAuthPollAcceptsGitHubAlias(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("github-copilot")}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/github/poll?session_id=device-123", nil)

	OAuthPoll(ctx, openHandlerTestStore(t), OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.polledSession.Provider != oauth.ProviderID("github-copilot") {
		t.Fatalf("polled provider = %q, want github-copilot", flow.polledSession.Provider)
	}
}

func TestOAuthPollUsesStoredVerifierAndAccountLabel(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("cursor")}
	flow.startSession = oauth.AuthSession{
		Provider:  flow.ProviderID(),
		AuthURL:   "https://cursor.example/loginDeepControl?uuid=cursor-state",
		SessionID: "cursor-state.cursor-verifier",
	}
	flow.pollResult = oauth.PollResult{
		Status: oauth.PollStatusComplete,
		Token: &oauth.TokenResult{
			Provider:     flow.ProviderID(),
			AccessToken:  "cursor-access-token",
			RefreshToken: "cursor-refresh-token",
			TokenType:    "Bearer",
		},
	}
	s := openHandlerTestStore(t)
	startCtx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/cursor/authorize", []byte(`{"account_label":"cursor-work"}`))

	OAuthStart(startCtx, s, OAuthFlows{flow.ProviderID(): flow})

	if startCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("start status = %d, want 200; body=%s", startCtx.Response.StatusCode(), startCtx.Response.Body())
	}
	var session oauth.AuthSession
	decodeOAuthBody(t, startCtx, &session)
	if session.SessionID != "cursor-state" {
		t.Fatalf("public session id = %q, want cursor-state", session.SessionID)
	}
	if strings.Contains(string(startCtx.Response.Body()), "cursor-verifier") {
		t.Fatalf("response leaked verifier: %s", startCtx.Response.Body())
	}

	pollCtx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/cursor/poll?session_id=cursor-state", nil)
	OAuthPoll(pollCtx, s, OAuthFlows{flow.ProviderID(): flow})

	if pollCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("poll status = %d, want 200; body=%s", pollCtx.Response.StatusCode(), pollCtx.Response.Body())
	}
	if flow.polledSession.SessionID != "cursor-state.cursor-verifier" {
		t.Fatalf("polled session id = %q, want stored verifier restored", flow.polledSession.SessionID)
	}
	if strings.Contains(string(pollCtx.Response.Body()), "cursor-access-token") || strings.Contains(string(pollCtx.Response.Body()), "cursor-refresh-token") {
		t.Fatalf("poll response leaked token material: %s", pollCtx.Response.Body())
	}
	connections, err := s.GetConnections("cursor")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 || connections[0].Name != "cursor-work" {
		t.Fatalf("connections = %+v, want cursor-work connection", connections)
	}
	if _, err := s.ConsumeOAuthSession("cursor-state"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("session should be consumed after complete poll, err = %v", err)
	}
}

func TestOAuthStartStoresCallbackSessionWithoutVerifierLeak(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flowSessionID := "state-123.verifier-secret"
	flowAuthURL := "https://auth.example/start?redirect_uri=http%3A%2F%2Flocalhost%2Foauth%2Fcallback&state=state-123"
	flow.started = false
	flow.startSession = oauth.AuthSession{Provider: flow.ProviderID(), AuthURL: flowAuthURL, SessionID: flowSessionID}
	s := openHandlerTestStore(t)
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/anthropic/authorize", []byte(`{"account_label":"work"}`))

	OAuthStart(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var session oauth.AuthSession
	decodeOAuthBody(t, ctx, &session)
	if session.SessionID != "state-123" {
		t.Fatalf("session id = %q, want public state only", session.SessionID)
	}
	if strings.Contains(string(ctx.Response.Body()), "verifier-secret") {
		t.Fatalf("response leaked verifier: %s", ctx.Response.Body())
	}

	stored, err := s.ConsumeOAuthSession("state-123")
	if err != nil {
		t.Fatalf("ConsumeOAuthSession: %v", err)
	}
	if stored.Provider != "anthropic" || stored.CodeVerifier != "verifier-secret" || stored.AccountLabel != "work" {
		t.Fatalf("stored session = %+v, want provider/verifier/account label", stored)
	}
	if stored.RedirectURI != "http://localhost/oauth/callback" {
		t.Fatalf("redirect uri = %q, want stored redirect", stored.RedirectURI)
	}
}

func TestOAuthCallbackUsesStoredVerifierAndPersistsConnection(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        "callback-state",
		Provider:     "anthropic",
		CodeVerifier: "stored-verifier",
		RedirectURI:  "http://localhost/oauth/callback",
		AccountLabel: "work",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/callback?state=callback-state&code=callback-code", nil)

	OAuthCallback(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.exSession.Provider != flow.ProviderID() {
		t.Fatalf("exchange provider = %q, want %q", flow.exSession.Provider, flow.ProviderID())
	}
	if flow.exSession.SessionID != "callback-state.stored-verifier" {
		t.Fatalf("exchange session = %q, want stored verifier session", flow.exSession.SessionID)
	}
	if flow.exCode != "callback-code" {
		t.Fatalf("exchange code = %q, want callback-code", flow.exCode)
	}
	if strings.Contains(string(ctx.Response.Body()), "access-token") || strings.Contains(string(ctx.Response.Body()), "refresh-token") {
		t.Fatalf("response leaked token material: %s", ctx.Response.Body())
	}
	connections, err := s.GetConnections("anthropic")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("connections = %d, want 1", len(connections))
	}
	if connections[0].AccessToken == nil || *connections[0].AccessToken != "access-token" {
		t.Fatalf("stored access token = %v, want access-token", connections[0].AccessToken)
	}
	if connections[0].Name != "work" {
		t.Fatalf("connection name = %q, want account label", connections[0].Name)
	}
}

func TestOAuthCallbackDoesNotLeakExchangeErrorSecrets(t *testing.T) {
	flow := &fakeOAuthFlow{
		provider: oauth.ProviderID("anthropic"),
		exErr:    errors.New(`token endpoint returned 400 {"access_token":"leaked-access","refresh_token":"leaked-refresh","Authorization":"Bearer leaked-auth","code":"callback-code"}`),
	}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        "callback-state",
		Provider:     "anthropic",
		CodeVerifier: "stored-verifier",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/callback?state=callback-state&code=callback-code", nil)

	OAuthCallback(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	for _, secret := range []string{"leaked-access", "leaked-refresh", "leaked-auth", "callback-code", "access_token", "refresh_token", "Authorization"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked %q in body: %s", secret, body)
		}
	}
	if !strings.Contains(body, "oauth exchange failed") {
		t.Fatalf("body = %s, want sanitized exchange failure", body)
	}
}

func TestOAuthExchangeUsesJSONBody(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("xai")}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        "state",
		Provider:     "xai",
		CodeVerifier: "verifier",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	body := []byte(`{"state":"state","code":"manual-code"}`)
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/xai/exchange", body)

	OAuthExchange(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.exSession.Provider != flow.ProviderID() {
		t.Fatalf("exchange provider = %q, want %q", flow.exSession.Provider, flow.ProviderID())
	}
	if flow.exCode != "manual-code" {
		t.Fatalf("exchange code = %q, want manual-code", flow.exCode)
	}
	connections, err := s.GetConnections("xai")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("connections = %d, want 1", len(connections))
	}
}

func TestOAuthExchangeStoresCodexTokenAsOpenAIConnection(t *testing.T) {
	flow := &fakeOAuthFlow{
		provider: oauth.ProviderID("codex"),
		token: oauth.TokenResult{
			Provider:     oauth.ProviderID("codex"),
			AccessToken:  "codex-access",
			RefreshToken: "codex-refresh",
			TokenType:    "bearer",
		},
	}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        "codex-state",
		Provider:     "codex",
		CodeVerifier: "verifier",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/codex/exchange", []byte(`{"state":"codex-state","code":"manual-code"}`))

	OAuthExchange(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	openAIConnections, err := s.GetConnections("openai")
	if err != nil {
		t.Fatalf("GetConnections openai: %v", err)
	}
	if len(openAIConnections) != 1 {
		t.Fatalf("openai connections = %d, want 1", len(openAIConnections))
	}
	codexConnections, err := s.GetConnections("codex")
	if err != nil {
		t.Fatalf("GetConnections codex: %v", err)
	}
	if len(codexConnections) != 0 {
		t.Fatalf("codex connections = %d, want 0", len(codexConnections))
	}
	if openAIConnections[0].ProviderSpecificData["oauth_provider"] != "codex" {
		t.Fatalf("provider data = %+v, want oauth_provider codex", openAIConnections[0].ProviderSpecificData)
	}
}

func TestOAuthExchangeAcceptsOpenAIAliasForCodexFlow(t *testing.T) {
	flow := &fakeOAuthFlow{
		provider: oauth.ProviderID("codex"),
		token: oauth.TokenResult{
			Provider:    oauth.ProviderID("codex"),
			AccessToken: "codex-access",
			TokenType:   "bearer",
		},
	}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        "codex-state",
		Provider:     "codex",
		CodeVerifier: "verifier",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/openai/exchange", []byte(`{"state":"codex-state","code":"manual-code"}`))

	OAuthExchange(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	openAIConnections, err := s.GetConnections("openai")
	if err != nil {
		t.Fatalf("GetConnections openai: %v", err)
	}
	if len(openAIConnections) != 1 {
		t.Fatalf("openai connections = %d, want 1", len(openAIConnections))
	}
}

func TestOAuthExchangeStoresVertexTokenAsVertexConnection(t *testing.T) {
	flow := &fakeOAuthFlow{
		provider: oauth.ProviderID("gemini"),
		token: oauth.TokenResult{
			Provider:     oauth.ProviderID("gemini"),
			AccessToken:  "vertex-access",
			RefreshToken: "vertex-refresh",
			TokenType:    "bearer",
		},
	}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:        "vertex-state",
		Provider:     "vertex",
		CodeVerifier: "verifier",
		AccountLabel: "gcp-work",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/vertex/exchange", []byte(`{"state":"vertex-state","code":"manual-code"}`))

	OAuthExchange(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if flow.exSession.Provider != flow.ProviderID() {
		t.Fatalf("exchange provider = %q, want gemini", flow.exSession.Provider)
	}
	vertexConnections, err := s.GetConnections("vertex")
	if err != nil {
		t.Fatalf("GetConnections vertex: %v", err)
	}
	if len(vertexConnections) != 1 {
		t.Fatalf("vertex connections = %d, want 1", len(vertexConnections))
	}
	if vertexConnections[0].Name != "gcp-work" {
		t.Fatalf("connection name = %q, want account label", vertexConnections[0].Name)
	}
	if vertexConnections[0].ProviderSpecificData["oauth_provider"] != "gemini" {
		t.Fatalf("provider data = %+v, want oauth_provider gemini", vertexConnections[0].ProviderSpecificData)
	}
	geminiConnections, err := s.GetConnections("gemini")
	if err != nil {
		t.Fatalf("GetConnections gemini: %v", err)
	}
	if len(geminiConnections) != 0 {
		t.Fatalf("gemini connections = %d, want 0", len(geminiConnections))
	}
}

func TestOAuthExchangePersistsAPIKeyFlowAsAPIKeyConnection(t *testing.T) {
	flow := &fakeOAuthFlow{
		provider: oauth.ProviderID("minimax"),
		token: oauth.TokenResult{
			Provider:    oauth.ProviderID("minimax"),
			AccessToken: "minimax-key",
			TokenType:   "api_key",
		},
	}
	s := openHandlerTestStore(t)
	if err := s.CreateOAuthSession(&store.OAuthSession{
		State:     "minimax-state",
		Provider:  "minimax",
		ExpiresAt: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/minimax/exchange", []byte(`{"state":"minimax-state","code":"minimax-key"}`))

	OAuthExchange(ctx, s, OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	connections, err := s.GetConnections("minimax")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("connections = %d, want 1", len(connections))
	}
	if connections[0].AuthType != store.AuthTypeAPIKey {
		t.Fatalf("auth type = %q, want api_key", connections[0].AuthType)
	}
	if connections[0].APIKey == nil || *connections[0].APIKey != "minimax-key" {
		t.Fatalf("api key = %v, want minimax-key", connections[0].APIKey)
	}
	if connections[0].AccessToken != nil {
		t.Fatalf("access token = %v, want nil for api_key flow", connections[0].AccessToken)
	}
}

func TestOAuthExchangeRejectsInvalidJSON(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("xai")}
	ctx := oauthRequestCtx(t, fasthttp.MethodPost, "/api/oauth/xai/exchange", []byte(`{"session":`))

	OAuthExchange(ctx, openHandlerTestStore(t), OAuthFlows{flow.ProviderID(): flow})

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestOAuthHandlersReturnFlowErrors(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("codex"), pollErr: errors.New("poll failed")}
	ctx := oauthRequestCtx(t, fasthttp.MethodGet, "/api/oauth/codex/poll?session_id=device-123", nil)

	OAuthPoll(ctx, openHandlerTestStore(t), OAuthFlows{flow.ProviderID(): flow})

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
