package store

import (
	"errors"
	"testing"
)

func TestUpsertTunnelConfigEncryptError(t *testing.T) {
	s := openTestStore(t)
	// Do NOT set encryption key

	cfg := TunnelConfig{
		Type:   "cloudflare",
		Config: `{"tunnel":"cf-123"}`,
		Status: "inactive",
	}

	err := s.UpsertTunnelConfig(cfg)
	if err == nil {
		t.Fatal("expected encrypt error")
	}
}

func TestUpsertTunnelConfigExecError(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")
	s.Close()

	cfg := TunnelConfig{
		Type:   "cloudflare",
		Config: `{"tunnel":"cf-123"}`,
		Status: "inactive",
	}

	err := s.UpsertTunnelConfig(cfg)
	if err == nil {
		t.Fatal("expected exec error")
	}
}

func TestListTunnelConfigsQueryError(t *testing.T) {
	s := openTestStore(t)
	s.Close()

	_, err := s.ListTunnelConfigs()
	if err == nil {
		t.Fatal("expected query error")
	}
}

func TestUpdateTunnelStatusNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")

	err := s.UpdateTunnelStatus("nonexistent", "active", "")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateTunnelStatusExecError(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")
	s.Close()

	err := s.UpdateTunnelStatus("cloudflare", "active", "")
	if err == nil {
		t.Fatal("expected exec error")
	}
}

func TestScanTunnelConfigDecryptError(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-tunnels")

	// Insert a row with invalid config_enc directly
	_, err := s.db.Exec(
		`INSERT INTO tunnel_config (type, is_enabled, config_enc, url, status, last_error)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"cloudflare", 1, "invalid-enc-data", "http://example.com", "active", "",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, err = s.GetTunnelConfig("cloudflare")
	if err == nil {
		t.Fatal("expected decrypt error")
	}
}
