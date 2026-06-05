package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
)

// TestRefreshConnectionEmptyAccessTokenMarksFailure exercises the
// token.AccessToken == "" branch in refreshConnectionIfNeeded (line 658-660).
func TestRefreshConnectionEmptyAccessTokenMarksFailure(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	conn := nearExpiryOAuthConn(t, s, "empty-token", now.Add(time.Minute))

	// Refresher returns a token with empty AccessToken.
	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{
		Provider:     oauth.ProviderID("codex"),
		AccessToken:  "", // empty!
		RefreshToken: "new-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	_, err := engine.refreshConnectionIfNeeded(context.Background(), "openai", conn)
	if err == nil {
		t.Fatal("empty access token should return error")
	}

	// Connection should be marked as needs_reauth.
	got, fetchErr := s.GetConnection(conn.ID)
	if fetchErr != nil {
		t.Fatalf("GetConnection: %v", fetchErr)
	}
	if !got.NeedsReauth {
		t.Error("NeedsReauth should be true after empty access token")
	}
}

// TestRefreshConnectionUpdateCredentialsError exercises the
// UpdateConnectionCredentials error branch (line 675-676).
func TestRefreshConnectionUpdateCredentialsError(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	conn := nearExpiryOAuthConn(t, s, "update-err", now.Add(time.Minute))

	// Refresher returns a valid token.
	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{
		Provider:     oauth.ProviderID("codex"),
		AccessToken:  "new-valid-token",
		RefreshToken: "new-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	// Close the store so UpdateConnectionCredentials fails.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := engine.refreshConnectionIfNeeded(context.Background(), "openai", conn)
	if err == nil {
		t.Fatal("UpdateConnectionCredentials on closed DB should return error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && err.Error() == "" {
		// any non-nil error is fine
	}
}

// TestRefreshConnectionNoRefresherReturnsConn exercises the !ok (no refresher)
// branch in refreshConnectionIfNeeded (line 647-649).
func TestRefreshConnectionNoRefresherReturnsConn(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	conn := nearExpiryOAuthConn(t, s, "no-refresher", now.Add(time.Minute))

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	// No refresher registered for "codex" → !ok path.

	got, err := engine.refreshConnectionIfNeeded(context.Background(), "openai", conn)
	if err != nil {
		t.Fatalf("refreshConnectionIfNeeded with no refresher should not error: %v", err)
	}
	if got == nil {
		t.Fatal("should return conn when no refresher registered")
	}
}

// TestResolveModelAliasStoreError exercises the non-ErrNotFound store error
// path in resolveModelAlias (line 427-428): when ResolveModelAlias returns
// an error other than ErrNotFound (e.g., closed DB).
func TestResolveModelAliasStoreError(t *testing.T) {
	s := openProxyTestStore(t)
	engine := NewEngine(s)

	// Close the store so ResolveModelAlias returns a DB error.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Dispatch should fail with a wrapped error from resolveModelAlias.
	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("Dispatch with closed DB should return error")
	}
}
