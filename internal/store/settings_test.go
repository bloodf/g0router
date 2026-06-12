package store

import (
	"errors"
	"testing"
)

func TestGetSettingMissing(t *testing.T) {
	st := newTestStore(t)

	_, err := st.GetSetting("missing-key")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err=%v, want ErrNotFound", err)
	}
}

func TestSetSettingAndGetSetting(t *testing.T) {
	st := newTestStore(t)

	if err := st.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	got, err := st.GetSetting("theme")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != "dark" {
		t.Errorf("GetSetting(theme)=%q, want dark", got)
	}

	// Upsert.
	if err := st.SetSetting("theme", "light"); err != nil {
		t.Fatalf("SetSetting upsert: %v", err)
	}
	got, err = st.GetSetting("theme")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != "light" {
		t.Errorf("GetSetting(theme)=%q, want light", got)
	}
}

func TestSetSettingUpdatesGetSettings(t *testing.T) {
	st := newTestStore(t)

	if err := st.SetSetting("log_level", "debug"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	all, err := st.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if all["log_level"] != "debug" {
		t.Errorf("settings[log_level]=%q, want debug", all["log_level"])
	}
}

func TestSetSettingLearnedTokenParam(t *testing.T) {
	st := newTestStore(t)

	key := "learned_token_param:openai:gpt-4"
	if err := st.SetSetting(key, "max_completion_tokens"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	got, err := st.GetSetting(key)
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != "max_completion_tokens" {
		t.Errorf("GetSetting(%q)=%q, want max_completion_tokens", key, got)
	}
}
