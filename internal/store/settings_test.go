package store

import (
	"errors"
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

func TestGetSettingsNotifyDefaults(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.NotifyWebhookURL != "" {
		t.Errorf("NotifyWebhookURL = %q, want empty", settings.NotifyWebhookURL)
	}
	if !settings.NotifyOnReauth {
		t.Error("NotifyOnReauth should default to true")
	}
}

func TestUpdateSettingsNotifyRoundTrip(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.NotifyWebhookURL = "https://discord.com/api/webhooks/1/abc"
	settings.NotifyOnReauth = false
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.NotifyWebhookURL != "https://discord.com/api/webhooks/1/abc" {
		t.Errorf("NotifyWebhookURL = %q, want round-tripped", got.NotifyWebhookURL)
	}
	if got.NotifyOnReauth {
		t.Error("NotifyOnReauth = true, want false after round-trip")
	}
}

func TestUpdateSettingsRejectsNonHTTPWebhookURL(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	for _, bad := range []string{"ftp://example.com/hook", "notaurl", "://nohost"} {
		settings.NotifyWebhookURL = bad
		if err := s.UpdateSettings(settings); err == nil {
			t.Errorf("UpdateSettings should reject notify_webhook_url %q", bad)
		}
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

func TestGetSettingsCacheDefaults(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if settings.CacheEnabled {
		t.Error("CacheEnabled should default to false")
	}
	if settings.CacheTTLSeconds != 300 {
		t.Errorf("CacheTTLSeconds = %d, want 300", settings.CacheTTLSeconds)
	}
}

func TestUpdateSettingsCacheRoundTrip(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.CacheEnabled = true
	settings.CacheTTLSeconds = 600
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !got.CacheEnabled {
		t.Error("CacheEnabled = false, want true after round-trip")
	}
	if got.CacheTTLSeconds != 600 {
		t.Errorf("CacheTTLSeconds = %d, want 600", got.CacheTTLSeconds)
	}

	// Zero disables caching and must round-trip as zero.
	settings.CacheTTLSeconds = 0
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings zero TTL: %v", err)
	}
	got, err = s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.CacheTTLSeconds != 0 {
		t.Errorf("CacheTTLSeconds = %d, want 0", got.CacheTTLSeconds)
	}
}

func TestUpdateSettingsRejectsNegativeCacheTTL(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.CacheTTLSeconds = -1
	if err := s.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings should reject negative CacheTTLSeconds")
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

func TestUpdateSettingsRequireLoginRejectsNoUsers(t *testing.T) {
	s := openTestStore(t)

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.RequireLogin = true
	if err := s.UpdateSettings(settings); !errors.Is(err, ErrRequireLoginNoUsers) {
		t.Fatalf("UpdateSettings = %v, want ErrRequireLoginNoUsers", err)
	}
}

func TestGetAPIKeySecretEmpty(t *testing.T) {
	s := openTestStore(t)

	secret, err := s.GetAPIKeySecret()
	if err != nil {
		t.Fatalf("GetAPIKeySecret: %v", err)
	}
	if secret != "" {
		t.Fatalf("secret = %q, want empty", secret)
	}
}

func TestSetAndGetAPIKeySecret(t *testing.T) {
	s := openTestStore(t)

	if err := s.SetAPIKeySecret("my-secret-value"); err != nil {
		t.Fatalf("SetAPIKeySecret: %v", err)
	}

	got, err := s.GetAPIKeySecret()
	if err != nil {
		t.Fatalf("GetAPIKeySecret: %v", err)
	}
	if got != "my-secret-value" {
		t.Fatalf("secret = %q, want my-secret-value", got)
	}
}

func TestUpdateSettingsRequireLoginAcceptsWithUsers(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
		t.Fatalf("CreateDashboardUser: %v", err)
	}

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.RequireLogin = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !got.RequireLogin {
		t.Error("RequireLogin = false, want true")
	}
}
