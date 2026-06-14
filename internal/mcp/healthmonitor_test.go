package mcp

import (
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestAccountHealth(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	cases := []struct {
		name      string
		expiresAt int64
		want      string
	}{
		{"future", now.Add(time.Hour).Unix(), "connected"},
		{"expired", now.Add(-time.Minute).Unix(), "expired"},
		{"unknown", 0, "connected"},
	}
	for _, c := range cases {
		got := accountHealth(&store.MCPOAuthAccount{ExpiresAt: c.expiresAt}, now)
		if got != c.want {
			t.Errorf("%s: accountHealth = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAccountsNeedingRefresh(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	lead := 5 * time.Minute
	accts := []*store.MCPOAuthAccount{
		{ID: "fresh", ExpiresAt: now.Add(time.Hour).Unix()},
		{ID: "soon", ExpiresAt: now.Add(2 * time.Minute).Unix()},
		{ID: "expired", ExpiresAt: now.Add(-time.Minute).Unix()},
		{ID: "unknown", ExpiresAt: 0},
	}
	got := accountsNeedingRefresh(accts, now, lead)
	if len(got) != 2 {
		t.Fatalf("got %d accounts needing refresh, want 2: %#v", len(got), got)
	}
	ids := map[string]bool{}
	for _, a := range got {
		ids[a.ID] = true
	}
	if !ids["soon"] || !ids["expired"] {
		t.Fatalf("expected soon+expired, got %v", ids)
	}
}
