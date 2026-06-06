package api

import (
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/governance"
	"github.com/bloodf/g0router/internal/ratelimit"
	"github.com/valyala/fasthttp"
)

// TestClassifySourceIPNilReturnsPublic exercises the ip==nil branch.
func TestClassifySourceIPNilReturnsPublic(t *testing.T) {
	got := classifySourceIP(nil)
	if got != "public" {
		t.Fatalf("classifySourceIP(nil) = %q, want public", got)
	}
}

// TestApplyMiddlewareSourceNotAllowedReturns403 exercises the !sourceAllowed branch.
func TestApplyMiddlewareSourceNotAllowedReturns403(t *testing.T) {
	s := newAPITestStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.AllowedSources = []string{"tailscale"}
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s, RequireAPIKey: false})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	resp, err := httpClient().Get(base + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("AllowedSources restricted: got %d, want 403", resp.StatusCode)
	}
}

// TestOriginMatchesHostEmptyReturnsFalse exercises the empty origin branch.
func TestOriginMatchesHostEmptyReturnsFalse(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Host", "localhost")
	if srv.originMatchesHost(ctx) {
		t.Fatal("originMatchesHost should return false with no origin/referer")
	}
}

// TestOriginMatchesHostBadURLReturnsFalse exercises the url.Parse error branch.
func TestOriginMatchesHostBadURLReturnsFalse(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Host", "localhost")
	ctx.Request.Header.Set("Origin", "://bad-url")
	if srv.originMatchesHost(ctx) {
		t.Fatal("originMatchesHost should return false for unparseable origin")
	}
}

// TestOriginMatchesHostRefererUsed exercises the fallback to Referer header.
func TestOriginMatchesHostRefererUsed(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Host", "localhost:8080")
	ctx.Request.Header.Set("Referer", "http://localhost:8080/page")
	if !srv.originMatchesHost(ctx) {
		t.Fatal("originMatchesHost should return true when referer matches host")
	}
}

// TestIsProtectedManagementPathExempt exercises the isExemptRoute true branch.
func TestIsProtectedManagementPathExempt(t *testing.T) {
	for _, path := range []string{"/api/oauth/callback", "/api/mcp/oauth/callback", "/api/auth/setup", "/api/auth/login", "/api/auth/status"} {
		if isProtectedManagementPath(path) {
			t.Fatalf("isProtectedManagementPath(%q) = true, want false", path)
		}
	}
}

// TestValidSessionStoreError exercises the GetDashboardSessionByRawToken error branch.
func TestValidSessionStoreError(t *testing.T) {
	s := newAPITestStore(t)
	s.Close()

	srv := NewServer(ServerConfig{Store: s})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Cookie", "g0router_session=token")
	_, err := srv.validSession(ctx)
	if err == nil {
		t.Fatal("expected error for closed db")
	}
}

// TestApplyMiddlewareValidSessionError exercises the validSession error branch in applyMiddleware.
func TestApplyMiddlewareValidSessionError(t *testing.T) {
	s := newAPITestStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s, RequireAPIKey: true})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/settings")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.Header.Set("Cookie", "g0router_session=token")
	result := srv.applyMiddleware(ctx)
	if result {
		t.Fatalf("expected applyMiddleware to return false when validSession errors, got true (status=%d)", ctx.Response.StatusCode())
	}
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

// TestValidVirtualKeyWithTeamID exercises the key.TeamID branch in validVirtualKey.
func TestValidVirtualKeyWithTeamID(t *testing.T) {
	s := newAPITestStore(t)
	team, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	_, raw, err := s.CreateVirtualKey("team-key", &team.ID, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	srv := NewServer(ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		Governance: governance.New(s, ratelimit.NewLimiter()),
	})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer "+raw)

	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("validAPIKey: %v", err)
	}
	if !ok {
		t.Fatal("expected valid virtual key with team to pass")
	}
	teamID := userValueStringPtr(ctx, requestVirtualKeyTeamIDKey)
	if teamID == nil || *teamID != "1" {
		t.Fatalf("team id = %v, want 1", teamID)
	}
}
