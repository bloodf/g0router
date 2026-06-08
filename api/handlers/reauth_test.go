package handlers

import (
	"bytes"
	"testing"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestConnectionResponseIncludesNeedsReauthFields(t *testing.T) {
	s := newHandlerStore(t)

	// Create a connection and then mark it as needing re-auth.
	conn := &store.Connection{
		Provider: "openai",
		AuthType: store.AuthTypeOAuth,
		IsActive: true,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.MarkConnectionRefreshFailure(conn.ID, "invalid_grant"); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	// needs_reauth should appear.
	if !bytes.Contains(body, []byte(`"needs_reauth":true`)) {
		t.Errorf("response missing needs_reauth:true; body=%s", body)
	}
	// last_error should appear with value (UI-facing name for last_refresh_error).
	if !bytes.Contains(body, []byte(`"last_error"`)) {
		t.Errorf("response missing last_error field; body=%s", body)
	}
	if !bytes.Contains(body, []byte("invalid_grant")) {
		t.Errorf("response missing error reason; body=%s", body)
	}
}

func TestConnectionResponseNeedsReauthFalseByDefault(t *testing.T) {
	s := newHandlerStore(t)

	conn := &store.Connection{
		Provider: "openai",
		AuthType: store.AuthTypeAPIKey,
		IsActive: true,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d; body=%s", ctx.Response.StatusCode(), body)
	}
	// needs_reauth false — present with false value or absent; must not be true.
	if bytes.Contains(body, []byte(`"needs_reauth":true`)) {
		t.Errorf("needs_reauth should not be true for fresh connection; body=%s", body)
	}
	// last_error omitted when nil (omitempty).
	if bytes.Contains(body, []byte(`"last_error"`)) {
		t.Errorf("last_error should be omitted when nil; body=%s", body)
	}
}

func TestConnectionResponseClearedReauthShowsFalse(t *testing.T) {
	s := newHandlerStore(t)

	conn := &store.Connection{
		Provider: "openai",
		AuthType: store.AuthTypeOAuth,
		IsActive: true,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.MarkConnectionRefreshFailure(conn.ID, "token_expired"); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}
	if err := s.ClearConnectionRefreshFailure(conn.ID); err != nil {
		t.Fatalf("ClearConnectionRefreshFailure: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d; body=%s", ctx.Response.StatusCode(), body)
	}
	if bytes.Contains(body, []byte(`"needs_reauth":true`)) {
		t.Errorf("needs_reauth should be false after clear; body=%s", body)
	}
	if bytes.Contains(body, []byte(`"last_error"`)) {
		t.Errorf("last_error should be absent after clear; body=%s", body)
	}
	// Must not leak token_expired in any field that would look like a secret.
	assertNoCredentialFields(t, body)
}

func TestConnectionResponseNeedsReauthNoTokenLeak(t *testing.T) {
	s := newHandlerStore(t)

	conn := &store.Connection{
		Provider:     "openai",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  strPtr("super-secret-access-token"),
		RefreshToken: strPtr("super-secret-refresh-token"),
		IsActive:     true,
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.MarkConnectionRefreshFailure(conn.ID, "invalid_grant"); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Connections(ctx, s, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d; body=%s", ctx.Response.StatusCode(), body)
	}

	assertNoCredentialFields(t, body)
	for _, secret := range []string{"super-secret-access-token", "super-secret-refresh-token"} {
		if bytes.Contains(body, []byte(secret)) {
			t.Errorf("response leaked secret %q; body=%s", secret, body)
		}
	}
}

func strPtr(s string) *string { return &s }
