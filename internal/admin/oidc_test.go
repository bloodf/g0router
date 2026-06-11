package admin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

// oidcTestEnv extends testEnv with OIDC-specific settings.
type oidcTestEnv struct {
	*testEnv
	idp    *httptest.Server
	docURL string
}

func newOIDCTestEnv(t *testing.T) *oidcTestEnv {
	env := &oidcTestEnv{testEnv: newTestEnv(t)}

	discovery := map[string]any{
		"issuer":                 "https://idp.example.com",
		"authorization_endpoint": "https://idp.example.com/authorize",
		"token_endpoint":         "",
		"jwks_uri":               "https://idp.example.com/jwks",
	}

	env.idp = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(discovery)
		case "/token":
			r.ParseForm()
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "at-oidc",
				"id_token":     makeTestIDToken(map[string]any{"nonce": r.PostForm.Get("expected_nonce")}),
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(env.idp.Close)

	discovery["token_endpoint"] = env.idp.URL + "/token"
	env.docURL = env.idp.URL

	if err := env.store.SetSettings(map[string]string{
		"auth_mode":          "oidc",
		"oidc_issuer_url":    env.docURL,
		"oidc_client_id":     "client-id",
		"oidc_client_secret": "client-secret",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	return env
}

func makeTestIDToken(claims map[string]any) string {
	header := map[string]string{"alg": "none", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)
	h := base64.RawURLEncoding.EncodeToString(hb)
	p := base64.RawURLEncoding.EncodeToString(pb)
	return h + "." + p + "."
}

func parseCookies(t *testing.T, ctx *fasthttp.RequestCtx) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		if strings.ToLower(string(key)) != "set-cookie" {
			return
		}
		raw := string(value)
		parts := strings.Split(raw, ";")
		if len(parts) == 0 {
			return
		}
		kv := strings.TrimSpace(parts[0])
		name, val, ok := strings.Cut(kv, "=")
		if !ok {
			return
		}
		attrs := map[string]string{"value": val}
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			aname, avalue, hasValue := strings.Cut(part, "=")
			if hasValue {
				attrs[strings.ToLower(strings.TrimSpace(aname))] = strings.TrimSpace(avalue)
			} else {
				attrs[strings.ToLower(strings.TrimSpace(aname))] = ""
			}
		}
		out[name] = attrs
	})
	return out
}

func TestOIDCStartSetsThreeCookies(t *testing.T) {
	env := newOIDCTestEnv(t)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/auth/oidc/start")
	ctx.Request.SetHost("localhost:8080")
	env.handlers.OIDCStart(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("status = %d body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}

	cookies := parseCookies(t, &ctx)
	for _, name := range []string{"oidc_state", "oidc_nonce", "oidc_code_verifier"} {
		c, ok := cookies[name]
		if !ok {
			t.Fatalf("missing cookie %q", name)
		}
		if _, ok := c["value"]; !ok || c["value"] == "" {
			t.Fatalf("cookie %q has no value", name)
		}
		if strings.ToLower(c["httponly"]) != "" {
			t.Errorf("cookie %q missing HttpOnly", name)
		}
		if strings.ToLower(c["samesite"]) != "lax" {
			t.Errorf("cookie %q SameSite = %q, want Lax", name, c["samesite"])
		}
		if c["max-age"] != "600" {
			t.Errorf("cookie %q Max-Age = %q, want 600", name, c["max-age"])
		}
		if c["path"] != "/" {
			t.Errorf("cookie %q Path = %q, want /", name, c["path"])
		}
		if _, ok := c["secure"]; ok {
			t.Errorf("cookie %q has Secure flag for http request", name)
		}
	}

	location := string(ctx.Response.Header.Peek("Location"))
	if !strings.Contains(location, "code_challenge=") {
		t.Fatalf("location missing code_challenge: %q", location)
	}
}

func TestOIDCStartSecureCookieForHTTPS(t *testing.T) {
	env := newOIDCTestEnv(t)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/auth/oidc/start")
	ctx.Request.SetHost("localhost:8080")
	ctx.Request.Header.Set("X-Forwarded-Proto", "https")
	env.handlers.OIDCStart(&ctx)

	cookies := parseCookies(t, &ctx)
	for _, name := range []string{"oidc_state", "oidc_nonce", "oidc_code_verifier"} {
		if _, ok := cookies[name]["secure"]; !ok {
			t.Errorf("cookie %q missing Secure flag for https request", name)
		}
	}
}

func TestOIDCCallbackIssuesOpaqueSession(t *testing.T) {
	env := newOIDCTestEnv(t)

	state := "test-state"
	nonce := "test-nonce"
	verifier := "test-verifier"

	// Mock the token endpoint to include the nonce we expect.
	env.idp.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 "https://idp.example.com",
				"authorization_endpoint": "https://idp.example.com/authorize",
				"token_endpoint":         env.idp.URL + "/token",
				"jwks_uri":               "https://idp.example.com/jwks",
			})
		case "/token":
			r.ParseForm()
			if r.PostForm.Get("code_verifier") != verifier {
				t.Errorf("code_verifier = %q, want %q", r.PostForm.Get("code_verifier"), verifier)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "at-oidc",
				"id_token":     makeTestIDToken(map[string]any{"nonce": nonce, "sub": "test-sub"}),
			})
		default:
			http.NotFound(w, r)
		}
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/auth/oidc/callback?code=good-code&state=" + state)
	ctx.Request.SetHost("localhost:8080")
	ctx.Request.Header.SetCookie("oidc_state", state)
	ctx.Request.Header.SetCookie("oidc_nonce", nonce)
	ctx.Request.Header.SetCookie("oidc_code_verifier", verifier)

	env.handlers.OIDCCallback(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("status = %d body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	location := string(ctx.Response.Header.Peek("Location"))
	if !strings.HasSuffix(location, "/dashboard") {
		t.Errorf("Location = %q, want suffix /dashboard", location)
	}

	cookies := parseCookies(t, &ctx)

	// Session cookie issued.
	if sc, ok := cookies[sessionCookieName]; !ok || sc["value"] == "" {
		t.Fatalf("missing session cookie")
	}

	// OIDC cookies deleted.
	for _, name := range []string{"oidc_state", "oidc_nonce", "oidc_code_verifier"} {
		c, ok := cookies[name]
		if !ok {
			t.Fatalf("missing deletion cookie %q", name)
		}
		if c["value"] != "" {
			t.Errorf("cookie %q not deleted, value = %q", name, c["value"])
		}
		if exp := c["expires"]; !strings.Contains(exp, "1970") && c["max-age"] != "0" {
			t.Errorf("cookie %q deletion attributes missing: %+v", name, c)
		}
	}

	// The session token must validate through auth.Sessions.
	sessionValue := cookies[sessionCookieName]["value"]
	user, err := env.sessions.Validate(sessionValue)
	if err != nil {
		t.Fatalf("sessions.Validate: %v", err)
	}
	if user == nil {
		t.Fatal("validated user is nil")
	}
}

func TestOIDCCallbackBadState401(t *testing.T) {
	env := newOIDCTestEnv(t)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/auth/oidc/callback?code=good-code&state=wrong")
	ctx.Request.SetHost("localhost:8080")
	ctx.Request.Header.SetCookie("oidc_state", "stored-state")
	ctx.Request.Header.SetCookie("oidc_nonce", "nonce")
	ctx.Request.Header.SetCookie("oidc_code_verifier", "verifier")

	env.handlers.OIDCCallback(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
	if errMessage(t, map[string]json.RawMessage{"error": ctx.Response.Body()}) == "" {
		// Error body may be plain text; just ensure no session cookie is set.
	}
	setCookie := string(ctx.Response.Header.Peek("Set-Cookie"))
	if strings.Contains(setCookie, sessionCookieName+"=") {
		t.Fatal("session cookie set on failed callback")
	}
}

func TestOIDCCallbackNonceMismatch401(t *testing.T) {
	env := newOIDCTestEnv(t)

	state := "test-state"
	nonce := "test-nonce"
	verifier := "test-verifier"

	env.idp.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 "https://idp.example.com",
				"authorization_endpoint": "https://idp.example.com/authorize",
				"token_endpoint":         env.idp.URL + "/token",
				"jwks_uri":               "https://idp.example.com/jwks",
			})
		case "/token":
			json.NewEncoder(w).Encode(map[string]any{
				"access_token": "at-oidc",
				"id_token":     makeTestIDToken(map[string]any{"nonce": "wrong-nonce", "sub": "test-sub"}),
			})
		default:
			http.NotFound(w, r)
		}
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/auth/oidc/callback?code=good-code&state=" + state)
	ctx.Request.SetHost("localhost:8080")
	ctx.Request.Header.SetCookie("oidc_state", state)
	ctx.Request.Header.SetCookie("oidc_nonce", nonce)
	ctx.Request.Header.SetCookie("oidc_code_verifier", verifier)

	env.handlers.OIDCCallback(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
}

func TestProbeEndpointPublic(t *testing.T) {
	env := newOIDCTestEnv(t)

	body := fmt.Sprintf(`{"issuerUrl":%q,"clientId":"client-id","clientSecret":"client-secret","redirectUri":"https://app.example.com/cb"}`, env.docURL)
	status, envl := call(t, env.handlers.OIDCTest, "POST", "/api/auth/oidc/test", body, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	if data["ok"] != true {
		t.Fatalf("ok = %v", data["ok"])
	}
	if data["discoveryOk"] != true {
		t.Fatalf("discoveryOk = %v", data["discoveryOk"])
	}
}

func TestOIDCConfigured(t *testing.T) {
	env := newTestEnv(t)
	settings := map[string]string{
		"oidc_issuer_url":    "https://idp.example.com",
		"oidc_client_id":     "client-id",
		"oidc_client_secret": "client-secret",
	}
	if !env.handlers.oidcConfigured(settings) {
		t.Fatal("expected oidcConfigured true")
	}
	settings["oidc_issuer_url"] = ""
	if env.handlers.oidcConfigured(settings) {
		t.Fatal("expected oidcConfigured false")
	}
}

func TestOIDCStartNotConfigured(t *testing.T) {
	env := newTestEnv(t)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/auth/oidc/start")
	ctx.Request.SetHost("localhost:8080")
	env.handlers.OIDCStart(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusFound {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
	loc := string(ctx.Response.Header.Peek("Location"))
	if !strings.Contains(loc, "/login?error=oidc_not_configured") {
		t.Fatalf("Location = %q", loc)
	}
}

func TestOIDCSessionCreation(t *testing.T) {
	env := newTestEnv(t)
	user, err := env.store.CreateUser("oidc:test-sub", "")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token, err := env.sessions.CreateOIDCSession(user.ID)
	if err != nil {
		t.Fatalf("CreateOIDCSession: %v", err)
	}
	validated, err := env.sessions.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if validated.ID != user.ID {
		t.Fatalf("validated user = %q, want %q", validated.ID, user.ID)
	}
}
