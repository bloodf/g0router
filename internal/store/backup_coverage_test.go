package store

import (
	"testing"
)

func TestExportBackupListErrors(t *testing.T) {
	cases := []struct {
		name      string
		dropTable string
	}{
		{"connections", "connections"},
		{"api_keys", "api_keys"},
		{"combos", "combos"},
		{"model_aliases", "model_aliases"},
		{"pricing_overrides", "pricing_overrides"},
		{"proxy_pools", "proxy_pools"},
		{"disabled_models", "disabled_models"},
		{"custom_models", "custom_models"},
		{"tunnel_config", "tunnel_config"},
		{"chat_sessions", "chat_sessions"},
		{"teams", "teams"},
		{"virtual_keys", "virtual_keys"},
		{"routing_rules", "routing_rules"},
		{"model_limits", "model_limits"},
		{"prompt_templates", "prompt_templates"},
		{"mcp_tool_groups", "mcp_tool_groups"},
		{"alert_channels", "alert_channels"},
		{"feature_flags", "feature_flags"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := openTestStore(t)
			if _, err := s.db.Exec("DROP TABLE " + tc.dropTable); err != nil {
				t.Fatalf("drop table: %v", err)
			}
			_, err := s.ExportBackup()
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestImportBackupUnmarshalErrors(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{"connections", `{"schema_version":"1","connections":"bad"}`},
		{"api_keys", `{"schema_version":"1","api_keys":"bad"}`},
		{"combos", `{"schema_version":"1","combos":"bad"}`},
		{"aliases", `{"schema_version":"1","aliases":"bad"}`},
		{"pricing_overrides", `{"schema_version":"1","pricing_overrides":"bad"}`},
		{"proxy_pools", `{"schema_version":"1","proxy_pools":"bad"}`},
		{"disabled_models", `{"schema_version":"1","disabled_models":"bad"}`},
		{"custom_models", `{"schema_version":"1","custom_models":"bad"}`},
		{"tunnel_configs", `{"schema_version":"1","tunnel_configs":"bad"}`},
		{"chat_sessions", `{"schema_version":"1","chat_sessions":"bad"}`},
		{"teams", `{"schema_version":"1","teams":"bad"}`},
		{"virtual_keys", `{"schema_version":"1","virtual_keys":"bad"}`},
		{"routing_rules", `{"schema_version":"1","routing_rules":"bad"}`},
		{"model_limits", `{"schema_version":"1","model_limits":"bad"}`},
		{"prompt_templates", `{"schema_version":"1","prompt_templates":"bad"}`},
		{"mcp_tool_groups", `{"schema_version":"1","mcp_tool_groups":"bad"}`},
		{"alert_channels", `{"schema_version":"1","alert_channels":"bad"}`},
		{"feature_flags", `{"schema_version":"1","feature_flags":"bad"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := openTestStore(t)
			err := s.ImportBackup([]byte(tc.payload))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestImportBackupRestoreTxErrors(t *testing.T) {
	cases := []struct {
		name      string
		payload   string
		dropTable string
	}{
		{"settings", `{"schema_version":"1","settings":{"require_api_key":false}}`, "settings"},
		{"connections", `{"schema_version":"1","connections":[{"id":"test","provider":"openai"}]}`, "connections"},
		{"api_keys", `{"schema_version":"1","api_keys":[{"id":"test","name":"test","key_hash":"hash","prefix":"pre"}]}`, "api_keys"},
		{"combos", `{"schema_version":"1","combos":[{"id":"test","name":"test","steps":[],"is_active":false,"strategy":"fallback"}]}`, "combos"},
		{"aliases", `{"schema_version":"1","aliases":[{"alias":"test","provider":"openai","model":"gpt-4"}]}`, "model_aliases"},
		{"pricing_overrides", `{"schema_version":"1","pricing_overrides":[{"provider":"openai","model":"gpt-4"}]}`, "pricing_overrides"},
		{"proxy_pools", `{"schema_version":"1","proxy_pools":[{"id":"test","name":"test","protocol":"http","host":"h","port":80}]}`, "proxy_pools"},
		{"disabled_models", `{"schema_version":"1","disabled_models":[{"id":"test","provider":"openai","model":"gpt-3"}]}`, "disabled_models"},
		{"custom_models", `{"schema_version":"1","custom_models":[{"id":"test","provider":"openai","model":"custom-1","display_name":"Custom"}]}`, "custom_models"},
		{"tunnel_configs", `{"schema_version":"1","tunnel_configs":[{"id":"test","type":"cloudflare"}]}`, "tunnel_config"},
		{"chat_sessions", `{"schema_version":"1","chat_sessions":[{"id":"test","title":"test","model":"gpt-4","provider":"openai"}]}`, "chat_sessions"},
		{"teams", `{"schema_version":"1","teams":[{"id":"test","name":"eng"}]}`, "teams"},
		{"virtual_keys", `{"schema_version":"1","virtual_keys":[{"id":"test","name":"v1"}]}`, "virtual_keys"},
		{"routing_rules", `{"schema_version":"1","routing_rules":[{"id":"test","name":"r1"}]}`, "routing_rules"},
		{"model_limits", `{"schema_version":"1","model_limits":[{"id":"test","model":"gpt-4"}]}`, "model_limits"},
		{"prompt_templates", `{"schema_version":"1","prompt_templates":[{"id":"test","name":"tpl1"}]}`, "prompt_templates"},
		{"mcp_tool_groups", `{"schema_version":"1","mcp_tool_groups":[{"id":"test","name":"tg1"}]}`, "mcp_tool_groups"},
		{"alert_channels", `{"schema_version":"1","alert_channels":[{"id":"test","name":"al1","channel_type":"webhook"}]}`, "alert_channels"},
		{"feature_flags", `{"schema_version":"1","feature_flags":[{"id":"test","key":"test"}]}`, "feature_flags"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := openTestStore(t)
			if _, err := s.db.Exec("DROP TABLE " + tc.dropTable); err != nil {
				t.Fatalf("drop table: %v", err)
			}
			err := s.ImportBackup([]byte(tc.payload))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestListAPIKeysForBackupDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	_, err := s.listAPIKeysForBackup()
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestRestoreConnectionTxInsertError(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.db.Exec("DROP TABLE connections"); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	err = restoreConnectionTx(s, tx, Connection{ID: "test", Provider: "openai"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRestoreProxyPoolTxEncryptError(t *testing.T) {
	s := openTestStore(t)
	// No enc key set
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	err = restoreProxyPoolTx(s, tx, ProxyPool{ID: "test", Name: "p1", Protocol: "http", Host: "h", Port: 80, Password: "secret"})
	if err == nil {
		t.Fatal("expected encrypt error")
	}
}

func TestRestoreTunnelConfigTxEncryptError(t *testing.T) {
	s := openTestStore(t)
	// No enc key set
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	err = restoreTunnelConfigTx(s, tx, TunnelConfig{ID: "test", Type: "cloudflare", Config: `{"token":"sekrit"}`})
	if err == nil {
		t.Fatal("expected encrypt error")
	}
}

func TestRestoreAlertChannelTxEncryptError(t *testing.T) {
	s := openTestStore(t)
	// No enc key set
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()
	err = restoreAlertChannelTx(s, tx, AlertChannel{ID: 1, Name: "al1", ChannelType: "webhook", Config: `{"url":"http://example.com"}`})
	if err == nil {
		t.Fatal("expected encrypt error")
	}
}
