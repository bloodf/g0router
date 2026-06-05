package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestRefreshErrorMarksConnectionNeedsReauth(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	oldExpires := now.Add(time.Minute).Unix()
	token := "old-access"
	refresh := "old-refresh"

	conn := &store.Connection{
		Provider:     "openai",
		Name:         "oauth",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &token,
		RefreshToken: &refresh,
		ExpiresAt:    &oldExpires,
		IsActive:     true,
		ProviderSpecificData: map[string]any{
			"oauth_provider": "codex",
		},
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-1"}}
	refresher := &fakeOAuthRefresher{err: errors.New("invalid_grant")}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("expected error from failed refresh")
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if !got.NeedsReauth {
		t.Error("NeedsReauth should be true after refresh error")
	}
	if got.LastRefreshError == nil || *got.LastRefreshError == "" {
		t.Error("LastRefreshError should be set after refresh error")
	}
	// Must not contain token-like blobs.
	if got.LastRefreshError != nil {
		for _, tok := range []string{"old-refresh", "old-access"} {
			if containsSubstr(*got.LastRefreshError, tok) {
				t.Errorf("LastRefreshError contains secret %q: %s", tok, *got.LastRefreshError)
			}
		}
	}
}

func TestRefreshSuccessClearsNeedsReauth(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	oldExpires := now.Add(time.Minute).Unix()
	token := "old-access"
	refresh := "old-refresh"

	conn := &store.Connection{
		Provider:     "openai",
		Name:         "oauth",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &token,
		RefreshToken: &refresh,
		ExpiresAt:    &oldExpires,
		IsActive:     true,
		NeedsReauth:  true,
		ProviderSpecificData: map[string]any{
			"oauth_provider": "codex",
		},
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	reason := "previous_error"
	if err := s.MarkConnectionRefreshFailure(conn.ID, reason); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-1"}}
	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{
		Provider:     oauth.ProviderID("codex"),
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(openAI)
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"}); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.NeedsReauth {
		t.Error("NeedsReauth should be false after successful refresh")
	}
	if got.LastRefreshError != nil {
		t.Errorf("LastRefreshError should be nil after successful refresh, got %q", *got.LastRefreshError)
	}
}

func TestSanitizeRefreshReasonRedactsTokenLike(t *testing.T) {
	longToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature"
	reason := "refresh failed: " + longToken
	got := sanitizeRefreshReason(errors.New(reason))
	if containsSubstr(got, longToken) {
		t.Errorf("sanitized reason still contains token-like blob: %q", got)
	}
}

func TestSanitizeRefreshReasonPreservesOAuthCode(t *testing.T) {
	got := sanitizeRefreshReason(errors.New("invalid_grant"))
	if got != "invalid_grant" {
		t.Errorf("short error code should be preserved, got %q", got)
	}
}

func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
