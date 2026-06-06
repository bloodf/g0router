package store

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExportBackupNoSecrets(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-backup-key")

	// Create data with secrets
	conn := &Connection{
		Provider:    "openai",
		AuthType:    "oauth",
		AccessToken: stringPtr("tok"),
		RefreshToken: stringPtr("ref"),
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if _, err := s.CreateProxyPool(ProxyPool{Name: "p1", Protocol: "http", Host: "h", Port: 80, Password: "secret"}); err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	if _, err := s.CreateAlertChannel("ops", "webhook", `{"url":"https://example.com","token":"sekrit"}`, []string{"quota_depleted"}, true); err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	var export map[string]any
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("unmarshal backup: %v", err)
	}

	if export["schema_version"] != "1" {
		t.Fatalf("schema_version = %v", export["schema_version"])
	}

	redacted, ok := export["redacted_fields"].([]any)
	if !ok || len(redacted) == 0 {
		t.Fatalf("expected redacted_fields, got %v", export["redacted_fields"])
	}

	// Check connections have null secrets
	connections, ok := export["connections"].([]any)
	if !ok || len(connections) == 0 {
		t.Fatal("expected connections in backup")
	}
	c := connections[0].(map[string]any)
	if c["access_token"] != nil {
		t.Fatalf("access_token should be null, got %v", c["access_token"])
	}
	if c["refresh_token"] != nil {
		t.Fatalf("refresh_token should be null, got %v", c["refresh_token"])
	}

	// Check proxy pools have null password
	pools, ok := export["proxy_pools"].([]any)
	if !ok || len(pools) == 0 {
		t.Fatal("expected proxy_pools in backup")
	}
	p := pools[0].(map[string]any)
	if p["password"] != nil {
		t.Fatalf("password should be null, got %v", p["password"])
	}

	// Check alert channels have null config
	channels, ok := export["alert_channels"].([]any)
	if !ok || len(channels) == 0 {
		t.Fatal("expected alert_channels in backup")
	}
	ch := channels[0].(map[string]any)
	if ch["config"] != nil {
		t.Fatalf("config should be null, got %v", ch["config"])
	}
}

func TestImportBackupPreservesSecrets(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-backup-key")

	// Create initial data with secrets
	conn := &Connection{
		Provider:     "openai",
		AuthType:     "oauth",
		AccessToken:  stringPtr("original-tok"),
		RefreshToken: stringPtr("original-ref"),
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if _, err := s.CreateProxyPool(ProxyPool{Name: "p1", Protocol: "http", Host: "h", Port: 80, Password: "original-secret"}); err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	if _, err := s.CreateAlertChannel("ops", "webhook", `{"url":"https://example.com","token":"original-sekrit"}`, []string{"quota_depleted"}, true); err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}

	// Export backup
	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	// Modify some non-secret fields in the backup JSON
	var backup map[string]any
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	connections := backup["connections"].([]any)
	c := connections[0].(map[string]any)
	c["name"] = "updated-name"

	modified, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Restore
	if err := s.ImportBackup(modified); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	// Verify name updated but secrets preserved
	conns, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].Name != "updated-name" {
		t.Fatalf("name = %q, want updated-name", conns[0].Name)
	}
	if conns[0].AccessToken == nil || *conns[0].AccessToken != "original-tok" {
		t.Fatalf("access_token = %v, want original-tok", conns[0].AccessToken)
	}
	if conns[0].RefreshToken == nil || *conns[0].RefreshToken != "original-ref" {
		t.Fatalf("refresh_token = %v, want original-ref", conns[0].RefreshToken)
	}

	pools, err := s.ListProxyPools()
	if err != nil {
		t.Fatalf("ListProxyPools: %v", err)
	}
	if len(pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(pools))
	}
	if pools[0].Password != "original-secret" {
		t.Fatalf("password = %q, want original-secret", pools[0].Password)
	}
}

func TestImportBackupRejectsBadSchema(t *testing.T) {
	s := openTestStore(t)

	bad := `{"schema_version":"99","connections":[]}`
	err := s.ImportBackup([]byte(bad))
	if err == nil || !strings.Contains(err.Error(), "schema version") {
		t.Fatalf("expected schema version error, got %v", err)
	}
}

func TestImportBackupRejectsUnknownShape(t *testing.T) {
	s := openTestStore(t)

	bad := `{"schema_version":"1","unknown_field":"xyz"}`
	err := s.ImportBackup([]byte(bad))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}
