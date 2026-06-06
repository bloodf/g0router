package store

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExportBackupNoSecrets(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-backup-key")

	// Create data with secrets
	conn := &Connection{
		Provider:     "openai",
		AuthType:     "oauth",
		AccessToken:  stringPtr("tok"),
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

func TestBackupRestoreRoundTripAllTables(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-backup-key")

	// Seed all tables with data
	settings := defaultSettings()
	settings.RequireAPIKey = false
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	conn := &Connection{Provider: "openai", AuthType: "api_key", APIKey: stringPtr("secret")}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	if _, _, err := s.CreateAPIKey("test-key", "secret"); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}

	if err := s.CreateCombo(&Combo{Name: "test-combo", Steps: []ComboStep{{Provider: "openai", Model: "gpt-4"}}, Strategy: "fallback"}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	if err := s.SetModelAlias(ModelAlias{Alias: "gptx", Provider: "openai", Model: "gpt-4"}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}

	if err := s.SetPricingOverride(PricingOverride{Provider: "openai", Model: "gpt-4", InputCostPerToken: 0.001, OutputCostPerToken: 0.002}); err != nil {
		t.Fatalf("SetPricingOverride: %v", err)
	}

	if _, err := s.CreateProxyPool(ProxyPool{Name: "p1", Protocol: "http", Host: "h", Port: 80, Password: "pw"}); err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	if _, err := s.CreateDisabledModel("openai", "gpt-3"); err != nil {
		t.Fatalf("CreateDisabledModel: %v", err)
	}

	if _, err := s.CreateCustomModel("openai", "custom-1", "Custom"); err != nil {
		t.Fatalf("CreateCustomModel: %v", err)
	}

	if err := s.UpsertTunnelConfig(TunnelConfig{Type: "cloudflare", IsEnabled: true, Config: `{"token":"sekrit"}`}); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	if _, err := s.CreateChatSession("chat1", "gpt-4", "openai", `[{"role":"user","content":"hi"}]`); err != nil {
		t.Fatalf("CreateChatSession: %v", err)
	}

	if _, err := s.CreateTeam("eng", nil, "monthly", nil); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	if _, _, err := s.CreateVirtualKey("v1", nil, nil, "monthly", nil, nil, ""); err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	tm := "gpt-4"
	if _, err := s.CreateRoutingRule("r1", 1, "model", "equals", "gpt-4", "openai", &tm); err != nil {
		t.Fatalf("CreateRoutingRule: %v", err)
	}

	maxTok := 4096
	maxRPM := 60
	if _, err := s.CreateModelLimit("gpt-4", &maxTok, &maxRPM, nil); err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	if _, err := s.CreatePromptTemplate("tpl1", "sys", []string{"gpt-4"}, true); err != nil {
		t.Fatalf("CreatePromptTemplate: %v", err)
	}

	if _, err := s.CreateMCPToolGroup("tg1", []string{"t1"}, true); err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}

	if _, err := s.CreateAlertChannel("al1", "webhook", `{"url":"http://example.com"}`, []string{"quota_depleted"}, true); err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}

	// Export
	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	// Restore to a fresh store
	s2 := openTestStore(t)
	s2.SetEncKey("test-backup-key")
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	// Verify key data restored
	conns, err := s2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}

	keys, err := s2.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 api key, got %d", len(keys))
	}

	combos, err := s2.ListCombos()
	if err != nil {
		t.Fatalf("ListCombos: %v", err)
	}
	if len(combos) != 1 {
		t.Fatalf("expected 1 combo, got %d", len(combos))
	}

	aliases, err := s2.ListModelAliases()
	if err != nil {
		t.Fatalf("ListModelAliases: %v", err)
	}
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	pricing, err := s2.ListPricingOverrides()
	if err != nil {
		t.Fatalf("ListPricingOverrides: %v", err)
	}
	if len(pricing) != 1 {
		t.Fatalf("expected 1 pricing override, got %d", len(pricing))
	}

	pools, err := s2.ListProxyPools()
	if err != nil {
		t.Fatalf("ListProxyPools: %v", err)
	}
	if len(pools) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(pools))
	}

	disabled, err := s2.ListDisabledModels()
	if err != nil {
		t.Fatalf("ListDisabledModels: %v", err)
	}
	if len(disabled) != 1 {
		t.Fatalf("expected 1 disabled model, got %d", len(disabled))
	}

	custom, err := s2.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels: %v", err)
	}
	if len(custom) != 1 {
		t.Fatalf("expected 1 custom model, got %d", len(custom))
	}

	tunnels, err := s2.ListTunnelConfigs()
	if err != nil {
		t.Fatalf("ListTunnelConfigs: %v", err)
	}
	if len(tunnels) != 1 {
		t.Fatalf("expected 1 tunnel config, got %d", len(tunnels))
	}

	chats, err := s2.ListChatSessions()
	if err != nil {
		t.Fatalf("ListChatSessions: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("expected 1 chat session, got %d", len(chats))
	}

	teams, err := s2.ListTeams()
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}

	vkeys, err := s2.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	if len(vkeys) != 1 {
		t.Fatalf("expected 1 virtual key, got %d", len(vkeys))
	}

	rules, err := s2.ListRoutingRules()
	if err != nil {
		t.Fatalf("ListRoutingRules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 routing rule, got %d", len(rules))
	}

	limits, err := s2.ListModelLimits()
	if err != nil {
		t.Fatalf("ListModelLimits: %v", err)
	}
	if len(limits) != 1 {
		t.Fatalf("expected 1 model limit, got %d", len(limits))
	}

	templates, err := s2.ListPromptTemplates()
	if err != nil {
		t.Fatalf("ListPromptTemplates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected 1 prompt template, got %d", len(templates))
	}

	groups, err := s2.ListMCPToolGroups()
	if err != nil {
		t.Fatalf("ListMCPToolGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 tool group, got %d", len(groups))
	}

	channels, err := s2.ListAlertChannels()
	if err != nil {
		t.Fatalf("ListAlertChannels: %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("expected 1 alert channel, got %d", len(channels))
	}

	flags, err := s2.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	if len(flags) != 5 {
		t.Fatalf("expected 5 feature flags, got %d", len(flags))
	}
}

func TestBackupRestoreNewRows(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-backup-key")

	// Create data and export
	conn := &Connection{Provider: "openai", AuthType: "noauth"}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	// Restore to fresh store (no existing rows)
	s2 := openTestStore(t)
	s2.SetEncKey("test-backup-key")
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	conns, err := s2.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
}

func TestBackupRestoreProviderSpecificData(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{
		Provider:             "openai",
		AuthType:             "oauth",
		ProviderSpecificData: map[string]any{"extra": "data"},
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	// Restore with provider_specific_data null (should preserve existing)
	var backup map[string]any
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	connections := backup["connections"].([]any)
	c := connections[0].(map[string]any)
	c["provider_specific_data"] = nil

	modified, err := json.Marshal(backup)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if err := s.ImportBackup(modified); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	conns, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if conns[0].ProviderSpecificData == nil || conns[0].ProviderSpecificData["extra"] != "data" {
		t.Fatalf("provider_specific_data not preserved: %v", conns[0].ProviderSpecificData)
	}
}

func TestBackupRestoreTeamAndVirtualKeyTimeFormats(t *testing.T) {
	s := openTestStore(t)

	now := time.Now().UTC()
	team, err := s.CreateTeam("eng", nil, "monthly", nil)
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	team.BudgetResetAt = &now
	// Update the reset_at in DB directly
	if _, err := s.db.Exec("UPDATE teams SET budget_reset_at = ? WHERE id = ?", now.Format(time.RFC3339), team.ID); err != nil {
		t.Fatalf("update team: %v", err)
	}

	_, _, err = s.CreateVirtualKey("v1", nil, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	teams, err := s2.ListTeams()
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(teams))
	}
}

func TestBackupRestoreFeatureFlagEnabled(t *testing.T) {
	s := openTestStore(t)

	flags, err := s.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	if err := s.ToggleFeatureFlag(flags[0].ID, true); err != nil {
		t.Fatalf("ToggleFeatureFlag: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	flags2, err := s2.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	found := false
	for _, f := range flags2 {
		if f.Key == flags[0].Key && f.Enabled {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("feature flag enabled state not restored")
	}
}

func TestJoinScopes(t *testing.T) {
	if got := joinScopes(nil); got != "" {
		t.Fatalf("joinScopes(nil) = %q, want empty", got)
	}
	if got := joinScopes([]string{}); got != "" {
		t.Fatalf("joinScopes([]) = %q, want empty", got)
	}
	if got := joinScopes([]string{"a", "b", "c"}); got != "a,b,c" {
		t.Fatalf("joinScopes = %q, want a,b,c", got)
	}
}

func TestCoalesceStringPtr(t *testing.T) {
	b := "fallback"
	if got := coalesceStringPtr(nil, &b); got == nil || *got != "fallback" {
		t.Fatalf("coalesceStringPtr(nil, &b) = %v, want fallback", got)
	}
	empty := ""
	if got := coalesceStringPtr(&empty, &b); got == nil || *got != "fallback" {
		t.Fatalf("coalesceStringPtr(&empty, &b) = %v, want fallback", got)
	}
	a := "primary"
	if got := coalesceStringPtr(&a, &b); got == nil || *got != "primary" {
		t.Fatalf("coalesceStringPtr(&a, &b) = %v, want primary", got)
	}
}

func TestImportBackupParseError(t *testing.T) {
	s := openTestStore(t)
	err := s.ImportBackup([]byte(`{not json`))
	if err == nil || !strings.Contains(err.Error(), "parse backup") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestImportBackupSettingsError(t *testing.T) {
	s := openTestStore(t)
	data := []byte(`{"schema_version":"1","settings":"bad"}`)
	err := s.ImportBackup(data)
	if err == nil || !strings.Contains(err.Error(), "restore settings") {
		t.Fatalf("expected restore settings error, got %v", err)
	}
}

func TestBackupRestoreAPIKeyWithScopes(t *testing.T) {
	s := openTestStore(t)

	// Insert an API key directly with all fields populated
	_, err := s.db.Exec(`INSERT INTO api_keys (id, name, key_hash, prefix, is_active, last_used_at, created_at, expires_at, scopes, rate_limit_rpm, rate_limit_tpm, daily_spend_cap_usd)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"key-1", "test", "hash", "pre", 1, "2024-01-01", "2024-01-01", 1704067200, `["read","write"]`, 100, 200, 50.0)
	if err != nil {
		t.Fatalf("insert api key: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	keys, err := s2.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	k := keys[0]
	if len(k.Scopes) != 2 || k.Scopes[0] != "read" || k.Scopes[1] != "write" {
		t.Fatalf("scopes = %v", k.Scopes)
	}
	if k.RateLimitRPM == nil || *k.RateLimitRPM != 100 {
		t.Fatalf("rpm = %v", k.RateLimitRPM)
	}
	if k.RateLimitTPM == nil || *k.RateLimitTPM != 200 {
		t.Fatalf("tpm = %v", k.RateLimitTPM)
	}
	if k.DailySpendCapUSD == nil || *k.DailySpendCapUSD != 50.0 {
		t.Fatalf("cap = %v", k.DailySpendCapUSD)
	}
	if k.ExpiresAt == nil || *k.ExpiresAt != 1704067200 {
		t.Fatalf("expires_at = %v", k.ExpiresAt)
	}
}

func TestBackupRestoreVirtualKeyWithBudgetReset(t *testing.T) {
	s := openTestStore(t)

	now := time.Now().UTC()
	vk, _, err := s.CreateVirtualKey("v1", nil, nil, "monthly", nil, nil, "")
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	_, err = s.db.Exec("UPDATE virtual_keys SET budget_reset_at = ? WHERE id = ?", now.Format(time.RFC3339), vk.ID)
	if err != nil {
		t.Fatalf("update virtual key: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	vkeys, err := s2.ListVirtualKeys()
	if err != nil {
		t.Fatalf("ListVirtualKeys: %v", err)
	}
	if len(vkeys) != 1 {
		t.Fatalf("expected 1 virtual key, got %d", len(vkeys))
	}
}

func TestBackupRestoreModelLimitWithAllowedKeyIDs(t *testing.T) {
	s := openTestStore(t)

	maxTok := 1000
	maxRPM := 60
	allowed := []string{"key1", "key2"}
	if _, err := s.CreateModelLimit("gpt-4", &maxTok, &maxRPM, allowed); err != nil {
		t.Fatalf("CreateModelLimit: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	limits, err := s2.ListModelLimits()
	if err != nil {
		t.Fatalf("ListModelLimits: %v", err)
	}
	if len(limits) != 1 {
		t.Fatalf("expected 1 limit, got %d", len(limits))
	}
	if len(limits[0].AllowedKeyIDs) != 2 || limits[0].AllowedKeyIDs[0] != "key1" {
		t.Fatalf("allowed_key_ids = %v", limits[0].AllowedKeyIDs)
	}
}

func TestBackupRestorePromptTemplateWithModels(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreatePromptTemplate("tpl1", "sys", []string{"gpt-4", "gpt-3.5"}, true); err != nil {
		t.Fatalf("CreatePromptTemplate: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	tpls, err := s2.ListPromptTemplates()
	if err != nil {
		t.Fatalf("ListPromptTemplates: %v", err)
	}
	if len(tpls) != 1 {
		t.Fatalf("expected 1 template, got %d", len(tpls))
	}
	if len(tpls[0].Models) != 2 || tpls[0].Models[0] != "gpt-4" {
		t.Fatalf("models = %v", tpls[0].Models)
	}
}

func TestBackupRestoreMCPToolGroupWithTools(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateMCPToolGroup("tg1", []string{"t1", "t2"}, true); err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}

	data, err := s.ExportBackup()
	if err != nil {
		t.Fatalf("ExportBackup: %v", err)
	}

	s2 := openTestStore(t)
	if err := s2.ImportBackup(data); err != nil {
		t.Fatalf("ImportBackup: %v", err)
	}

	groups, err := s2.ListMCPToolGroups()
	if err != nil {
		t.Fatalf("ListMCPToolGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].ToolIDs) != 2 || groups[0].ToolIDs[0] != "t1" {
		t.Fatalf("tool_ids = %v", groups[0].ToolIDs)
	}
}

func TestExportBackupDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	_, err := s.ExportBackup()
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestImportBackupDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	err := s.ImportBackup([]byte(`{"schema_version":"1"}`))
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}
