package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// TestRefreshExpiringConnectionsSkipsInactive exercises the conn.IsActive==false
// branch and the nil-conn guard in RefreshExpiringConnections.
func TestRefreshExpiringConnectionsSkipsInactive(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)

	// Create an inactive OAuth connection near expiry.
	access := "tok"
	refresh := "rtok"
	exp := now.Add(time.Minute).Unix()
	if err := s.CreateConnection(&store.Connection{
		Provider:     "openai",
		Name:         "inactive-oauth",
		AuthType:     store.AuthTypeOAuth,
		AccessToken:  &access,
		RefreshToken: &refresh,
		ExpiresAt:    &exp,
		IsActive:     false, // inactive → must be skipped
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }

	outcomes := engine.RefreshExpiringConnections(context.Background(), now)
	if len(outcomes) != 0 {
		t.Fatalf("inactive connection should be skipped, got outcomes: %+v", outcomes)
	}
}

// TestRefreshExpiringConnectionsListError exercises the store.ListConnections error
// path: when the store is closed, ListConnections errors and the function returns nil.
func TestRefreshExpiringConnectionsListError(t *testing.T) {
	s := openProxyTestStore(t)
	now := time.Unix(1700000000, 0)

	engine := NewEngine(s)
	engine.now = func() time.Time { return now }

	// Close the store so ListConnections will error.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	outcomes := engine.RefreshExpiringConnections(context.Background(), now)
	if outcomes != nil {
		t.Fatalf("list error should return nil, got %+v", outcomes)
	}
}

// TestFetchTelemetryStatsNilStore exercises the nil-store guard in fetchTelemetryStats.
func TestFetchTelemetryStatsNilStore(t *testing.T) {
	got := fetchTelemetryStats(nil)
	if got != nil {
		t.Fatalf("fetchTelemetryStats(nil) = %v, want nil", got)
	}
}

// TestFetchTelemetryStatsQueryError exercises the error return from
// ProviderModelStats (store closed → error → returns nil map).
func TestFetchTelemetryStatsQueryError(t *testing.T) {
	s := openProxyTestStore(t)
	// Close the store so ProviderModelStats errors.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	got := fetchTelemetryStats(s)
	if got != nil {
		t.Fatalf("fetchTelemetryStats with closed store = %v, want nil", got)
	}
}
