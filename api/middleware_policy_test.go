package api

import (
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

// policyValidator implements APIKeyIdentityValidator returning a fixed identity.
type policyValidator struct {
	identity APIKeyIdentity
}

func (v policyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return true, nil
}

func (v policyValidator) ValidateAPIKeyIdentity(key, secret string) (*APIKeyIdentity, bool, error) {
	id := v.identity
	return &id, true, nil
}

func TestValidAPIKeyRejectsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()
	srv := NewServer(ServerConfig{APIKeyValidator: policyValidator{identity: APIKeyIdentity{ID: "id-1", ExpiresAt: &past}}})
	ctx := newAuthCtx("k")
	ok, err := srv.validAPIKey(ctx)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if ok {
		t.Fatal("expired key should be rejected")
	}
	if ctx.UserValue(requestAPIKeyIDKey) != nil {
		t.Fatal("expired key must not set api key id")
	}
}

func TestValidAPIKeyAllowsUnexpiredAndStoresPolicy(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()
	identity := APIKeyIdentity{ID: "id-1", ExpiresAt: &future, Scopes: []string{"gpt-*"}}
	srv := NewServer(ServerConfig{APIKeyValidator: policyValidator{identity: identity}})
	ctx := newAuthCtx("k")
	ok, err := srv.validAPIKey(ctx)
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	stored, has := ctx.UserValue(requestAPIKeyPolicyKey).(APIKeyIdentity)
	if !has || stored.ID != "id-1" || len(stored.Scopes) != 1 {
		t.Fatalf("policy not stored: %+v has=%v", stored, has)
	}
}

func TestApplyMiddlewareExpiredKeyReturns401(t *testing.T) {
	past := time.Now().Add(-time.Hour).Unix()
	srv := NewServer(ServerConfig{APIKeyValidator: policyValidator{identity: APIKeyIdentity{ID: "id-1", ExpiresAt: &past}}})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.Set("Authorization", "Bearer k")
	if srv.applyMiddleware(ctx) {
		t.Fatal("expired key must not pass middleware")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", ctx.Response.StatusCode())
	}
}
