package store

import (
	"errors"
	"testing"
)

func TestTunnelConfigUpsertListGetRoundTrip(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")

	cfg := TunnelConfig{
		Type:      "cloudflare",
		IsEnabled: true,
		Config:    `{"tunnel":"cf-123"}`,
		URL:       "https://cf.example.com",
		Status:    "inactive",
		LastError: "",
	}

	if err := s.UpsertTunnelConfig(cfg); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	list, err := s.ListTunnelConfigs()
	if err != nil {
		t.Fatalf("ListTunnelConfigs: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if list[0].Type != "cloudflare" {
		t.Fatalf("type = %q, want cloudflare", list[0].Type)
	}
	if list[0].Config != `{"tunnel":"cf-123"}` {
		t.Fatalf("config = %q, want {\"tunnel\":\"cf-123\"}", list[0].Config)
	}
	if !list[0].IsEnabled {
		t.Fatal("IsEnabled should be true")
	}

	got, err := s.GetTunnelConfig("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if got.Type != "cloudflare" {
		t.Fatalf("type = %q, want cloudflare", got.Type)
	}
	if got.Config != `{"tunnel":"cf-123"}` {
		t.Fatalf("config = %q, want {\"tunnel\":\"cf-123\"}", got.Config)
	}
	if got.URL != "https://cf.example.com" {
		t.Fatalf("url = %q, want https://cf.example.com", got.URL)
	}
}

func TestTunnelConfigEncryptedInDB(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")

	cfg := TunnelConfig{
		Type:   "tailscale",
		Config: `{"auth":"ts-secret"}`,
		Status: "inactive",
	}

	if err := s.UpsertTunnelConfig(cfg); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	var configEnc string
	err := s.db.QueryRow("SELECT config_enc FROM tunnel_config WHERE type = ?", "tailscale").Scan(&configEnc)
	if err != nil {
		t.Fatalf("query db: %v", err)
	}
	if configEnc == "" {
		t.Fatal("config_enc should not be empty")
	}
	if configEnc == `{"auth":"ts-secret"}` {
		t.Fatal("config_enc should be encrypted, not plaintext")
	}

	got, err := s.GetTunnelConfig("tailscale")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if got.Config != `{"auth":"ts-secret"}` {
		t.Fatalf("config decrypted = %q, want {\"auth\":\"ts-secret\"}", got.Config)
	}
}

func TestTunnelConfigUpdateStatus(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")

	cfg := TunnelConfig{
		Type:   "cloudflare",
		Config: `{"tunnel":"cf-123"}`,
		Status: "inactive",
	}

	if err := s.UpsertTunnelConfig(cfg); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	if err := s.UpdateTunnelStatus("cloudflare", "active", ""); err != nil {
		t.Fatalf("UpdateTunnelStatus: %v", err)
	}

	got, err := s.GetTunnelConfig("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if got.Status != "active" {
		t.Fatalf("status = %q, want active", got.Status)
	}
	if got.LastError != "" {
		t.Fatalf("last_error = %q, want empty", got.LastError)
	}
	if got.Config != `{"tunnel":"cf-123"}` {
		t.Fatalf("config = %q, want unchanged", got.Config)
	}

	if err := s.UpdateTunnelStatus("cloudflare", "error", "connection refused"); err != nil {
		t.Fatalf("UpdateTunnelStatus: %v", err)
	}

	got, err = s.GetTunnelConfig("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if got.Status != "error" {
		t.Fatalf("status = %q, want error", got.Status)
	}
	if got.LastError != "connection refused" {
		t.Fatalf("last_error = %q, want connection refused", got.LastError)
	}
}

func TestTunnelConfigGetNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")

	_, err := s.GetTunnelConfig("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
