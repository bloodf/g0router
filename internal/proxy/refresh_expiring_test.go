package proxy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
)

func nearExpiryOAuthConn(t *testing.T, s *store.Store, name string, expires time.Time) *store.Connection {
	t.Helper()
	access := "old-access"
	refresh := "old-refresh"
	exp := expires.Unix()
	conn := &store.Connection{
		Provider:     "openai",
		Name:         name,
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &access,
		RefreshToken: &refresh,
		ExpiresAt:    &exp,
		IsActive:     true,
		ProviderSpecificData: map[string]any{
			"oauth_provider": "codex",
		},
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	return conn
}

func TestRefreshExpiringConnectionsRefreshesNearExpiry(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	conn := nearExpiryOAuthConn(t, s, "oauth", now.Add(time.Minute))

	refresher := &fakeOAuthRefresher{token: oauth.TokenResult{
		Provider:     oauth.ProviderID("codex"),
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	outcomes := engine.RefreshExpiringConnections(context.Background(), now)
	if len(outcomes) != 1 {
		t.Fatalf("outcomes = %d, want 1", len(outcomes))
	}
	if !outcomes[0].Refreshed || outcomes[0].Failed {
		t.Errorf("outcome = %+v, want Refreshed", outcomes[0])
	}
	if outcomes[0].ConnectionID != conn.ID {
		t.Errorf("ConnectionID = %q, want %q", outcomes[0].ConnectionID, conn.ID)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.NeedsReauth {
		t.Error("NeedsReauth should be false after successful proactive refresh")
	}
	if got.AccessToken == nil || *got.AccessToken != "new-access" {
		t.Errorf("AccessToken not persisted, got %v", got.AccessToken)
	}
}

func TestRefreshExpiringConnectionsMarksFailure(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)
	conn := nearExpiryOAuthConn(t, s, "oauth", now.Add(time.Minute))

	refresher := &fakeOAuthRefresher{err: errors.New("invalid_grant")}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.RegisterOAuthRefresher(oauth.ProviderID("codex"), refresher)

	outcomes := engine.RefreshExpiringConnections(context.Background(), now)
	if len(outcomes) != 1 {
		t.Fatalf("outcomes = %d, want 1", len(outcomes))
	}
	if outcomes[0].Refreshed || !outcomes[0].Failed {
		t.Errorf("outcome = %+v, want Failed", outcomes[0])
	}
	if outcomes[0].Reason == "" {
		t.Error("failed outcome should carry a reason")
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if !got.NeedsReauth {
		t.Error("NeedsReauth should be true after failed proactive refresh")
	}
}

func TestRefreshExpiringConnectionsSkipsHealthyAndNoRefresher(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)

	// Healthy connection far from expiry: skipped.
	nearExpiryOAuthConn(t, s, "healthy", now.Add(time.Hour))
	// Near expiry but no registered refresher: skipped.
	nearExpiryOAuthConn(t, s, "no-refresher", now.Add(time.Minute))

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }

	outcomes := engine.RefreshExpiringConnections(context.Background(), now)
	if len(outcomes) != 0 {
		t.Fatalf("outcomes = %+v, want none (healthy + no refresher skipped)", outcomes)
	}
}
