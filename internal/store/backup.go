package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

const backupSchemaVersion = "1"

// backupProxyPool is the public backup shape (password nulled).
type backupProxyPool struct {
	ProxyPool
	Password *string `json:"password"`
}

// backupTunnelConfig is the public backup shape (config nulled).
type backupTunnelConfig struct {
	TunnelConfig
	Config *string `json:"config"`
}

// backupAlertChannel is the public backup shape (config nulled).
type backupAlertChannel struct {
	AlertChannel
	Config *string `json:"config"`
}

// ExportBackup returns a JSON export of all configuration tables with secrets
// replaced by null placeholders and a redacted_fields manifest.
func (s *Store) ExportBackup() ([]byte, error) {
	settings, err := s.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("export settings: %w", err)
	}

	conns, err := s.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("export connections: %w", err)
	}

	keys, err := s.listAPIKeysForBackup()
	if err != nil {
		return nil, fmt.Errorf("export api keys: %w", err)
	}

	combosList, err := s.ListCombos()
	if err != nil {
		return nil, fmt.Errorf("export combos: %w", err)
	}

	aliases, err := s.ListModelAliases()
	if err != nil {
		return nil, fmt.Errorf("export aliases: %w", err)
	}

	pricing, err := s.ListPricingOverrides()
	if err != nil {
		return nil, fmt.Errorf("export pricing: %w", err)
	}

	pools, err := s.ListProxyPools()
	if err != nil {
		return nil, fmt.Errorf("export proxy pools: %w", err)
	}

	disabled, err := s.ListDisabledModels()
	if err != nil {
		return nil, fmt.Errorf("export disabled models: %w", err)
	}

	custom, err := s.ListCustomModels()
	if err != nil {
		return nil, fmt.Errorf("export custom models: %w", err)
	}

	tunnels, err := s.ListTunnelConfigs()
	if err != nil {
		return nil, fmt.Errorf("export tunnel configs: %w", err)
	}

	chats, err := s.ListChatSessions()
	if err != nil {
		return nil, fmt.Errorf("export chat sessions: %w", err)
	}

	teams, err := s.ListTeams()
	if err != nil {
		return nil, fmt.Errorf("export teams: %w", err)
	}

	vkeys, err := s.ListVirtualKeys()
	if err != nil {
		return nil, fmt.Errorf("export virtual keys: %w", err)
	}

	rules, err := s.ListRoutingRules()
	if err != nil {
		return nil, fmt.Errorf("export routing rules: %w", err)
	}

	limits, err := s.ListModelLimits()
	if err != nil {
		return nil, fmt.Errorf("export model limits: %w", err)
	}

	templates, err := s.ListPromptTemplates()
	if err != nil {
		return nil, fmt.Errorf("export prompt templates: %w", err)
	}

	groups, err := s.ListMCPToolGroups()
	if err != nil {
		return nil, fmt.Errorf("export mcp tool groups: %w", err)
	}

	channels, err := s.ListAlertChannels()
	if err != nil {
		return nil, fmt.Errorf("export alert channels: %w", err)
	}

	flags, err := s.ListFeatureFlags()
	if err != nil {
		return nil, fmt.Errorf("export feature flags: %w", err)
	}

	var redacted []string

	// Build connections with nulled secrets
	var bc []Connection
	for _, c := range conns {
		cpy := *c
		cpy.AccessToken = nil
		cpy.RefreshToken = nil
		cpy.APIKey = nil
		cpy.ProviderSpecificData = nil
		bc = append(bc, cpy)
		redacted = append(redacted, "connections[].access_token", "connections[].refresh_token", "connections[].api_key", "connections[].provider_specific_data")
	}

	// Build combos as values
	var combos []Combo
	for _, c := range combosList {
		combos = append(combos, *c)
	}

	// Build proxy pools with nulled password
	var bp []backupProxyPool
	for _, p := range pools {
		bp = append(bp, backupProxyPool{ProxyPool: p, Password: nil})
		redacted = append(redacted, "proxy_pools[].password")
	}

	// Build tunnel configs with nulled config
	var bt []backupTunnelConfig
	for _, t := range tunnels {
		bt = append(bt, backupTunnelConfig{TunnelConfig: t, Config: nil})
		redacted = append(redacted, "tunnel_configs[].config")
	}

	// Build alert channels with nulled config
	var ba []backupAlertChannel
	for _, ch := range channels {
		ba = append(ba, backupAlertChannel{AlertChannel: ch, Config: nil})
		redacted = append(redacted, "alert_channels[].config")
	}

	// Deduplicate redacted fields
	redacted = uniqStrings(redacted)

	m := map[string]any{
		"schema_version":    backupSchemaVersion,
		"redacted_fields":   redacted,
		"settings":          settings,
		"connections":       bc,
		"api_keys":          keys,
		"combos":            combos,
		"aliases":           aliases,
		"pricing_overrides": pricing,
		"proxy_pools":       bp,
		"disabled_models":   disabled,
		"custom_models":     custom,
		"tunnel_configs":    bt,
		"chat_sessions":     chats,
		"teams":             teams,
		"virtual_keys":      vkeys,
		"routing_rules":     rules,
		"model_limits":      limits,
		"prompt_templates":  templates,
		"mcp_tool_groups":   groups,
		"alert_channels":    ba,
		"feature_flags":     flags,
	}

	return json.MarshalIndent(m, "", "  ")
}

// ImportBackup restores configuration from a JSON export. It validates the
// schema version and rejects unknown top-level fields. Secret fields that are
// null in the backup keep their existing database values.
func (s *Store) ImportBackup(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse backup: %w", err)
	}

	allowed := map[string]bool{
		"schema_version": true, "redacted_fields": true, "settings": true,
		"connections": true, "api_keys": true, "combos": true,
		"aliases": true, "pricing_overrides": true, "proxy_pools": true,
		"disabled_models": true, "custom_models": true, "tunnel_configs": true,
		"chat_sessions": true, "teams": true, "virtual_keys": true,
		"routing_rules": true, "model_limits": true, "prompt_templates": true,
		"mcp_tool_groups": true, "alert_channels": true, "feature_flags": true,
	}
	for k := range raw {
		if !allowed[k] {
			return fmt.Errorf("unknown field %q in backup", k)
		}
	}

	var version string
	if v, ok := raw["schema_version"]; ok {
		_ = json.Unmarshal(v, &version)
	}
	if version != backupSchemaVersion {
		return fmt.Errorf("unsupported schema version %q", version)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin restore: %w", err)
	}
	defer tx.Rollback()

	// Settings
	if v, ok := raw["settings"]; ok {
		var settings Settings
		if err := json.Unmarshal(v, &settings); err != nil {
			return fmt.Errorf("restore settings: %w", err)
		}
		if err := restoreSettingsTx(tx, settings); err != nil {
			return err
		}
	}

	// Connections
	if v, ok := raw["connections"]; ok {
		var rows []Connection
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore connections: %w", err)
		}
		for _, r := range rows {
			if err := restoreConnectionTx(s, tx, r); err != nil {
				return err
			}
		}
	}

	// API Keys
	if v, ok := raw["api_keys"]; ok {
		var rows []backupAPIKey
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore api_keys: %w", err)
		}
		for _, r := range rows {
			if err := restoreAPIKeyTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Combos
	if v, ok := raw["combos"]; ok {
		var rows []Combo
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore combos: %w", err)
		}
		for _, r := range rows {
			if err := restoreComboTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Aliases
	if v, ok := raw["aliases"]; ok {
		var rows []ModelAlias
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore aliases: %w", err)
		}
		for _, r := range rows {
			if err := restoreAliasTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Pricing overrides
	if v, ok := raw["pricing_overrides"]; ok {
		var rows []PricingOverride
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore pricing_overrides: %w", err)
		}
		for _, r := range rows {
			if err := restorePricingTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Proxy pools
	if v, ok := raw["proxy_pools"]; ok {
		var rows []backupProxyPool
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore proxy_pools: %w", err)
		}
		for _, r := range rows {
			if err := restoreProxyPoolTx(s, tx, r.ProxyPool); err != nil {
				return err
			}
		}
	}

	// Disabled models
	if v, ok := raw["disabled_models"]; ok {
		var rows []DisabledModel
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore disabled_models: %w", err)
		}
		for _, r := range rows {
			if err := restoreDisabledModelTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Custom models
	if v, ok := raw["custom_models"]; ok {
		var rows []CustomModel
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore custom_models: %w", err)
		}
		for _, r := range rows {
			if err := restoreCustomModelTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Tunnel configs
	if v, ok := raw["tunnel_configs"]; ok {
		var rows []backupTunnelConfig
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore tunnel_configs: %w", err)
		}
		for _, r := range rows {
			if err := restoreTunnelConfigTx(s, tx, r.TunnelConfig); err != nil {
				return err
			}
		}
	}

	// Chat sessions
	if v, ok := raw["chat_sessions"]; ok {
		var rows []ChatSession
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore chat_sessions: %w", err)
		}
		for _, r := range rows {
			if err := restoreChatSessionTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Teams
	if v, ok := raw["teams"]; ok {
		var rows []Team
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore teams: %w", err)
		}
		for _, r := range rows {
			if err := restoreTeamTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Virtual keys
	if v, ok := raw["virtual_keys"]; ok {
		var rows []VirtualKey
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore virtual_keys: %w", err)
		}
		for _, r := range rows {
			if err := restoreVirtualKeyTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Routing rules
	if v, ok := raw["routing_rules"]; ok {
		var rows []RoutingRule
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore routing_rules: %w", err)
		}
		for _, r := range rows {
			if err := restoreRoutingRuleTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Model limits
	if v, ok := raw["model_limits"]; ok {
		var rows []ModelLimit
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore model_limits: %w", err)
		}
		for _, r := range rows {
			if err := restoreModelLimitTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Prompt templates
	if v, ok := raw["prompt_templates"]; ok {
		var rows []PromptTemplate
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore prompt_templates: %w", err)
		}
		for _, r := range rows {
			if err := restorePromptTemplateTx(tx, r); err != nil {
				return err
			}
		}
	}

	// MCP tool groups
	if v, ok := raw["mcp_tool_groups"]; ok {
		var rows []MCPToolGroup
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore mcp_tool_groups: %w", err)
		}
		for _, r := range rows {
			if err := restoreMCPToolGroupTx(tx, r); err != nil {
				return err
			}
		}
	}

	// Alert channels
	if v, ok := raw["alert_channels"]; ok {
		var rows []backupAlertChannel
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore alert_channels: %w", err)
		}
		for _, r := range rows {
			if err := restoreAlertChannelTx(s, tx, r.AlertChannel); err != nil {
				return err
			}
		}
	}

	// Feature flags
	if v, ok := raw["feature_flags"]; ok {
		var rows []FeatureFlag
		if err := json.Unmarshal(v, &rows); err != nil {
			return fmt.Errorf("restore feature_flags: %w", err)
		}
		for _, r := range rows {
			if err := restoreFeatureFlagTx(tx, r); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func restoreSettingsTx(tx *sql.Tx, settings Settings) error {
	values := map[string]string{
		"require_api_key":     boolString(settings.RequireAPIKey),
		"require_login":       boolString(settings.RequireLogin),
		"trust_proxy_headers": boolString(settings.TrustProxyHeaders),
		"rtk_enabled":         boolString(settings.RTKEnabled),
		"caveman_enabled":     boolString(settings.CavemanEnabled),
		"caveman_level":       settings.CavemanLevel,
		"enable_request_logs": boolString(settings.EnableRequestLogs),
		"proxy_url":           settings.ProxyURL,
		"data_dir":            settings.DataDir,
		"log_retention_days":  fmt.Sprintf("%d", settings.LogRetentionDays),
		"allowed_sources":     strings.Join(settings.AllowedSources, ","),
		"notify_webhook_url":  settings.NotifyWebhookURL,
		"notify_on_reauth":    boolString(settings.NotifyOnReauth),
		"cache_enabled":       boolString(settings.CacheEnabled),
		"cache_ttl_seconds":   fmt.Sprintf("%d", settings.CacheTTLSeconds),
	}
	for key, value := range values {
		_, err := tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return fmt.Errorf("restore setting %q: %w", key, err)
		}
	}
	return nil
}

func restoreConnectionTx(s *Store, tx *sql.Tx, r Connection) error {
	var existingID string
	existsErr := tx.QueryRow("SELECT id FROM connections WHERE id = ?", r.ID).Scan(&existingID)

	var existing Connection
	if existsErr == nil {
		row := tx.QueryRow("SELECT access_token, refresh_token, api_key, provider_specific_data FROM connections WHERE id = ?", r.ID)
		var at, rt, ak sql.NullString
		var psd sql.NullString
		if err := row.Scan(&at, &rt, &ak, &psd); err == nil {
			existing.AccessToken = stringPtrFromNull(at)
			existing.RefreshToken = stringPtrFromNull(rt)
			existing.APIKey = stringPtrFromNull(ak)
			if psd.Valid {
				_ = json.Unmarshal([]byte(psd.String), &existing.ProviderSpecificData)
			}
		}
	}

	providerData, _ := encodeJSON(r.ProviderSpecificData)
	if providerData == nil && existing.ProviderSpecificData != nil {
		pd, _ := encodeJSON(existing.ProviderSpecificData)
		providerData = pd
	}
	modelLocks, _ := encodeJSON(r.ModelLocks)

	if existsErr == nil {
		_, err := tx.Exec(`UPDATE connections SET
			provider = ?, name = ?, auth_type = ?, access_token = ?, refresh_token = ?,
			expires_at = ?, api_key = ?, is_active = ?, provider_specific_data = ?,
			account_id = ?, email = ?, unavailable_until = ?, backoff_level = ?,
			model_locks = ?, needs_reauth = ?, last_refresh_error = ?,
			quota_limit = ?, quota_remaining = ?
			WHERE id = ?`,
			r.Provider, r.Name, r.AuthType,
			coalesceStringPtr(r.AccessToken, existing.AccessToken),
			coalesceStringPtr(r.RefreshToken, existing.RefreshToken),
			r.ExpiresAt, coalesceStringPtr(r.APIKey, existing.APIKey), boolToInt(r.IsActive),
			providerData, r.AccountID, r.Email, r.UnavailableUntil, r.BackoffLevel, modelLocks,
			r.NeedsReauth, r.LastRefreshError, r.QuotaLimit, r.QuotaRemaining,
			r.ID,
		)
		if err != nil {
			return fmt.Errorf("update connection %q: %w", r.ID, err)
		}
		return nil
	}

	_, err := tx.Exec(`INSERT INTO connections (
		id, provider, name, auth_type, access_token, refresh_token, expires_at, api_key, is_active,
		provider_specific_data, account_id, email, unavailable_until, backoff_level, model_locks,
		needs_reauth, last_refresh_error, quota_limit, quota_remaining
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Provider, r.Name, r.AuthType, r.AccessToken, r.RefreshToken,
		r.ExpiresAt, r.APIKey, boolToInt(r.IsActive), providerData, r.AccountID,
		r.Email, r.UnavailableUntil, r.BackoffLevel, modelLocks,
		r.NeedsReauth, r.LastRefreshError, r.QuotaLimit, r.QuotaRemaining,
	)
	if err != nil {
		return fmt.Errorf("insert connection %q: %w", r.ID, err)
	}
	return nil
}

func coalesceStringPtr(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}

type backupAPIKey struct {
	APIKey
	KeyHash string `json:"key_hash"`
}

func (s *Store) listAPIKeysForBackup() ([]backupAPIKey, error) {
	rows, err := s.db.Query(`SELECT id, name, key_hash, prefix, is_active, last_used_at, created_at,
		expires_at, scopes, rate_limit_rpm, rate_limit_tpm, daily_spend_cap_usd FROM api_keys`)
	if err != nil {
		return nil, fmt.Errorf("query api keys for backup: %w", err)
	}
	defer rows.Close()

	var keys []backupAPIKey
	for rows.Next() {
		var k backupAPIKey
		var expiresAt sql.NullInt64
		var scopes sql.NullString
		var rpm, tpm sql.NullInt64
		var cap sql.NullFloat64
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyHash, &k.Prefix, &k.IsActive, &k.LastUsedAt, &k.CreatedAt,
			&expiresAt, &scopes, &rpm, &tpm, &cap); err != nil {
			return nil, fmt.Errorf("scan api key for backup: %w", err)
		}
		k.ExpiresAt = int64PtrFromNull(expiresAt)
		_ = decodeJSON(scopes, &k.Scopes)
		if rpm.Valid {
			v := int(rpm.Int64)
			k.RateLimitRPM = &v
		}
		if tpm.Valid {
			v := int(tpm.Int64)
			k.RateLimitTPM = &v
		}
		if cap.Valid {
			v := cap.Float64
			k.DailySpendCapUSD = &v
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys for backup: %w", err)
	}
	return keys, nil
}

func restoreAPIKeyTx(tx *sql.Tx, r backupAPIKey) error {
	var scopes *string
	if len(r.Scopes) > 0 {
		encoded, _ := json.Marshal(r.Scopes)
		v := string(encoded)
		scopes = &v
	}
	_, err := tx.Exec(`INSERT OR REPLACE INTO api_keys (
		id, name, key_hash, prefix, is_active, last_used_at, expires_at, scopes, rate_limit_rpm, rate_limit_tpm, daily_spend_cap_usd
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.KeyHash, r.Prefix, boolToInt(r.IsActive), r.LastUsedAt,
		r.ExpiresAt, scopes, r.RateLimitRPM, r.RateLimitTPM, r.DailySpendCapUSD,
	)
	if err != nil {
		return fmt.Errorf("restore api key %q: %w", r.ID, err)
	}
	return nil
}

func joinScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}
	return strings.Join(scopes, ",")
}

func restoreComboTx(tx *sql.Tx, r Combo) error {
	stepsJSON, _ := json.Marshal(r.Steps)
	_, err := tx.Exec(`INSERT OR REPLACE INTO combos (id, name, steps, is_active, strategy, mcp_tool_group) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, string(stepsJSON), boolToInt(r.IsActive), r.Strategy, r.MCPToolGroup)
	if err != nil {
		return fmt.Errorf("restore combo %q: %w", r.ID, err)
	}
	return nil
}

func restoreAliasTx(tx *sql.Tx, r ModelAlias) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO model_aliases (alias, provider, model) VALUES (?, ?, ?)`,
		r.Alias, r.Provider, r.Model)
	if err != nil {
		return fmt.Errorf("restore alias %q: %w", r.Alias, err)
	}
	return nil
}

func restorePricingTx(tx *sql.Tx, r PricingOverride) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO pricing_overrides (provider, model, input_cost_per_token, output_cost_per_token) VALUES (?, ?, ?, ?)`,
		r.Provider, r.Model, r.InputCostPerToken, r.OutputCostPerToken)
	if err != nil {
		return fmt.Errorf("restore pricing %q/%q: %w", r.Provider, r.Model, err)
	}
	return nil
}

func restoreProxyPoolTx(s *Store, tx *sql.Tx, r ProxyPool) error {
	var existing ProxyPool
	row := tx.QueryRow("SELECT password_enc FROM proxy_pools WHERE id = ?", r.ID)
	var pwdEnc sql.NullString
	scanErr := row.Scan(&pwdEnc)
	if scanErr == nil {
		existing.Password = stringValueFromNull(pwdEnc)
	}

	passwordEnc := emptyStringNil(r.Password)
	if r.Password == "" && existing.Password != "" {
		passwordEnc = pwdEnc.String
	} else if r.Password != "" {
		enc, err := s.encryptString(r.Password)
		if err != nil {
			return fmt.Errorf("encrypt proxy pool password: %w", err)
		}
		passwordEnc = enc
	}

	_, err := tx.Exec(`INSERT OR REPLACE INTO proxy_pools (id, name, protocol, host, port, username, password_enc, is_active, last_check_at, last_check_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Protocol, r.Host, r.Port, emptyStringNil(r.Username), passwordEnc, boolToInt(r.IsActive), emptyStringNil(r.LastCheckAt), emptyStringNil(r.LastCheckStatus))
	if err != nil {
		return fmt.Errorf("restore proxy pool %q: %w", r.ID, err)
	}
	return nil
}

func restoreDisabledModelTx(tx *sql.Tx, r DisabledModel) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO disabled_models (id, provider, model, created_at) VALUES (?, ?, ?, ?)`,
		r.ID, r.Provider, r.Model, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore disabled model: %w", err)
	}
	return nil
}

func restoreCustomModelTx(tx *sql.Tx, r CustomModel) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO custom_models (id, provider, model, display_name, created_at) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Provider, r.Model, r.DisplayName, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore custom model: %w", err)
	}
	return nil
}

func restoreTunnelConfigTx(s *Store, tx *sql.Tx, r TunnelConfig) error {
	var existing TunnelConfig
	row := tx.QueryRow("SELECT config_enc FROM tunnel_config WHERE id = ?", r.ID)
	var cfgEnc sql.NullString
	scanErr := row.Scan(&cfgEnc)
	if scanErr == nil {
		existing.Config = stringValueFromNull(cfgEnc)
	}

	configEnc := emptyStringNil(r.Config)
	if r.Config == "" && existing.Config != "" {
		configEnc = cfgEnc.String
	} else if r.Config != "" {
		enc, err := s.encryptString(r.Config)
		if err != nil {
			return fmt.Errorf("encrypt tunnel config: %w", err)
		}
		configEnc = enc
	}

	_, err := tx.Exec(`INSERT OR REPLACE INTO tunnel_config (id, type, is_enabled, config_enc, url, status, last_error, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Type, boolToInt(r.IsEnabled), configEnc, r.URL, r.Status, r.LastError, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore tunnel config %q: %w", r.Type, err)
	}
	return nil
}

func restoreChatSessionTx(tx *sql.Tx, r ChatSession) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO chat_sessions (id, title, model, provider, messages_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Title, r.Model, r.Provider, r.MessagesJSON, r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("restore chat session: %w", err)
	}
	return nil
}

func restoreTeamTx(tx *sql.Tx, r Team) error {
	var resetAt any
	if r.BudgetResetAt != nil {
		resetAt = r.BudgetResetAt.Format("2006-01-02T15:04:05")
	}
	createdAt := r.CreatedAt.Format("2006-01-02T15:04:05")
	_, err := tx.Exec(`INSERT OR REPLACE INTO teams (id, name, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.BudgetUSD, r.BudgetPeriod, r.BudgetUsedUSD, resetAt, r.RateLimitRPM, createdAt)
	if err != nil {
		return fmt.Errorf("restore team %q: %w", r.Name, err)
	}
	return nil
}

func restoreVirtualKeyTx(tx *sql.Tx, r VirtualKey) error {
	var resetAt any
	if r.BudgetResetAt != nil {
		resetAt = r.BudgetResetAt.Format("2006-01-02T15:04:05")
	}
	createdAt := r.CreatedAt.Format("2006-01-02T15:04:05")
	_, err := tx.Exec(`INSERT OR REPLACE INTO virtual_keys (id, name, key_prefix, key_hash, budget_usd, budget_period, budget_used_usd, budget_reset_at, rate_limit_rpm, rate_limit_tpm, team_id, is_active, mcp_tool_group, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.KeyPrefix, r.KeyHash, r.BudgetUSD, r.BudgetPeriod, r.BudgetUsedUSD, resetAt, r.RateLimitRPM, r.RateLimitTPM, r.TeamID, boolToInt(r.IsActive), r.MCPToolGroup, createdAt)
	if err != nil {
		return fmt.Errorf("restore virtual key %q: %w", r.Name, err)
	}
	return nil
}

func restoreRoutingRuleTx(tx *sql.Tx, r RoutingRule) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO routing_rules (id, name, priority, cond_field, cond_operator, cond_value, target_provider, target_model, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.Priority, r.CondField, r.CondOperator, r.CondValue, r.TargetProvider, r.TargetModel, boolToInt(r.IsActive), r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore routing rule %q: %w", r.Name, err)
	}
	return nil
}

func restoreModelLimitTx(tx *sql.Tx, r ModelLimit) error {
	allowedJSON, _ := json.Marshal(r.AllowedKeyIDs)
	_, err := tx.Exec(`INSERT OR REPLACE INTO model_limits (id, model, max_tokens, max_rpm, allowed_key_ids, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Model, r.MaxTokens, r.MaxRPM, string(allowedJSON), r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore model limit %q: %w", r.Model, err)
	}
	return nil
}

func restorePromptTemplateTx(tx *sql.Tx, r PromptTemplate) error {
	modelsJSON, _ := json.Marshal(r.Models)
	_, err := tx.Exec(`INSERT OR REPLACE INTO prompt_templates (id, name, system_prompt, models_json, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.SystemPrompt, string(modelsJSON), boolToInt(r.IsActive), r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("restore prompt template %q: %w", r.Name, err)
	}
	return nil
}

func restoreMCPToolGroupTx(tx *sql.Tx, r MCPToolGroup) error {
	toolsJSON, _ := json.Marshal(r.ToolIDs)
	_, err := tx.Exec(`INSERT OR REPLACE INTO mcp_tool_groups (id, name, tool_ids_json, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, string(toolsJSON), boolToInt(r.IsActive), r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return fmt.Errorf("restore mcp tool group %q: %w", r.Name, err)
	}
	return nil
}

func restoreAlertChannelTx(s *Store, tx *sql.Tx, r AlertChannel) error {
	var existing AlertChannel
	row := tx.QueryRow("SELECT config_enc FROM alert_channels WHERE id = ?", r.ID)
	var cfgEnc sql.NullString
	scanErr := row.Scan(&cfgEnc)
	if scanErr == nil {
		existing.Config = stringValueFromNull(cfgEnc)
	}

	configEnc := emptyStringNil(r.Config)
	if r.Config == "" && existing.Config != "" {
		configEnc = cfgEnc.String
	} else if r.Config != "" {
		enc, err := s.encryptString(r.Config)
		if err != nil {
			return fmt.Errorf("encrypt alert config: %w", err)
		}
		configEnc = enc
	}

	eventsJSON, _ := json.Marshal(r.Events)
	_, err := tx.Exec(`INSERT OR REPLACE INTO alert_channels (id, name, channel_type, config_enc, events_json, is_active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.Name, r.ChannelType, configEnc, string(eventsJSON), boolToInt(r.IsActive), r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore alert channel %q: %w", r.Name, err)
	}
	return nil
}

func restoreFeatureFlagTx(tx *sql.Tx, r FeatureFlag) error {
	_, err := tx.Exec(`INSERT OR REPLACE INTO feature_flags (id, key, enabled, description, created_at) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.Key, boolToInt(r.Enabled), r.Description, r.CreatedAt)
	if err != nil {
		return fmt.Errorf("restore feature flag %q: %w", r.Key, err)
	}
	return nil
}

func uniqStrings(ss []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
