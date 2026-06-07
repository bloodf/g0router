package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ErrRequireLoginNoUsers is returned when an operator tries to enable
// require_login without creating at least one dashboard user first.
var ErrRequireLoginNoUsers = errors.New("require_login cannot be enabled without at least one dashboard user")

type Settings struct {
	RequireAPIKey     bool     `json:"require_api_key"`
	RequireLogin      bool     `json:"require_login"`
	// TrustProxyHeaders controls whether X-Forwarded-For is trusted for client
	// IP. Enable this when g0router sits behind Cloudflare, Tailscale tunnels,
	// or another reverse proxy where the direct remote address is constant.
	TrustProxyHeaders bool     `json:"trust_proxy_headers"`
	RTKEnabled        bool     `json:"rtk_enabled"`
	CavemanEnabled    bool     `json:"caveman_enabled"`
	CavemanLevel      string   `json:"caveman_level"`
	EnableRequestLogs bool     `json:"enable_request_logs"`
	ProxyURL          string   `json:"proxy_url"`
	DataDir           string   `json:"data_dir"`
	LogRetentionDays  int      `json:"log_retention_days"`
	AllowedSources    []string `json:"allowed_sources"`
	NotifyWebhookURL  string   `json:"notify_webhook_url"`
	NotifyOnReauth    bool     `json:"notify_on_reauth"`
	CacheEnabled      bool     `json:"cache_enabled"`
	CacheTTLSeconds   int      `json:"cache_ttl_seconds"`
	Locale            string   `json:"locale"`
}

// validSourceClasses enumerates the connection-source classes an operator may
// allow. "public" is a superset that permits every class.
var validSourceClasses = map[string]bool{
	"local":     true,
	"lan":       true,
	"tailscale": true,
	"public":    true,
}

// defaultAllowedSources returns the open-by-default policy: all four classes.
func defaultAllowedSources() []string {
	return []string{"local", "lan", "tailscale", "public"}
}

func (s *Store) GetSettings() (Settings, error) {
	settings := defaultSettings()

	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return Settings{}, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return Settings{}, fmt.Errorf("scan setting: %w", err)
		}
		applySetting(&settings, key, value)
	}
	if err := rows.Err(); err != nil {
		return Settings{}, fmt.Errorf("iterate settings: %w", err)
	}

	return settings, nil
}

func (s *Store) UpdateSettings(settings Settings) error {
	if settings.LogRetentionDays < 0 {
		return fmt.Errorf("update settings: log_retention_days must be >= 0, got %d", settings.LogRetentionDays)
	}

	for _, source := range settings.AllowedSources {
		if !validSourceClasses[source] {
			return fmt.Errorf("update settings: invalid allowed_sources token %q", source)
		}
	}

	if settings.CacheTTLSeconds < 0 {
		return fmt.Errorf("update settings: cache_ttl_seconds must be >= 0, got %d", settings.CacheTTLSeconds)
	}

	if err := validateNotifyWebhookURL(settings.NotifyWebhookURL); err != nil {
		return err
	}

	if settings.RequireLogin {
		users, err := s.ListDashboardUsers()
		if err != nil {
			return fmt.Errorf("update settings: %w", err)
		}
		if len(users) == 0 {
			return ErrRequireLoginNoUsers
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin settings update: %w", err)
	}
	defer tx.Rollback()

	values := map[string]string{
		"require_api_key":      boolString(settings.RequireAPIKey),
		"require_login":        boolString(settings.RequireLogin),
		"trust_proxy_headers":  boolString(settings.TrustProxyHeaders),
		"rtk_enabled":          boolString(settings.RTKEnabled),
		"caveman_enabled":      boolString(settings.CavemanEnabled),
		"caveman_level":        settings.CavemanLevel,
		"enable_request_logs":  boolString(settings.EnableRequestLogs),
		"proxy_url":            settings.ProxyURL,
		"data_dir":             settings.DataDir,
		"log_retention_days":   strconv.Itoa(settings.LogRetentionDays),
		"allowed_sources":      strings.Join(settings.AllowedSources, ","),
		"notify_webhook_url":   settings.NotifyWebhookURL,
		"notify_on_reauth":     boolString(settings.NotifyOnReauth),
		"cache_enabled":        boolString(settings.CacheEnabled),
		"cache_ttl_seconds":    strconv.Itoa(settings.CacheTTLSeconds),
		"locale":               settings.Locale,
	}

	for key, value := range values {
		_, err := tx.Exec(
			"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
			key,
			value,
		)
		if err != nil {
			return fmt.Errorf("upsert setting %q: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit settings update: %w", err)
	}
	return nil
}

func defaultSettings() Settings {
	return Settings{
		RequireAPIKey:     true,
		RequireLogin:      false,
		TrustProxyHeaders: false,
		RTKEnabled:        true,
		CavemanEnabled:    false,
		CavemanLevel:      "full",
		EnableRequestLogs: false,
		ProxyURL:          "",
		DataDir:           "",
		LogRetentionDays:  30,
		AllowedSources:    defaultAllowedSources(),
		NotifyWebhookURL:  "",
		NotifyOnReauth:    true,
		CacheEnabled:      false,
		CacheTTLSeconds:   300,
		Locale:            "en",
	}
}

func applySetting(settings *Settings, key, value string) {
	switch key {
	case "require_api_key":
		settings.RequireAPIKey = value == "true"
	case "require_login":
		settings.RequireLogin = value == "true"
	case "trust_proxy_headers":
		settings.TrustProxyHeaders = value == "true"
	case "rtk_enabled":
		settings.RTKEnabled = value == "true"
	case "caveman_enabled":
		settings.CavemanEnabled = value == "true"
	case "caveman_level":
		settings.CavemanLevel = value
	case "enable_request_logs":
		settings.EnableRequestLogs = value == "true"
	case "proxy_url":
		settings.ProxyURL = value
	case "data_dir":
		settings.DataDir = value
	case "log_retention_days":
		if parsed, err := strconv.Atoi(value); err == nil {
			settings.LogRetentionDays = parsed
		}
	case "allowed_sources":
		settings.AllowedSources = parseAllowedSources(value)
	case "notify_webhook_url":
		settings.NotifyWebhookURL = value
	case "notify_on_reauth":
		settings.NotifyOnReauth = value == "true"
	case "cache_enabled":
		settings.CacheEnabled = value == "true"
	case "cache_ttl_seconds":
		if parsed, err := strconv.Atoi(value); err == nil {
			settings.CacheTTLSeconds = parsed
		}
	case "locale":
		settings.Locale = value
	}
}

// validateNotifyWebhookURL ensures a non-empty webhook URL uses http or https.
func validateNotifyWebhookURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("update settings: invalid notify_webhook_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("update settings: notify_webhook_url must be http or https, got %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("update settings: notify_webhook_url must include a host")
	}
	return nil
}

// parseAllowedSources splits the persisted comma-joined list. An empty or
// unset value defaults to all classes so an absent setting never locks anyone
// out.
func parseAllowedSources(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultAllowedSources()
	}
	parts := strings.Split(trimmed, ",")
	sources := make([]string, 0, len(parts))
	for _, part := range parts {
		if token := strings.TrimSpace(part); token != "" {
			sources = append(sources, token)
		}
	}
	if len(sources) == 0 {
		return defaultAllowedSources()
	}
	return sources
}

// GetAPIKeySecret reads the api_key_secret from settings. Returns empty string if not set.
func (s *Store) GetAPIKeySecret() (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", "api_key_secret").Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get api_key_secret: %w", err)
	}
	return value, nil
}

// SetAPIKeySecret persists the api_key_secret in settings.
func (s *Store) SetAPIKeySecret(secret string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", "api_key_secret", secret)
	if err != nil {
		return fmt.Errorf("set api_key_secret: %w", err)
	}
	return nil
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// GuardrailsConfig holds guardrails and PII redaction settings.
type GuardrailsConfig struct {
	GuardrailsEnabled   bool
	GuardrailsBlocklist []string
	PIIRedactionEnabled bool
	PIIRedactionTypes   []string
}

// GetGuardrailsConfig reads guardrails settings from the settings table.
func (s *Store) GetGuardrailsConfig() (GuardrailsConfig, error) {
	cfg := GuardrailsConfig{}

	rows, err := s.db.Query(
		"SELECT key, value FROM settings WHERE key IN (?, ?, ?, ?)",
		"guardrails_enabled", "guardrails_blocklist_json", "pii_redaction_enabled", "pii_types_json",
	)
	if err != nil {
		return GuardrailsConfig{}, fmt.Errorf("query guardrails settings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return GuardrailsConfig{}, fmt.Errorf("scan guardrails setting: %w", err)
		}
		switch key {
		case "guardrails_enabled":
			cfg.GuardrailsEnabled = value == "true"
		case "guardrails_blocklist_json":
			_ = json.Unmarshal([]byte(value), &cfg.GuardrailsBlocklist)
		case "pii_redaction_enabled":
			cfg.PIIRedactionEnabled = value == "true"
		case "pii_types_json":
			_ = json.Unmarshal([]byte(value), &cfg.PIIRedactionTypes)
		}
	}
	if err := rows.Err(); err != nil {
		return GuardrailsConfig{}, fmt.Errorf("iterate guardrails settings: %w", err)
	}

	return cfg, nil
}

// UpdateGuardrailsConfig writes guardrails settings to the settings table.
func (s *Store) UpdateGuardrailsConfig(cfg GuardrailsConfig) error {
	blocklistJSON, _ := json.Marshal(cfg.GuardrailsBlocklist)
	typesJSON, _ := json.Marshal(cfg.PIIRedactionTypes)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin guardrails update: %w", err)
	}
	defer tx.Rollback()

	values := map[string]string{
		"guardrails_enabled":        boolString(cfg.GuardrailsEnabled),
		"guardrails_blocklist_json": string(blocklistJSON),
		"pii_redaction_enabled":     boolString(cfg.PIIRedactionEnabled),
		"pii_types_json":            string(typesJSON),
	}

	for key, value := range values {
		_, err := tx.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
		if err != nil {
			return fmt.Errorf("upsert guardrails setting %q: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit guardrails update: %w", err)
	}
	return nil
}
