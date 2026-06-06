package store

import (
	"testing"
)

func TestListAlertChannelsDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	_, err := s.ListAlertChannels()
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestCreateAlertChannelEncryptError(t *testing.T) {
	s := openTestStore(t)
	// No enc key set, encrypt will fail
	_, err := s.CreateAlertChannel("ops", "webhook", `{}`, []string{"quota_depleted"}, true)
	if err == nil {
		t.Fatal("expected encrypt error")
	}
}

func TestCreateAlertChannelDBError(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key")
	s.db.Close()
	_, err := s.CreateAlertChannel("ops", "webhook", `{}`, []string{"quota_depleted"}, true)
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestGetAlertChannelDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	_, err := s.GetAlertChannel(1)
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestUpdateAlertChannelEncryptError(t *testing.T) {
	s := openTestStore(t)
	// No enc key set, encrypt will fail on non-empty config
	err := s.UpdateAlertChannel(1, "ops", "webhook", `{"url":"http://example.com"}`, []string{"quota_depleted"}, true)
	if err == nil {
		t.Fatal("expected encrypt error")
	}
}

func TestUpdateAlertChannelDBError(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key")
	s.db.Close()
	err := s.UpdateAlertChannel(1, "ops", "webhook", `{}`, []string{"quota_depleted"}, true)
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestDeleteAlertChannelDBError(t *testing.T) {
	s := openTestStore(t)
	s.db.Close()
	err := s.DeleteAlertChannel(1)
	if err == nil {
		t.Fatal("expected error from closed DB")
	}
}

func TestScanAlertChannelDecryptError(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key")
	if _, err := s.db.Exec(`INSERT INTO alert_channels (name, channel_type, config_enc, events_json, is_active) VALUES (?, ?, ?, ?, ?)`,
		"ops", "webhook", "not-valid-encrypted-data", `[]`, 1); err != nil {
		t.Fatalf("insert: %v", err)
	}
	_, err := s.GetAlertChannel(1)
	if err == nil {
		t.Fatal("expected decrypt error")
	}
}
