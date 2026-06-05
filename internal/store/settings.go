package store

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type Settings struct {
	RequireAPIKey     bool     `json:"require_api_key"`
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

	if err := validateNotifyWebhookURL(settings.NotifyWebhookURL); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin settings update: %w", err)
	}
	defer tx.Rollback()

	values := map[string]string{
		"require_api_key":     boolString(settings.RequireAPIKey),
		"rtk_enabled":         boolString(settings.RTKEnabled),
		"caveman_enabled":     boolString(settings.CavemanEnabled),
		"caveman_level":       settings.CavemanLevel,
		"enable_request_logs": boolString(settings.EnableRequestLogs),
		"proxy_url":           settings.ProxyURL,
		"data_dir":            settings.DataDir,
		"log_retention_days":  strconv.Itoa(settings.LogRetentionDays),
		"allowed_sources":     strings.Join(settings.AllowedSources, ","),
		"notify_webhook_url":  settings.NotifyWebhookURL,
		"notify_on_reauth":    boolString(settings.NotifyOnReauth),
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
	}
}

func applySetting(settings *Settings, key, value string) {
	switch key {
	case "require_api_key":
		settings.RequireAPIKey = value == "true"
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

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
