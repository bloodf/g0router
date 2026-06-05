package store

import (
	"strings"
	"testing"
)

func TestMarkConnectionRefreshFailureSetsFlag(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "openai", AuthType: AuthTypeOAuth, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	if err := s.MarkConnectionRefreshFailure(conn.ID, "invalid_grant"); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if !got.NeedsReauth {
		t.Error("NeedsReauth should be true after failure")
	}
	if got.LastRefreshError == nil || *got.LastRefreshError != "invalid_grant" {
		t.Errorf("LastRefreshError = %v, want invalid_grant", got.LastRefreshError)
	}
}

func TestClearConnectionRefreshFailureClearsFlag(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "openai", AuthType: AuthTypeOAuth, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.MarkConnectionRefreshFailure(conn.ID, "token_expired"); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	if err := s.ClearConnectionRefreshFailure(conn.ID); err != nil {
		t.Fatalf("ClearConnectionRefreshFailure: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.NeedsReauth {
		t.Error("NeedsReauth should be false after clear")
	}
	if got.LastRefreshError != nil {
		t.Errorf("LastRefreshError should be nil after clear, got %v", *got.LastRefreshError)
	}
}

func TestMarkConnectionRefreshFailureTruncatesReason(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "openai", AuthType: AuthTypeOAuth, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	long := strings.Repeat("x", 300)
	if err := s.MarkConnectionRefreshFailure(conn.ID, long); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.LastRefreshError == nil {
		t.Fatal("LastRefreshError is nil")
	}
	if len([]rune(*got.LastRefreshError)) > maxRefreshErrorLen {
		t.Errorf("reason length = %d, want <= %d", len([]rune(*got.LastRefreshError)), maxRefreshErrorLen)
	}
}

func TestMarkConnectionRefreshFailureRoundTripViaList(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "anthropic", AuthType: AuthTypeOAuth, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.MarkConnectionRefreshFailure(conn.ID, "refresh_failed"); err != nil {
		t.Fatalf("MarkConnectionRefreshFailure: %v", err)
	}

	list, err := s.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(list))
	}
	if !list[0].NeedsReauth {
		t.Error("NeedsReauth should be true in ListConnections result")
	}
	if list[0].LastRefreshError == nil || *list[0].LastRefreshError != "refresh_failed" {
		t.Errorf("LastRefreshError via list = %v, want refresh_failed", list[0].LastRefreshError)
	}
}

func TestMarkConnectionRefreshFailureNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.MarkConnectionRefreshFailure("nonexistent", "err")
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestClearConnectionRefreshFailureNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.ClearConnectionRefreshFailure("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent id")
	}
}

func TestNewConnectionDefaultsNeedsReauthFalse(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.NeedsReauth {
		t.Error("new connection NeedsReauth should default to false")
	}
	if got.LastRefreshError != nil {
		t.Errorf("new connection LastRefreshError should default to nil, got %v", *got.LastRefreshError)
	}
}

func TestSanitizeRefreshError(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"invalid_grant", "invalid_grant"},
		{strings.Repeat("a", 201), strings.Repeat("a", 200)},
		{"", ""},
	}
	for _, tc := range cases {
		got := sanitizeRefreshError(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeRefreshError(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMigrationAddsNeedsReauthColumns(t *testing.T) {
	// openTestStore runs migrate(); if columns are absent the scan would fail.
	s := openTestStore(t)

	conn := &Connection{
		Provider:     "openai",
		AuthType:     AuthTypeOAuth,
		IsActive:     true,
		NeedsReauth:  true,
		LastRefreshError: strPtr("test_error"),
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection with new fields: %v", err)
	}
	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if !got.NeedsReauth {
		t.Error("NeedsReauth not persisted")
	}
	if got.LastRefreshError == nil || *got.LastRefreshError != "test_error" {
		t.Errorf("LastRefreshError not persisted: %v", got.LastRefreshError)
	}
}
