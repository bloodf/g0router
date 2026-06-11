package admin

import (
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func loginWithHost(t *testing.T, h *Handlers, host string) (int, string, bool) {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/api/auth/login")
	ctx.Request.SetBody([]byte(`{"username":"admin","password":"123456"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	if host != "" {
		ctx.Request.SetHost(host)
	}
	h.Login(&ctx)

	setCookie := string(ctx.Response.Header.Peek("Set-Cookie"))
	hasSessionCookie := strings.Contains(setCookie, sessionCookieName+"=")
	return ctx.Response.StatusCode(), string(ctx.Response.Body()), hasSessionCookie
}

func TestLoginBlockedViaTunnelHost(t *testing.T) {
	env := newTestEnv(t)

	// Block login when Host matches tunnelUrl and tunnelDashboardAccess is false.
	if err := env.store.SetSettings(map[string]string{
		"tunnelUrl":             "https://tunnel.example",
		"tunnelDashboardAccess": "false",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}

	status, body, hasCookie := loginWithHost(t, env.handlers, "tunnel.example")
	if status != fasthttp.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", status, body)
	}
	if !strings.Contains(body, "Dashboard access via tunnel is disabled") {
		t.Fatalf("body = %s, want tunnel-disabled error", body)
	}
	if hasCookie {
		t.Fatalf("Set-Cookie present for blocked login: %v", hasCookie)
	}

	// Allow login when tunnelDashboardAccess is true.
	if err := env.store.SetSettings(map[string]string{
		"tunnelUrl":             "https://tunnel.example",
		"tunnelDashboardAccess": "true",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	status, body, hasCookie = loginWithHost(t, env.handlers, "tunnel.example")
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, body)
	}
	if !hasCookie {
		t.Fatalf("Set-Cookie session missing for allowed login")
	}
}

func TestLogoutClearsOIDCCookies(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/api/auth/logout")
	ctx.Request.Header.Set("Authorization", "Bearer "+token)
	ctx.Request.Header.SetCookie("oidc_state", "state-value")
	ctx.Request.Header.SetCookie("oidc_nonce", "nonce-value")
	ctx.Request.Header.SetCookie("oidc_code_verifier", "verifier-value")

	env.handlers.RequireSession(env.handlers.Logout)(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}

	cookies := parseCookies(t, &ctx)
	for _, name := range []string{"oidc_state", "oidc_nonce", "oidc_code_verifier"} {
		c, ok := cookies[name]
		if !ok {
			t.Fatalf("missing deletion cookie %q", name)
		}
		if c["value"] != "" {
			t.Errorf("cookie %q not deleted, value = %q", name, c["value"])
		}
	}

	// Session should also be cleared.
	if _, ok := cookies[sessionCookieName]; !ok {
		t.Fatal("missing session deletion cookie")
	}
}

func TestLoginNormalHostUnaffected(t *testing.T) {
	env := newTestEnv(t)

	// Normal host log in regardless of tunnelDashboardAccess toggle.
	if err := env.store.SetSettings(map[string]string{
		"tunnelUrl":             "https://tunnel.example",
		"tunnelDashboardAccess": "false",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}

	status, body, hasCookie := loginWithHost(t, env.handlers, "localhost:8080")
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, body)
	}
	if !hasCookie {
		t.Fatalf("Set-Cookie session missing for localhost login")
	}
}
