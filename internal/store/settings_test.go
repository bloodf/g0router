package store

import (
	"reflect"
	"testing"
)

func TestGetSettingsDefaults(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	if !settings.RequireAPIKey {
		t.Error("RequireAPIKey should default to true")
	}
	if !settings.RTKEnabled {
		t.Error("RTKEnabled should default to true")
	}
	if settings.CavemanEnabled {
		t.Error("CavemanEnabled should default to false")
	}
	if settings.CavemanLevel != "full" {
		t.Errorf("CavemanLevel = %q, want full", settings.CavemanLevel)
	}
	if settings.EnableRequestLogs {
		t.Error("EnableRequestLogs should default to false")
	}
	if settings.ProxyURL != "" {
		t.Errorf("ProxyURL = %q, want empty", settings.ProxyURL)
	}
	if settings.DataDir != "" {
		t.Errorf("DataDir = %q, want empty", settings.DataDir)
	}
	if settings.LogRetentionDays != 30 {
		t.Errorf("LogRetentionDays = %d, want 30", settings.LogRetentionDays)
	}
}

func TestUpdateSettingsLogRetentionRoundTrip(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.LogRetentionDays = 7
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.LogRetentionDays != 7 {
		t.Fatalf("LogRetentionDays = %d, want 7", got.LogRetentionDays)
	}

	settings.LogRetentionDays = 0
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings zero: %v", err)
	}
	got, err = s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.LogRetentionDays != 0 {
		t.Fatalf("LogRetentionDays = %d, want 0 (keep forever)", got.LogRetentionDays)
	}
}

func TestUpdateSettingsRejectsNegativeRetention(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.LogRetentionDays = -1
	if err := s.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings should reject negative LogRetentionDays")
	}
}

func TestUpdateAndGetSettings(t *testing.T) {
	s := openTestStore(t)

	want := Settings{
		RequireAPIKey:     false,
		RTKEnabled:        false,
		CavemanEnabled:    true,
		CavemanLevel:      "lite",
		EnableRequestLogs: true,
		ProxyURL:          "http://proxy.local:8080",
		DataDir:           "/tmp/g0router-data",
		AllowedSources:    []string{"local", "lan"},
	}

	if err := s.UpdateSettings(want); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

func TestUpdateSettingsIdempotent(t *testing.T) {
	s := openTestStore(t)

	settings := Settings{
		RequireAPIKey:     true,
		RTKEnabled:        true,
		CavemanEnabled:    false,
		CavemanLevel:      "ultra",
		EnableRequestLogs: false,
		ProxyURL:          "",
		DataDir:           "/tmp/g0router",
		AllowedSources:    []string{"public"},
	}

	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("first UpdateSettings: %v", err)
	}
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("second UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !reflect.DeepEqual(got, settings) {
		t.Fatalf("settings = %+v, want %+v", got, settings)
	}
}

func TestGetSettingsAllowedSourcesDefault(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	want := []string{"local", "lan", "tailscale", "public"}
	if !reflect.DeepEqual(settings.AllowedSources, want) {
		t.Fatalf("AllowedSources = %v, want %v", settings.AllowedSources, want)
	}
}

func TestUpdateSettingsRejectsUnknownSource(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.AllowedSources = []string{"local", "bogus"}
	if err := s.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings should reject unknown allowed_sources token")
	}
}

func TestUpdateSettingsEmptyAllowedSourcesDefaultsToAll(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.AllowedSources = nil
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	want := []string{"local", "lan", "tailscale", "public"}
	if !reflect.DeepEqual(got.AllowedSources, want) {
		t.Fatalf("AllowedSources = %v, want default %v", got.AllowedSources, want)
	}
}
