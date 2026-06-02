package store

import (
	"fmt"
)

type Settings struct {
	RequireAPIKey     bool
	RTKEnabled        bool
	CavemanEnabled    bool
	CavemanLevel      string
	EnableRequestLogs bool
	ProxyURL          string
	DataDir           string
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
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
