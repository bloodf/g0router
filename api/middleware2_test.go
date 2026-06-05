package api

import (
	"errors"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestIsAllowedLocalOrigin(t *testing.T) {
	cases := map[string]bool{
		"":                       false,
		"http://localhost:3000":  true,
		"https://127.0.0.1:8080": true,
		"http://[::1]:9000":      true,
		"http://example.com":     false,
		"ftp://localhost":        false,
		"://bad":                 false,
	}
	for origin, want := range cases {
		if got := isAllowedLocalOrigin(origin); got != want {
			t.Fatalf("isAllowedLocalOrigin(%q) = %v, want %v", origin, got, want)
		}
	}
}

// identityValidator implements APIKeyIdentityValidator.
type identityValidator struct {
	id  string
	ok  bool
	err error
}

func (v identityValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return v.ok, v.err
}

func (v identityValidator) ValidateAPIKeyIdentity(key, secret string) (*APIKeyIdentity, bool, error) {
	if v.err != nil {
		return nil, false, v.err
	}
	if !v.ok {
		return nil, false, nil
	}
	return &APIKeyIdentity{ID: v.id}, true, nil
}

// plainValidator implements only APIKeyValidator.
type plainValidator struct {
	ok  bool
	err error
}

func (v plainValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return v.ok, v.err
}

func newAuthCtx(token string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	if token != "" {
		ctx.Request.Header.Set("Authorization", "Bearer "+token)
	}
	return ctx
}

func TestValidAPIKeyNilValidator(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ok, err := srv.validAPIKey(newAuthCtx("k"))
	if ok || err != nil {
		t.Fatalf("nil validator: ok=%v err=%v", ok, err)
	}
}

func TestValidAPIKeyNoKey(t *testing.T) {
	srv := NewServer(ServerConfig{APIKeyValidator: plainValidator{ok: true}})
	ok, err := srv.validAPIKey(&fasthttp.RequestCtx{})
	if ok || err != nil {
		t.Fatalf("no key: ok=%v err=%v", ok, err)
	}
}

func TestValidAPIKeyXAPIKeyHeader(t *testing.T) {
	srv := NewServer(ServerConfig{APIKeyValidator: plainValidator{ok: true}})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("X-API-Key", "k")
	ok, err := srv.validAPIKey(ctx)
	if !ok || err != nil {
		t.Fatalf("x-api-key: ok=%v err=%v", ok, err)
	}
}

func TestValidAPIKeyIdentitySuccess(t *testing.T) {
	srv := NewServer(ServerConfig{APIKeyValidator: identityValidator{id: "id-1", ok: true}})
	ctx := newAuthCtx("k")
	ok, err := srv.validAPIKey(ctx)
	if !ok || err != nil {
		t.Fatalf("identity ok: ok=%v err=%v", ok, err)
	}
	if ctx.UserValue(requestAPIKeyIDKey) != "id-1" {
		t.Fatalf("api key id not set: %v", ctx.UserValue(requestAPIKeyIDKey))
	}
}

func TestValidAPIKeyIdentityError(t *testing.T) {
	srv := NewServer(ServerConfig{APIKeyValidator: identityValidator{err: errors.New("boom")}})
	if _, err := srv.validAPIKey(newAuthCtx("k")); err == nil {
		t.Fatal("expected identity validator error")
	}
}

func TestValidAPIKeyPlainSuccessAndError(t *testing.T) {
	srv := NewServer(ServerConfig{APIKeyValidator: plainValidator{ok: true}})
	ctx := newAuthCtx("k")
	ok, err := srv.validAPIKey(ctx)
	if !ok || err != nil {
		t.Fatalf("plain ok: ok=%v err=%v", ok, err)
	}
	if ctx.UserValue(requestAuthTypeKey) != requestAuthTypeAPIKey {
		t.Fatalf("auth type not set")
	}

	srvErr := NewServer(ServerConfig{APIKeyValidator: plainValidator{err: errors.New("x")}})
	if _, err := srvErr.validAPIKey(newAuthCtx("k")); err == nil {
		t.Fatal("expected plain validator error")
	}
}

func TestApplyMiddlewareOptions(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	if srv.applyMiddleware(ctx) {
		t.Fatal("OPTIONS should short-circuit")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
}

func TestApplyMiddlewareUnauthorized(t *testing.T) {
	srv := NewServer(ServerConfig{RequireAPIKey: true, APIKeyValidator: plainValidator{ok: false}})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	if srv.applyMiddleware(ctx) {
		t.Fatal("missing key should be rejected")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
}

func TestApplyMiddlewareValidatorError(t *testing.T) {
	srv := NewServer(ServerConfig{RequireAPIKey: true, APIKeyValidator: plainValidator{err: errors.New("x")}})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.Set("Authorization", "Bearer k")
	if srv.applyMiddleware(ctx) {
		t.Fatal("validator error should reject")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestApplyMiddlewareAllowsCORSOrigin(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Origin", "http://localhost:5173")
	ctx.Request.SetRequestURI("/healthz")
	if !srv.applyMiddleware(ctx) {
		t.Fatal("expected pass-through")
	}
	if got := string(ctx.Response.Header.Peek("Access-Control-Allow-Origin")); got != "http://localhost:5173" {
		t.Fatalf("CORS origin = %q", got)
	}
}

func TestNewRequestIDFormat(t *testing.T) {
	id, err := newRequestID()
	if err != nil {
		t.Fatalf("newRequestID: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("request id = %q (len %d), want UUID form", id, len(id))
	}
}
