package store

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"
)

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func TestCreateDashboardSession(t *testing.T) {
	s := openTestStore(t)

	expiresAt := time.Now().Add(time.Hour)
	rawToken := "test-token-123"
	if err := s.CreateDashboardSession(1, rawToken, "Mozilla/5.0", "127.0.0.1", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession: %v", err)
	}

	got, err := s.GetDashboardSessionByTokenHash(sha256Hash(rawToken))
	if err != nil {
		t.Fatalf("GetDashboardSessionByTokenHash: %v", err)
	}
	if got.TokenHash != sha256Hash(rawToken) {
		t.Errorf("TokenHash = %q, want %q", got.TokenHash, sha256Hash(rawToken))
	}
	if got.UserID != 1 {
		t.Errorf("UserID = %d, want 1", got.UserID)
	}
	if got.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %q, want Mozilla/5.0", got.UserAgent)
	}
	if got.IP != "127.0.0.1" {
		t.Errorf("IP = %q, want 127.0.0.1", got.IP)
	}
	if got.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
	if got.LastSeenAt == "" {
		t.Error("LastSeenAt should be set")
	}
	if got.ExpiresAt == "" {
		t.Error("ExpiresAt should be set")
	}
}

func TestGetDashboardSessionByRawToken(t *testing.T) {
	s := openTestStore(t)

	expiresAt := time.Now().Add(time.Hour)
	rawToken := "raw-token-456"
	if err := s.CreateDashboardSession(2, rawToken, "", "", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession: %v", err)
	}

	got, err := s.GetDashboardSessionByRawToken(rawToken)
	if err != nil {
		t.Fatalf("GetDashboardSessionByRawToken: %v", err)
	}
	if got.TokenHash != sha256Hash(rawToken) {
		t.Errorf("TokenHash = %q, want %q", got.TokenHash, sha256Hash(rawToken))
	}
	if got.UserID != 2 {
		t.Errorf("UserID = %d, want 2", got.UserID)
	}
}

func TestGetDashboardSessionByTokenHashNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetDashboardSessionByTokenHash("nonexistent-hash")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestGetDashboardSessionByRawTokenNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetDashboardSessionByRawToken("nonexistent-token")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestTouchDashboardSession(t *testing.T) {
	s := openTestStore(t)

	expiresAt := time.Now().Add(time.Hour)
	rawToken := "touch-token"
	if err := s.CreateDashboardSession(1, rawToken, "", "", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession: %v", err)
	}

	tokenHash := sha256Hash(rawToken)

	// Manually set last_seen_at to 2 minutes ago so first touch updates it
	_, err := s.db.Exec(
		"UPDATE dashboard_sessions SET last_seen_at = datetime('now', '-2 minutes') WHERE token_hash = ?",
		tokenHash,
	)
	if err != nil {
		t.Fatalf("set old last_seen_at: %v", err)
	}

	before, err := s.GetDashboardSessionByTokenHash(tokenHash)
	if err != nil {
		t.Fatalf("GetDashboardSessionByTokenHash: %v", err)
	}

	// First touch should update last_seen_at
	if err := s.TouchDashboardSession(tokenHash); err != nil {
		t.Fatalf("TouchDashboardSession: %v", err)
	}

	after, err := s.GetDashboardSessionByTokenHash(tokenHash)
	if err != nil {
		t.Fatalf("GetDashboardSessionByTokenHash after touch: %v", err)
	}

	if after.LastSeenAt == before.LastSeenAt {
		t.Error("expected last_seen_at to update after first touch")
	}

	// Second touch within same minute should be a no-op
	if err := s.TouchDashboardSession(tokenHash); err != nil {
		t.Fatalf("TouchDashboardSession second: %v", err)
	}

	second, err := s.GetDashboardSessionByTokenHash(tokenHash)
	if err != nil {
		t.Fatalf("GetDashboardSessionByTokenHash after second touch: %v", err)
	}

	if second.LastSeenAt != after.LastSeenAt {
		t.Error("expected last_seen_at to stay same after second touch within same minute")
	}
}

func TestDeleteDashboardSession(t *testing.T) {
	s := openTestStore(t)

	expiresAt := time.Now().Add(time.Hour)
	rawToken := "delete-token"
	if err := s.CreateDashboardSession(1, rawToken, "", "", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession: %v", err)
	}

	tokenHash := sha256Hash(rawToken)
	if err := s.DeleteDashboardSession(tokenHash); err != nil {
		t.Fatalf("DeleteDashboardSession: %v", err)
	}

	_, err := s.GetDashboardSessionByTokenHash(tokenHash)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteDashboardSessionsByUserID(t *testing.T) {
	s := openTestStore(t)

	expiresAt := time.Now().Add(time.Hour)
	if err := s.CreateDashboardSession(1, "token-a", "", "", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession token-a: %v", err)
	}
	if err := s.CreateDashboardSession(1, "token-b", "", "", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession token-b: %v", err)
	}
	if err := s.CreateDashboardSession(2, "token-c", "", "", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession token-c: %v", err)
	}

	if err := s.DeleteDashboardSessionsByUserID(1); err != nil {
		t.Fatalf("DeleteDashboardSessionsByUserID: %v", err)
	}

	_, err := s.GetDashboardSessionByTokenHash(sha256Hash("token-a"))
	if err == nil {
		t.Fatal("expected token-a to be deleted")
	}
	_, err = s.GetDashboardSessionByTokenHash(sha256Hash("token-b"))
	if err == nil {
		t.Fatal("expected token-b to be deleted")
	}

	// Session for user 2 should still exist
	got, err := s.GetDashboardSessionByTokenHash(sha256Hash("token-c"))
	if err != nil {
		t.Fatalf("expected token-c to survive: %v", err)
	}
	if got.UserID != 2 {
		t.Errorf("UserID = %d, want 2", got.UserID)
	}
}

func TestPurgeExpiredDashboardSessions(t *testing.T) {
	s := openTestStore(t)

	expired := time.Now().Add(-time.Hour)
	notExpired := time.Now().Add(time.Hour)

	if err := s.CreateDashboardSession(1, "expired-token", "", "", expired); err != nil {
		t.Fatalf("CreateDashboardSession expired: %v", err)
	}
	if err := s.CreateDashboardSession(1, "valid-token", "", "", notExpired); err != nil {
		t.Fatalf("CreateDashboardSession valid: %v", err)
	}

	if err := s.PurgeExpiredDashboardSessions(); err != nil {
		t.Fatalf("PurgeExpiredDashboardSessions: %v", err)
	}

	_, err := s.GetDashboardSessionByTokenHash(sha256Hash("expired-token"))
	if err == nil {
		t.Fatal("expected expired token to be purged")
	}

	got, err := s.GetDashboardSessionByTokenHash(sha256Hash("valid-token"))
	if err != nil {
		t.Fatalf("expected valid token to survive: %v", err)
	}
	if got.UserID != 1 {
		t.Errorf("UserID = %d, want 1", got.UserID)
	}
}

func TestDashboardSessionSurviveReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "survive.db")

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour)
	rawToken := "survive-token"
	if err := s.CreateDashboardSession(42, rawToken, "agent", "1.2.3.4", expiresAt); err != nil {
		t.Fatalf("CreateDashboardSession: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore reopen: %v", err)
	}
	defer s2.Close()

	got, err := s2.GetDashboardSessionByRawToken(rawToken)
	if err != nil {
		t.Fatalf("GetDashboardSessionByRawToken after reopen: %v", err)
	}
	if got.UserID != 42 {
		t.Errorf("UserID = %d, want 42", got.UserID)
	}
	if got.UserAgent != "agent" {
		t.Errorf("UserAgent = %q, want agent", got.UserAgent)
	}
	if got.IP != "1.2.3.4" {
		t.Errorf("IP = %q, want 1.2.3.4", got.IP)
	}
}
