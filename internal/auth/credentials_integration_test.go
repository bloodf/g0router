package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/translation"
)

// seedGeminiProvider inserts a provider with the fixed id "gemini" so that
// router.Resolve("gemini/...") finds a matching store record.
func seedGeminiProvider(t *testing.T, st *store.Store) {
	t.Helper()
	now := time.Now().Add(-time.Hour).Unix()
	_, err := st.DB().Exec(
		"INSERT INTO providers (id, name, type, base_url, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"gemini", "Gemini", "gemini", "", 1, now, now,
	)
	if err != nil {
		t.Fatalf("insert provider: %v", err)
	}
}

// TestCredentialResolverRouterIntegration wires a real CredentialResolver
// (backed by a real store.Store with a seeded provider connection) into a
// Router via SetKeyResolver and asserts Resolve returns the connection's
// access token and provider-specific metadata.
func TestCredentialResolverRouterIntegration(t *testing.T) {
	st := newTestStore(t)
	seedGeminiProvider(t, st)

	conn := &store.Connection{
		ProviderID:  "gemini",
		Name:        "gemini-oauth",
		Kind:        "oauth",
		AccessToken: "access-token-initial",
		Metadata:    `{"project_id":"p-123","region":"us-central1"}`,
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resolver := NewCredentialResolver(st, nil)
	router := inference.NewRouter(translation.NewRegistry())
	router.SetKeyResolver(resolver)

	_, key, err := router.Resolve("gemini/gemini-1.5-pro")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if key.Provider != "gemini" {
		t.Errorf("key.Provider = %q, want gemini", key.Provider)
	}
	if key.Value != "access-token-initial" {
		t.Errorf("key.Value = %q, want access-token-initial", key.Value)
	}
	if key.ProviderSpecificData["project_id"] != "p-123" {
		t.Errorf("project_id = %q, want p-123", key.ProviderSpecificData["project_id"])
	}
	if key.ProviderSpecificData["region"] != "us-central1" {
		t.Errorf("region = %q, want us-central1", key.ProviderSpecificData["region"])
	}
}

// TestCredentialResolverRouterIntegrationRefresh exercises the refresh path:
// a near-expiry connection causes CredentialResolver to call the provider's
// OAuth flow, which hits an httptest token endpoint, persists the new token,
// and returns it through Router.Resolve.
func TestCredentialResolverRouterIntegrationRefresh(t *testing.T) {
	st := newTestStore(t)
	seedGeminiProvider(t, st)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token-refreshed",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	cfg := OAuthConfig{
		Provider: "gemini",
		ClientID: "test-client-id",
		TokenURL: srv.URL,
	}
	flow := NewOAuthFlow(cfg, st, srv.Client())

	conn := &store.Connection{
		ProviderID:   "gemini",
		Name:         "gemini-oauth",
		Kind:         "oauth",
		AccessToken:  "access-token-initial",
		RefreshToken: "refresh-token-orig",
		// Gemini refresh lead is 5 minutes; 1 minute until expiry triggers refresh.
		ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
		Metadata:  `{"project_id":"p-123"}`,
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resolver := NewCredentialResolver(st, map[string]*OAuthFlow{"gemini": flow})
	router := inference.NewRouter(translation.NewRegistry())
	router.SetKeyResolver(resolver)

	_, key, err := router.Resolve("gemini/gemini-1.5-pro")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if key.Provider != "gemini" {
		t.Errorf("key.Provider = %q, want gemini", key.Provider)
	}
	if key.Value != "access-token-refreshed" {
		t.Errorf("key.Value = %q, want access-token-refreshed", key.Value)
	}
	if key.ProviderSpecificData["project_id"] != "p-123" {
		t.Errorf("project_id = %q, want p-123", key.ProviderSpecificData["project_id"])
	}

	// The refreshed token must have been persisted back to the store.
	stored, err := st.GetConnection(conn.ID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.AccessToken != "access-token-refreshed" {
		t.Errorf("stored AccessToken = %q, want access-token-refreshed", stored.AccessToken)
	}
	if stored.ExpiresAt == 0 {
		t.Error("stored ExpiresAt unset after refresh")
	}
}
