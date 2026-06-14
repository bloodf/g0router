package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestAlertChannelCRUD(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateAlertChannel(&AlertChannel{
		Name:        "Webhook Alerts",
		ChannelType: "webhook",
		Config:      map[string]any{"url": "https://hooks.example.com/g0router"},
		Events:      []string{"quota_exceeded", "provider_error"},
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created ID is zero")
	}
	if created.CreatedAt == "" {
		t.Fatal("created_at not set")
	}
	if created.Config["url"] != "https://hooks.example.com/g0router" {
		t.Fatalf("config not persisted: %+v", created.Config)
	}
	if !reflect.DeepEqual(created.Events, []string{"quota_exceeded", "provider_error"}) {
		t.Fatalf("events = %v", created.Events)
	}

	got, err := st.GetAlertChannelByID(created.ID)
	if err != nil {
		t.Fatalf("GetAlertChannelByID: %v", err)
	}
	if got.Config["url"] != "https://hooks.example.com/g0router" {
		t.Fatalf("config round-trip mismatch: %+v", got.Config)
	}
	if !got.IsActive {
		t.Fatalf("IsActive = false, want true")
	}

	created2, err := st.CreateAlertChannel(&AlertChannel{Name: "Discord", ChannelType: "discord",
		Config: map[string]any{"webhook_url": "https://discord.com/api/webhooks/xxx"}, IsActive: false})
	if err != nil {
		t.Fatalf("CreateAlertChannel second: %v", err)
	}

	list, err := st.ListAlertChannels()
	if err != nil {
		t.Fatalf("ListAlertChannels: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	if list[0].ID != created.ID {
		t.Fatalf("ORDER BY id ASC violated: %d", list[0].ID)
	}

	updated, err := st.UpdateAlertChannel(created.ID, &AlertChannel{
		Name:        "Webhook v2",
		ChannelType: "webhook",
		Config:      map[string]any{"url": "https://new.example.com"},
		Events:      []string{"provider_error"},
		IsActive:    false,
	})
	if err != nil {
		t.Fatalf("UpdateAlertChannel: %v", err)
	}
	if updated.Name != "Webhook v2" || updated.Config["url"] != "https://new.example.com" || updated.IsActive {
		t.Fatalf("update not persisted: %+v", updated)
	}

	if err := st.DeleteAlertChannel(created2.ID); err != nil {
		t.Fatalf("DeleteAlertChannel: %v", err)
	}
	if _, err := st.GetAlertChannelByID(created2.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted err = %v, want ErrNotFound", err)
	}

	// Unknown id paths.
	if err := st.DeleteAlertChannel(99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete unknown err = %v, want ErrNotFound", err)
	}
	if _, err := st.UpdateAlertChannel(99999, &AlertChannel{Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update unknown err = %v, want ErrNotFound", err)
	}
	if _, err := st.GetAlertChannelByID(99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get unknown err = %v, want ErrNotFound", err)
	}
}

func TestAlertChannelConfigEncryptedAtRest(t *testing.T) {
	st := newTestStore(t)

	secretURL := "https://discord.com/api/webhooks/SUPERSECRETTOKEN"
	created, err := st.CreateAlertChannel(&AlertChannel{
		Name:        "Discord",
		ChannelType: "discord",
		Config:      map[string]any{"webhook_url": secretURL},
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}

	// Raw config_enc column must NOT be plaintext.
	var rawEnc string
	if err := st.db.QueryRow("SELECT config_enc FROM alert_channels WHERE id = ?", created.ID).Scan(&rawEnc); err != nil {
		t.Fatalf("read config_enc: %v", err)
	}
	if strings.Contains(rawEnc, "SUPERSECRETTOKEN") {
		t.Fatalf("config_enc leaks plaintext secret: %q", rawEnc)
	}
	if strings.Contains(rawEnc, "webhook_url") {
		t.Fatalf("config_enc stores plaintext JSON: %q", rawEnc)
	}

	// Decrypted read returns the original config.
	got, err := st.GetAlertChannelByID(created.ID)
	if err != nil {
		t.Fatalf("GetAlertChannelByID: %v", err)
	}
	if got.Config["webhook_url"] != secretURL {
		t.Fatalf("decrypted config mismatch: %+v", got.Config)
	}
}
