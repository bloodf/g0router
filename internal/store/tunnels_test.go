package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func newTunnelTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestTunnelUpsertGetList(t *testing.T) {
	st := newTunnelTestStore(t)

	// Unknown tunnel → ErrNotFound.
	if _, err := st.GetTunnel("cloudflare"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetTunnel(empty) = %v, want ErrNotFound", err)
	}

	if err := st.UpsertTunnel(Tunnel{
		Type:      "cloudflare",
		IsEnabled: true,
		Status:    "active",
		URL:       "https://brave-tree-1234.trycloudflare.com",
		Token:     "cf-named-token-secret",
		Mode:      "named",
	}); err != nil {
		t.Fatalf("UpsertTunnel: %v", err)
	}

	got, err := st.GetTunnel("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnel: %v", err)
	}
	if got.Type != "cloudflare" || !got.IsEnabled || got.Status != "active" {
		t.Fatalf("got = %+v", got)
	}
	if got.URL != "https://brave-tree-1234.trycloudflare.com" || got.Mode != "named" {
		t.Fatalf("got = %+v", got)
	}
	if got.Token != "cf-named-token-secret" {
		t.Fatalf("token did not round-trip: %q", got.Token)
	}
	if got.UpdatedAt == 0 {
		t.Fatalf("missing updated_at: %+v", got)
	}

	// Upsert again (conflict on type) updates in place.
	if err := st.UpsertTunnel(Tunnel{Type: "cloudflare", IsEnabled: false, Status: "inactive", URL: "", Token: "", Mode: "quick"}); err != nil {
		t.Fatalf("UpsertTunnel (update): %v", err)
	}
	got, err = st.GetTunnel("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnel after update: %v", err)
	}
	if got.IsEnabled || got.Status != "inactive" || got.Mode != "quick" {
		t.Fatalf("update not persisted: %+v", got)
	}

	// Seed tailscale, then list returns both in deterministic order.
	if err := st.UpsertTunnel(Tunnel{Type: "tailscale", Status: "inactive"}); err != nil {
		t.Fatalf("UpsertTunnel tailscale: %v", err)
	}
	list, err := st.ListTunnels()
	if err != nil {
		t.Fatalf("ListTunnels: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(list))
	}
	if list[0].Type != "cloudflare" || list[1].Type != "tailscale" {
		t.Fatalf("deterministic order broken: %s, %s", list[0].Type, list[1].Type)
	}
}

func TestTunnelTokenEncryptedAtRest(t *testing.T) {
	st := newTunnelTestStore(t)
	if err := st.UpsertTunnel(Tunnel{Type: "cloudflare", Token: "topsecret-token"}); err != nil {
		t.Fatalf("UpsertTunnel: %v", err)
	}
	var raw string
	if err := st.DB().QueryRow("SELECT token_enc FROM tunnels WHERE type = ?", "cloudflare").Scan(&raw); err != nil {
		t.Fatalf("read raw token_enc: %v", err)
	}
	if raw == "" {
		t.Fatalf("token_enc empty for a non-empty token")
	}
	if raw == "topsecret-token" {
		t.Fatalf("token stored in cleartext: %q", raw)
	}
}

func TestSetTunnelState(t *testing.T) {
	st := newTunnelTestStore(t)
	if err := st.UpsertTunnel(Tunnel{Type: "cloudflare", Token: "keepme", Mode: "named"}); err != nil {
		t.Fatalf("UpsertTunnel: %v", err)
	}

	if err := st.SetTunnelState("cloudflare", "active", "https://x.trycloudflare.com", "", true); err != nil {
		t.Fatalf("SetTunnelState: %v", err)
	}
	got, err := st.GetTunnel("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnel: %v", err)
	}
	if !got.IsEnabled || got.Status != "active" || got.URL != "https://x.trycloudflare.com" || got.LastError != "" {
		t.Fatalf("active state not persisted: %+v", got)
	}
	// Token + mode preserved across a state transition.
	if got.Token != "keepme" || got.Mode != "named" {
		t.Fatalf("token/mode clobbered by SetTunnelState: %+v", got)
	}

	if err := st.SetTunnelState("cloudflare", "error", "", "spawn failed", true); err != nil {
		t.Fatalf("SetTunnelState error: %v", err)
	}
	got, _ = st.GetTunnel("cloudflare")
	if got.Status != "error" || got.LastError != "spawn failed" || !got.IsEnabled {
		t.Fatalf("error state not persisted: %+v", got)
	}

	if err := st.SetTunnelState("cloudflare", "inactive", "", "", false); err != nil {
		t.Fatalf("SetTunnelState inactive: %v", err)
	}
	got, _ = st.GetTunnel("cloudflare")
	if got.IsEnabled || got.Status != "inactive" || got.URL != "" {
		t.Fatalf("inactive state not persisted: %+v", got)
	}
}
