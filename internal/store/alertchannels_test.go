package store

import (
	"errors"
	"testing"
)

func TestAlertChannelCreateListGetUpdateDelete(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-alert-channels")

	created, err := s.CreateAlertChannel("ops-webhook", "webhook", `{"url":"https://hooks.example.com"}`, []string{"quota_depleted", "rate_limit"}, true)
	if err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected id")
	}
	if created.Config != `{"url":"https://hooks.example.com"}` {
		t.Fatalf("config = %q", created.Config)
	}

	list, err := s.ListAlertChannels()
	if err != nil {
		t.Fatalf("ListAlertChannels: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}
	if list[0].Name != "ops-webhook" {
		t.Fatalf("name = %q", list[0].Name)
	}

	got, err := s.GetAlertChannel(created.ID)
	if err != nil {
		t.Fatalf("GetAlertChannel: %v", err)
	}
	if got.Config != `{"url":"https://hooks.example.com"}` {
		t.Fatalf("config = %q", got.Config)
	}

	if err := s.UpdateAlertChannel(created.ID, "ops-discord", "discord", `{"webhook_url":"https://discord.com/webhook"}`, []string{"budget_exhausted"}, false); err != nil {
		t.Fatalf("UpdateAlertChannel: %v", err)
	}

	updated, err := s.GetAlertChannel(created.ID)
	if err != nil {
		t.Fatalf("GetAlertChannel after update: %v", err)
	}
	if updated.Name != "ops-discord" {
		t.Fatalf("name = %q", updated.Name)
	}
	if updated.ChannelType != "discord" {
		t.Fatalf("channel_type = %q", updated.ChannelType)
	}
	if updated.Config != `{"webhook_url":"https://discord.com/webhook"}` {
		t.Fatalf("config = %q", updated.Config)
	}
	if len(updated.Events) != 1 || updated.Events[0] != "budget_exhausted" {
		t.Fatalf("events = %v", updated.Events)
	}
	if updated.IsActive {
		t.Fatal("expected inactive")
	}

	if err := s.DeleteAlertChannel(created.ID); err != nil {
		t.Fatalf("DeleteAlertChannel: %v", err)
	}

	_, err = s.GetAlertChannel(created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAlertChannelConfigEncryptedInDB(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-alert-channels")

	created, err := s.CreateAlertChannel("ops-webhook", "webhook", `{"url":"https://hooks.example.com","token":"secret"}`, []string{"quota_depleted"}, true)
	if err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}

	var configEnc string
	err = s.db.QueryRow("SELECT config_enc FROM alert_channels WHERE id = ?", created.ID).Scan(&configEnc)
	if err != nil {
		t.Fatalf("query db: %v", err)
	}
	if configEnc == "" {
		t.Fatal("config_enc should not be empty")
	}
	if configEnc == `{"url":"https://hooks.example.com","token":"secret"}` {
		t.Fatal("config_enc should be encrypted")
	}
}

func TestAlertChannelUpdatePreservesConfigWhenEmpty(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-alert-channels")

	created, err := s.CreateAlertChannel("ops-webhook", "webhook", `{"url":"https://hooks.example.com"}`, []string{"quota_depleted"}, true)
	if err != nil {
		t.Fatalf("CreateAlertChannel: %v", err)
	}

	if err := s.UpdateAlertChannel(created.ID, "ops-renamed", "webhook", "", []string{"quota_depleted"}, true); err != nil {
		t.Fatalf("UpdateAlertChannel: %v", err)
	}

	got, err := s.GetAlertChannel(created.ID)
	if err != nil {
		t.Fatalf("GetAlertChannel: %v", err)
	}
	if got.Config != `{"url":"https://hooks.example.com"}` {
		t.Fatalf("config = %q, want preserved", got.Config)
	}
}

func TestAlertChannelGetNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-alert-channels")

	_, err := s.GetAlertChannel(999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAlertChannelDeleteNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-alert-channels")

	err := s.DeleteAlertChannel(999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAlertChannelUpdateNotFound(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-alert-channels")

	err := s.UpdateAlertChannel(999, "x", "webhook", "{}", []string{}, true)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
