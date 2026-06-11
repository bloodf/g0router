package server

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func newTestGuard(t *testing.T) (*Guard, *auth.Sessions, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	g := &Guard{
		Sessions: sessions,
		Settings: st,
	}
	return g, sessions, st
}

func testNextHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(`{"data":"ok","error":null}`)
}

func callGuard(t *testing.T, g *Guard, method, uri string, headers map[string]string, cookies map[string]string) *fasthttp.RequestCtx {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	for k, v := range headers {
		if strings.EqualFold(k, "Host") {
			ctx.Request.SetHost(v)
		} else {
			ctx.Request.Header.Set(k, v)
		}
	}
	for k, v := range cookies {
		ctx.Request.Header.SetCookie(k, v)
	}
	wrapped := g.Wrap(testNextHandler)
	wrapped(&ctx)
	return &ctx
}

func envelopeMessage(t *testing.T, body []byte) string {
	t.Helper()
	var env struct {
		Data  any `json:"data"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v\nbody: %s", err, body)
	}
	return env.Error.Message
}

func TestGuardListContents(t *testing.T) {
	wantPublic := []string{
		"/api/health",
		"/api/init",
		"/api/locale",
		"/api/auth/login",
		"/api/auth/logout",
		"/api/auth/status",
		"/api/auth/oidc",
		"/api/version",
		"/api/settings/require-login",
	}
	if len(PUBLIC_API_PATHS) != len(wantPublic) {
		t.Fatalf("PUBLIC_API_PATHS len = %d, want %d", len(PUBLIC_API_PATHS), len(wantPublic))
	}
	for i := range wantPublic {
		if PUBLIC_API_PATHS[i] != wantPublic[i] {
			t.Fatalf("PUBLIC_API_PATHS[%d] = %q, want %q", i, PUBLIC_API_PATHS[i], wantPublic[i])
		}
	}

	wantLocal := []string{"/api/mcp/"}
	if len(LOCAL_ONLY_PATHS) != len(wantLocal) {
		t.Fatalf("LOCAL_ONLY_PATHS len = %d, want %d", len(LOCAL_ONLY_PATHS), len(wantLocal))
	}
	for i := range wantLocal {
		if LOCAL_ONLY_PATHS[i] != wantLocal[i] {
			t.Fatalf("LOCAL_ONLY_PATHS[%d] = %q, want %q", i, LOCAL_ONLY_PATHS[i], wantLocal[i])
		}
	}

	if len(ALWAYS_PROTECTED) != 0 {
		t.Fatalf("ALWAYS_PROTECTED len = %d, want 0", len(ALWAYS_PROTECTED))
	}
}

func TestGuardLocalOnlyPaths(t *testing.T) {
	g, _, st := newTestGuard(t)
	if err := st.SetSettings(map[string]string{"requireLogin": "false"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}

	// loopback + no origin → allow
	ctx := callGuard(t, g, "GET", "/api/mcp/tools", map[string]string{"Host": "localhost"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("localhost no-origin status = %d", ctx.Response.StatusCode())
	}

	// remote host → 403
	ctx = callGuard(t, g, "GET", "/api/mcp/tools", map[string]string{"Host": "evil.com"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("remote status = %d, want 403", ctx.Response.StatusCode())
	}
	msg := envelopeMessage(t, ctx.Response.Body())
	if msg != "Local only: CLI token required" {
		t.Fatalf("remote error = %q", msg)
	}

	// loopback host + remote origin → 403
	ctx = callGuard(t, g, "GET", "/api/mcp/tools", map[string]string{
		"Host":   "localhost",
		"Origin": "http://evil.com",
	}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("loopback+remote-origin status = %d, want 403", ctx.Response.StatusCode())
	}

	// malformed origin → 403
	ctx = callGuard(t, g, "GET", "/api/mcp/tools", map[string]string{
		"Host":   "localhost",
		"Origin": "not-a-valid-url",
	}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
		t.Fatalf("malformed-origin status = %d, want 403", ctx.Response.StatusCode())
	}

	// valid CLI token bypasses on remote host
	g.CLITokenValidator = func(_ *fasthttp.RequestCtx) bool { return true }
	ctx = callGuard(t, g, "GET", "/api/mcp/tools", map[string]string{"Host": "evil.com"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("remote+cli-token status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestGuardAlwaysProtected(t *testing.T) {
	old := ALWAYS_PROTECTED
	ALWAYS_PROTECTED = []string{"/api/shutdown"}
	t.Cleanup(func() { ALWAYS_PROTECTED = old })

	g, sessions, _ := newTestGuard(t)

	// no session → 401
	ctx := callGuard(t, g, "POST", "/api/shutdown", nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("no session status = %d, want 401", ctx.Response.StatusCode())
	}

	// valid session → allow
	token, err := sessions.Login("admin", "123456")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	ctx = callGuard(t, g, "POST", "/api/shutdown", map[string]string{"Authorization": "Bearer " + token}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("valid session status = %d, want 200", ctx.Response.StatusCode())
	}

	// nil CLITokenValidator → deny even with header present
	g.CLITokenValidator = nil
	ctx = callGuard(t, g, "POST", "/api/shutdown", map[string]string{"X-Cli-Token": "whatever"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("nil validator status = %d, want 401", ctx.Response.StatusCode())
	}
}

func TestGuardV1LoopbackKeyless(t *testing.T) {
	g, _, _ := newTestGuard(t)

	cases := []string{
		"/v1/chat/completions",
		"/v1beta/models",
		"/api/v1/embeddings",
		"/api/v1beta/messages",
	}
	for _, path := range cases {
		ctx := callGuard(t, g, "POST", path, map[string]string{"Host": "localhost"}, nil)
		if ctx.Response.StatusCode() != fasthttp.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, ctx.Response.StatusCode())
		}
	}
}

func TestGuardV1RemoteRequiresKey(t *testing.T) {
	g, _, _ := newTestGuard(t)

	cases := []string{
		"/v1/chat/completions",
		"/v1beta/models",
		"/api/v1/embeddings",
		"/api/v1beta/messages",
	}
	for _, path := range cases {
		ctx := callGuard(t, g, "POST", path, map[string]string{"Host": "remote.example.com"}, nil)
		if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, ctx.Response.StatusCode())
		}
		msg := envelopeMessage(t, ctx.Response.Body())
		if msg != "API key required for remote API access" {
			t.Fatalf("%s error = %q, want %q", path, msg, "API key required for remote API access")
		}
	}
}

func TestGuardV1RemoteValidKey(t *testing.T) {
	g, _, st := newTestGuard(t)
	created, err := st.CreateAPIKey("remote")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	key := created.Key

	g.APIKeyValidator = auth.NewAPIKeyValidator(func(k string) (string, bool, error) {
		rec, err := st.GetAPIKeyByKey(k)
		if err != nil {
			return "", false, err
		}
		return rec.MachineID, rec.IsActive, nil
	})

	// Valid key in Authorization Bearer header.
	ctx := callGuard(t, g, "POST", "/v1/chat/completions",
		map[string]string{"Host": "remote.example.com", "Authorization": "Bearer " + key}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("Bearer valid key status = %d, want 200", ctx.Response.StatusCode())
	}

	// Valid key in x-api-key header.
	ctx = callGuard(t, g, "POST", "/v1/chat/completions",
		map[string]string{"Host": "remote.example.com", "x-api-key": key}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("x-api-key valid key status = %d, want 200", ctx.Response.StatusCode())
	}

	// Inactive key -> 401.
	if err := st.SetAPIKeyActive(created.ID, false); err != nil {
		t.Fatalf("SetAPIKeyActive: %v", err)
	}
	ctx = callGuard(t, g, "POST", "/v1/chat/completions",
		map[string]string{"Host": "remote.example.com", "x-api-key": key}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("inactive key status = %d, want 401", ctx.Response.StatusCode())
	}
	if err := st.SetAPIKeyActive(created.ID, true); err != nil {
		t.Fatalf("SetAPIKeyActive reactivate: %v", err)
	}

	// CRC-corrupted key -> 401.
	corrupt := key[:len(key)-1] + "0"
	if corrupt == key {
		corrupt = key[:len(key)-1] + "1"
	}
	ctx = callGuard(t, g, "POST", "/v1/chat/completions",
		map[string]string{"Host": "remote.example.com", "x-api-key": corrupt}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("corrupt key status = %d, want 401", ctx.Response.StatusCode())
	}
}

func TestGuardCLIToken(t *testing.T) {
	g, _, st := newTestGuard(t)
	dataDir := st.DataDir()
	cliToken, err := auth.MachineID(dataDir, "9r-cli-auth")
	if err != nil {
		t.Fatalf("MachineID cli: %v", err)
	}
	g.CLITokenValidator = auth.NewCLITokenValidator(dataDir)

	// Set ALWAYS_PROTECTED to a route that exists for this test.
	old := ALWAYS_PROTECTED
	ALWAYS_PROTECTED = []string{"/api/shutdown"}
	t.Cleanup(func() { ALWAYS_PROTECTED = old })

	// Correct CLI token bypasses always-protected.
	ctx := callGuard(t, g, "POST", "/api/shutdown",
		map[string]string{"Host": "remote.example.com", "x-9r-cli-token": cliToken}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("always-protected+cli status = %d, want 200", ctx.Response.StatusCode())
	}

	// Wrong CLI token -> 401 for always-protected.
	ctx = callGuard(t, g, "POST", "/api/shutdown",
		map[string]string{"Host": "remote.example.com", "x-9r-cli-token": "wrong"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("wrong cli always-protected status = %d, want 401", ctx.Response.StatusCode())
	}
}

func TestGuardV1CLITokenRejectedRemote(t *testing.T) {
	g, _, st := newTestGuard(t)
	dataDir := st.DataDir()
	cliToken, err := auth.MachineID(dataDir, "9r-cli-auth")
	if err != nil {
		t.Fatalf("MachineID cli: %v", err)
	}
	g.CLITokenValidator = auth.NewCLITokenValidator(dataDir)

	cases := []string{
		"/v1/chat/completions",
		"/v1beta/models",
		"/api/v1/embeddings",
		"/api/v1beta/messages",
	}
	for _, path := range cases {
		ctx := callGuard(t, g, "POST", path,
			map[string]string{"Host": "remote.example.com", "x-9r-cli-token": cliToken}, nil)
		if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
			t.Fatalf("%s status = %d, want 401", path, ctx.Response.StatusCode())
		}
		msg := envelopeMessage(t, ctx.Response.Body())
		if msg != "API key required for remote API access" {
			t.Fatalf("%s error = %q, want %q", path, msg, "API key required for remote API access")
		}
	}
}

func TestGuardApiDenyByDefault(t *testing.T) {
	g, _, _ := newTestGuard(t)

	// unlisted /api/x → 401
	ctx := callGuard(t, g, "GET", "/api/unknown", nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("unlisted api status = %d, want 401", ctx.Response.StatusCode())
	}

	// each PUBLIC_API_PATHS entry allowed
	for _, p := range PUBLIC_API_PATHS {
		// For prefix entries, test the exact path and a realistic subpath.
		paths := []string{p}
		if p == "/api/auth/oidc" {
			paths = append(paths, p+"/start")
		}
		for _, pp := range paths {
			ctx := callGuard(t, g, "GET", pp, nil, nil)
			if ctx.Response.StatusCode() != fasthttp.StatusOK {
				t.Fatalf("public path %q status = %d, want 200", pp, ctx.Response.StatusCode())
			}
		}
	}

	// requireLogin=false allows unlisted /api/x
	st := g.Settings.(*store.Store)
	if err := st.SetSettings(map[string]string{"requireLogin": "false"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	ctx = callGuard(t, g, "GET", "/api/unknown", nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("requireLogin=false status = %d, want 200", ctx.Response.StatusCode())
	}
}

func TestGuardDashboardRedirects(t *testing.T) {
	g, sessions, st := newTestGuard(t)

	// no token → redirect /login
	ctx := callGuard(t, g, "GET", "/dashboard", map[string]string{"Host": "localhost"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("no token status = %d, want 302", ctx.Response.StatusCode())
	}
	loc := string(ctx.Response.Header.Peek("Location"))
	if !strings.HasSuffix(loc, "/login") {
		t.Fatalf("no token redirect = %q, want suffix /login", loc)
	}

	// requireLogin=false allows
	if err := st.SetSettings(map[string]string{"requireLogin": "false"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	ctx = callGuard(t, g, "GET", "/dashboard", map[string]string{"Host": "localhost"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("requireLogin=false status = %d, want 200", ctx.Response.StatusCode())
	}

	// reset
	if err := st.SetSettings(map[string]string{"requireLogin": ""}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}

	// tunnel host + access disabled → redirect /login
	if err := st.SetSettings(map[string]string{
		"tunnelUrl":           "https://tunnel.example.com",
		"tunnelDashboardAccess": "",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	ctx = callGuard(t, g, "GET", "/dashboard", map[string]string{"Host": "tunnel.example.com"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("tunnel blocked status = %d, want 302", ctx.Response.StatusCode())
	}
	loc = string(ctx.Response.Header.Peek("Location"))
	if !strings.HasSuffix(loc, "/login") {
		t.Fatalf("tunnel blocked redirect = %q, want suffix /login", loc)
	}

	// tunnelDashboardAccess=true removes the tunnel block, but login is still required.
	if err := st.SetSettings(map[string]string{
		"tunnelUrl":             "https://tunnel.example.com",
		"tunnelDashboardAccess": "true",
		"requireLogin":          "false",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	ctx = callGuard(t, g, "GET", "/dashboard", map[string]string{"Host": "tunnel.example.com"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("tunnel allowed status = %d, want 200", ctx.Response.StatusCode())
	}

	// valid session does not bypass the tunnel block when access is disabled.
	if err := st.SetSettings(map[string]string{
		"tunnelUrl":             "https://tunnel.example.com",
		"tunnelDashboardAccess": "",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	token, err := sessions.Login("admin", "123456")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	ctx = callGuard(t, g, "GET", "/dashboard", map[string]string{"Host": "tunnel.example.com"}, map[string]string{"g0_session": token})
	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("tunnel+session status = %d, want 302", ctx.Response.StatusCode())
	}
}

func TestGuardRootRedirect(t *testing.T) {
	g, _, _ := newTestGuard(t)

	ctx := callGuard(t, g, "GET", "/", map[string]string{"Host": "localhost"}, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("root status = %d, want 302", ctx.Response.StatusCode())
	}
	loc := string(ctx.Response.Header.Peek("Location"))
	if !strings.HasSuffix(loc, "/dashboard") {
		t.Fatalf("root redirect = %q, want suffix /dashboard", loc)
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	g, sessions, st := newTestGuard(t)

	// Use real login handler to get the cookie.
	h := admin.New(st, sessions, nil)
	var loginCtx fasthttp.RequestCtx
	loginCtx.Request.Header.SetMethod("POST")
	loginCtx.Request.SetRequestURI("/api/auth/login")
	loginCtx.Request.SetBody([]byte(`{"username":"admin","password":"123456"}`))
	h.Login(&loginCtx)

	if loginCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("login status = %d", loginCtx.Response.StatusCode())
	}

	setCookie := string(loginCtx.Response.Header.Peek("Set-Cookie"))
	if !strings.Contains(setCookie, "g0_session=") {
		t.Fatalf("Set-Cookie missing g0_session: %q", setCookie)
	}

	token := extractCookieValue(setCookie, "g0_session")
	if token == "" {
		t.Fatalf("could not extract token from %q", setCookie)
	}

	// Guarded request with the exact cookie passes.
	ctx := callGuard(t, g, "GET", "/api/settings", nil, map[string]string{"g0_session": token})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("guarded with valid cookie status = %d, want 200", ctx.Response.StatusCode())
	}

	// Renamed cookie → 401
	ctx = callGuard(t, g, "GET", "/api/settings", nil, map[string]string{"wrong_cookie": token})
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("guarded with wrong cookie status = %d, want 401", ctx.Response.StatusCode())
	}
}

func extractCookieValue(setCookie, name string) string {
	prefix := name + "="
	start := strings.Index(setCookie, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(setCookie[start:], ";")
	if end == -1 {
		return setCookie[start:]
	}
	return setCookie[start : start+end]
}


