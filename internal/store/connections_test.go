package store

import (
	"errors"
	"testing"
	"time"
)

func TestConnectionCreateAndGetByID(t *testing.T) {
	s := openTestStore(t)
	expiresAt := time.Now().Add(time.Hour).Unix()
	unavailableUntil := time.Now().Add(2 * time.Hour).Unix()

	conn := &Connection{
		Provider:             "anthropic",
		Name:                 "work",
		AuthType:             AuthTypeOAuth,
		AccessToken:          strPtr("tok-123"),
		RefreshToken:         strPtr("ref-456"),
		ExpiresAt:            &expiresAt,
		IsActive:             true,
		ProviderSpecificData: map[string]any{"tier": "pro", "org_id": "org-123"},
		AccountID:            strPtr("acct-123"),
		Email:                strPtr("user@example.com"),
		UnavailableUntil:     &unavailableUntil,
		BackoffLevel:         2,
		ModelLocks:           map[string]int64{"claude-sonnet-4": unavailableUntil},
	}

	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if conn.ID == "" {
		t.Fatal("ID should be set after create")
	}
	if conn.CreatedAt == "" {
		t.Fatal("CreatedAt should be set after create")
	}
	if conn.UpdatedAt == "" {
		t.Fatal("UpdatedAt should be set after create")
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.Provider != "anthropic" || got.Name != "work" {
		t.Errorf("got: %+v", got)
	}
	if got.AccessToken == nil || *got.AccessToken != "tok-123" {
		t.Error("access_token mismatch")
	}
	if got.RefreshToken == nil || *got.RefreshToken != "ref-456" {
		t.Error("refresh_token mismatch")
	}
	if got.ExpiresAt == nil || *got.ExpiresAt != expiresAt {
		t.Errorf("expires_at mismatch: %+v", got.ExpiresAt)
	}
	if got.AccountID == nil || *got.AccountID != "acct-123" {
		t.Error("account_id mismatch")
	}
	if got.Email == nil || *got.Email != "user@example.com" {
		t.Error("email mismatch")
	}
	if got.UnavailableUntil == nil || *got.UnavailableUntil != unavailableUntil {
		t.Errorf("unavailable_until mismatch: %+v", got.UnavailableUntil)
	}
	if got.BackoffLevel != 2 {
		t.Errorf("backoff_level = %d, want 2", got.BackoffLevel)
	}
	if got.ProviderSpecificData["tier"] != "pro" {
		t.Errorf("tier: %v", got.ProviderSpecificData["tier"])
	}
	if got.ModelLocks["claude-sonnet-4"] != unavailableUntil {
		t.Errorf("model lock: %v", got.ModelLocks["claude-sonnet-4"])
	}
}

func TestConnectionGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetConnection("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestConnectionGetByProvider(t *testing.T) {
	s := openTestStore(t)

	for _, provider := range []string{"anthropic", "anthropic", "openai"} {
		if err := s.CreateConnection(&Connection{Provider: provider, AuthType: AuthTypeAPIKey, IsActive: true}); err != nil {
			t.Fatalf("CreateConnection: %v", err)
		}
	}

	conns, err := s.GetConnections("anthropic")
	if err != nil {
		t.Fatalf("GetConnections: %v", err)
	}
	if len(conns) != 2 {
		t.Errorf("expected 2, got %d", len(conns))
	}
	for _, conn := range conns {
		if conn.Provider != "anthropic" {
			t.Errorf("provider = %q, want anthropic", conn.Provider)
		}
	}
}

func TestConnectionGetActive(t *testing.T) {
	s := openTestStore(t)

	if err := s.CreateConnection(&Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true}); err != nil {
		t.Fatalf("CreateConnection active: %v", err)
	}
	if err := s.CreateConnection(&Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: false}); err != nil {
		t.Fatalf("CreateConnection inactive: %v", err)
	}

	active, err := s.GetActiveConnections("openai")
	if err != nil {
		t.Fatalf("GetActiveConnections: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}
	if !active[0].IsActive {
		t.Error("returned connection should be active")
	}
}

func TestConnectionUpdate(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	conn.Name = "renamed"
	conn.IsActive = false
	conn.APIKey = strPtr("sk-test")
	conn.ProviderSpecificData = map[string]any{"region": "us"}
	conn.ModelLocks = map[string]int64{"gpt-4o": 123}
	if err := s.UpdateConnection(conn); err != nil {
		t.Fatalf("UpdateConnection: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.Name != "renamed" || got.IsActive {
		t.Errorf("update failed: %+v", got)
	}
	if got.APIKey == nil || *got.APIKey != "sk-test" {
		t.Error("api_key mismatch")
	}
	if got.ProviderSpecificData["region"] != "us" {
		t.Errorf("region: %v", got.ProviderSpecificData["region"])
	}
	if got.ModelLocks["gpt-4o"] != 123 {
		t.Errorf("model lock: %v", got.ModelLocks["gpt-4o"])
	}
	if got.UpdatedAt < conn.UpdatedAt {
		t.Errorf("updated_at moved backward: %q < %q", got.UpdatedAt, conn.UpdatedAt)
	}
}

func TestConnectionUpdateNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.UpdateConnection(&Connection{ID: "missing", Provider: "openai", AuthType: AuthTypeAPIKey})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestConnectionUpdateCredentialsPreservesMetadata(t *testing.T) {
	s := openTestStore(t)
	oldExpires := time.Now().Add(time.Minute).Unix()
	newExpires := time.Now().Add(time.Hour).Unix()
	unavailableUntil := time.Now().Add(2 * time.Hour).Unix()

	conn := &Connection{
		Provider:             "openai",
		Name:                 "work",
		AuthType:             AuthTypeOAuth,
		AccessToken:          strPtr("old-access"),
		RefreshToken:         strPtr("old-refresh"),
		ExpiresAt:            &oldExpires,
		IsActive:             true,
		ProviderSpecificData: map[string]any{"oauth_provider": "codex"},
		AccountID:            strPtr("acct-123"),
		Email:                strPtr("user@example.com"),
		UnavailableUntil:     &unavailableUntil,
		BackoffLevel:         2,
		ModelLocks:           map[string]int64{"gpt-4o": unavailableUntil},
	}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	if err := s.UpdateConnectionCredentials(conn.ID, strPtr("new-access"), strPtr("new-refresh"), &newExpires); err != nil {
		t.Fatalf("UpdateConnectionCredentials: %v", err)
	}

	got, err := s.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if got.Provider != "openai" || got.Name != "work" || !got.IsActive {
		t.Fatalf("metadata changed: %+v", got)
	}
	if got.AccessToken == nil || *got.AccessToken != "new-access" {
		t.Fatalf("access token = %v, want new-access", got.AccessToken)
	}
	if got.RefreshToken == nil || *got.RefreshToken != "new-refresh" {
		t.Fatalf("refresh token = %v, want new-refresh", got.RefreshToken)
	}
	if got.ExpiresAt == nil || *got.ExpiresAt != newExpires {
		t.Fatalf("expires at = %v, want %d", got.ExpiresAt, newExpires)
	}
	if got.ProviderSpecificData["oauth_provider"] != "codex" {
		t.Fatalf("provider data changed: %+v", got.ProviderSpecificData)
	}
	if got.AccountID == nil || *got.AccountID != "acct-123" {
		t.Fatalf("account ID changed: %+v", got.AccountID)
	}
	if got.Email == nil || *got.Email != "user@example.com" {
		t.Fatalf("email changed: %+v", got.Email)
	}
	if got.UnavailableUntil == nil || *got.UnavailableUntil != unavailableUntil {
		t.Fatalf("unavailable until changed: %+v", got.UnavailableUntil)
	}
	if got.BackoffLevel != 2 {
		t.Fatalf("backoff level = %d, want 2", got.BackoffLevel)
	}
	if got.ModelLocks["gpt-4o"] != unavailableUntil {
		t.Fatalf("model locks changed: %+v", got.ModelLocks)
	}
}

func TestConnectionUpdateCredentialsNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.UpdateConnectionCredentials("missing", strPtr("new-access"), strPtr("new-refresh"), nil)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestConnectionDelete(t *testing.T) {
	s := openTestStore(t)

	conn := &Connection{Provider: "openai", AuthType: AuthTypeAPIKey, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	if err := s.DeleteConnection(conn.ID); err != nil {
		t.Fatalf("DeleteConnection: %v", err)
	}

	_, err := s.GetConnection(conn.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got: %v", err)
	}
}

func TestConnectionDeleteNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.DeleteConnection("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func strPtr(s string) *string { return &s }
