package store

import (
	"testing"
)

// TestValidateNotifyWebhookURLNoHost exercises the parsed.Host=="" branch (line 171).
func TestValidateNotifyWebhookURLNoHost(t *testing.T) {
	// "http://" has scheme but no host.
	if err := validateNotifyWebhookURL("http://"); err == nil {
		t.Fatal("http:// should fail validation (no host)")
	}
}

// TestValidateNotifyWebhookURLNonHTTP exercises the scheme check (line 168-169).
func TestValidateNotifyWebhookURLNonHTTP(t *testing.T) {
	if err := validateNotifyWebhookURL("ftp://example.com/hook"); err == nil {
		t.Fatal("ftp:// should fail (not http/https)")
	}
}

// TestParseAllowedSourcesAllWhitespace exercises the len(sources)==0 branch (line 192).
func TestParseAllowedSourcesAllWhitespace(t *testing.T) {
	// A string that is non-empty but all commas/whitespace produces len(sources)==0.
	got := parseAllowedSources("  ,  ,  ")
	defaults := defaultAllowedSources()
	if len(got) != len(defaults) {
		t.Fatalf("all-whitespace sources = %v, want defaults %v", got, defaults)
	}
}

// TestGetSettingsEmptyDBReturnsDefaults exercises the GetSettings path when
// the settings table has no rows (fresh DB with no rows → returns defaults).
func TestGetSettingsDefaultOnEmptyDB(t *testing.T) {
	s := openTestStore(t)
	// GetSettings on a fresh store should return defaults without error.
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !settings.RequireAPIKey {
		t.Error("defaults: RequireAPIKey should be true")
	}
}

// TestUpdateSettingsWebhookURLValidation exercises the UpdateSettings validation
// path for an invalid webhook URL.
func TestUpdateSettingsWebhookURLInvalid(t *testing.T) {
	s := openTestStore(t)
	settings := Settings{NotifyWebhookURL: "not-a-url"}
	if err := s.UpdateSettings(settings); err == nil {
		t.Fatal("UpdateSettings with invalid webhook URL should return error")
	}
}
