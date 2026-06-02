package store

import "testing"

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
	if got != settings {
		t.Fatalf("settings = %+v, want %+v", got, settings)
	}
}
