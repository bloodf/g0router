package store

import (
	"testing"
)

// GetSettings: covers applySetting branches not yet hit (all key paths).
func TestGetSettingsAppliesAllKeys(t *testing.T) {
	s := openTestStore(t)

	want := Settings{
		RequireAPIKey:     false,
		RTKEnabled:        false,
		CavemanEnabled:    true,
		CavemanLevel:      "lite",
		EnableRequestLogs: true,
		ProxyURL:          "http://proxy:8080",
		DataDir:           "/tmp/data",
		LogRetentionDays:  14,
	}
	if err := s.UpdateSettings(want); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got != want {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

// log_retention_days: non-parseable value leaves default intact.
func TestApplySettingLogRetentionDaysInvalidIgnored(t *testing.T) {
	s := defaultSettings()
	applySetting(&s, "log_retention_days", "not-a-number")
	if s.LogRetentionDays != 30 {
		t.Fatalf("LogRetentionDays = %d, want 30 (default preserved on parse error)", s.LogRetentionDays)
	}
}

// applySetting: unknown key is a no-op (default switch fallthrough).
func TestApplySettingUnknownKeyIgnored(t *testing.T) {
	s := defaultSettings()
	before := s
	applySetting(&s, "unknown_key", "some_value")
	if s != before {
		t.Fatalf("unknown key mutated settings: %+v", s)
	}
}

// UpdateSettings: negative log_retention_days rejected.
func TestUpdateSettingsNegativeLogRetentionRejected(t *testing.T) {
	s := openTestStore(t)
	settings := defaultSettings()
	settings.LogRetentionDays = -10
	if err := s.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings should return error for negative LogRetentionDays")
	}
}

// UpdateSettings: LogRetentionDays=0 persists and round-trips.
func TestUpdateSettingsLogRetentionZeroPersists(t *testing.T) {
	s := openTestStore(t)
	settings := defaultSettings()
	settings.LogRetentionDays = 0
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings zero: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.LogRetentionDays != 0 {
		t.Fatalf("LogRetentionDays = %d, want 0", got.LogRetentionDays)
	}
}

// GetSettings scan error: replace settings table with a view that returns
// only one column so "SELECT key, value" fails at scan time.
func TestGetSettingsScanError(t *testing.T) {
	s := openTestStore(t)
	// Replace the real settings table with a view that returns a single NULL
	// column named "key" — scanning into (key string, value string) will fail
	// because the result set has only one column.
	if _, err := s.db.Exec("DROP TABLE IF EXISTS settings"); err != nil {
		t.Fatalf("drop settings: %v", err)
	}
	if _, err := s.db.Exec("CREATE VIEW settings AS SELECT NULL AS key"); err != nil {
		t.Fatalf("create view: %v", err)
	}
	if _, err := s.GetSettings(); err == nil {
		t.Fatal("GetSettings with one-column view: want scan/query error")
	}
}

// UpdateSettings tx.Exec error: corrupt settings table so INSERT OR REPLACE fails.
func TestUpdateSettingsTxExecError(t *testing.T) {
	s := openTestStore(t)
	// Drop and recreate settings with a NOT NULL constraint on value that
	// we can violate, causing tx.Exec to fail but tx.Begin to succeed.
	// Simpler: make the table read-only by dropping it and creating a VIEW.
	if _, err := s.db.Exec("DROP TABLE IF EXISTS settings"); err != nil {
		t.Fatalf("drop settings: %v", err)
	}
	if _, err := s.db.Exec("CREATE VIEW settings AS SELECT 'x' AS key, 'y' AS value"); err != nil {
		t.Fatalf("create view: %v", err)
	}
	settings := defaultSettings()
	if err := s.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings with view instead of table: want error")
	}
}

// GetSettings: all boolean keys flip correctly through the store.
func TestGetSettingsBooleanFlips(t *testing.T) {
	s := openTestStore(t)

	// Default has RequireAPIKey=true, RTKEnabled=true, CavemanEnabled=false, EnableRequestLogs=false.
	// Flip them all.
	want := Settings{
		RequireAPIKey:     false,
		RTKEnabled:        false,
		CavemanEnabled:    true,
		CavemanLevel:      "full",
		EnableRequestLogs: true,
		LogRetentionDays:  30,
	}
	if err := s.UpdateSettings(want); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.RequireAPIKey != false || got.RTKEnabled != false || got.CavemanEnabled != true || got.EnableRequestLogs != true {
		t.Fatalf("boolean settings = %+v", got)
	}
}
