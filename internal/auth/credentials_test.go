package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestShouldRefreshLeadWindow(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name      string
		provider  string
		expiresAt int64
		want      bool
	}{
		{"anthropic inside 4h window", "anthropic", now + 2*3600, true},
		{"anthropic outside 4h window", "anthropic", now + 5*3600, false},
		{"gemini inside 5m window", "gemini", now + 3*60, true},
		{"gemini outside 5m window", "gemini", now + 10*60, false},
		{"xai inside 5m window", "xai", now + 3*60, true},
		{"default inside 5m window", "unknown", now + 3*60, true},
		{"no expiry", "anthropic", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &store.Connection{ExpiresAt: tt.expiresAt}
			if got := shouldRefresh(tt.provider, conn); got != tt.want {
				t.Errorf("shouldRefresh(%q, expiresAt=%d) = %v, want %v", tt.provider, tt.expiresAt, got, tt.want)
			}
		})
	}
}

func TestRefreshSingleFlight(t *testing.T) {
	st := newTestStore(t)

	var requestCount int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		mu.Lock()
		requestCount++
		mu.Unlock()
		// Slow down to ensure concurrent requests hit the lock
		time.Sleep(500 * time.Millisecond)
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-new", "refresh_token": "rt-new", "expires_in": 3600})
	}))
	defer srv.Close()

	flow := NewOAuthFlow(OAuthConfig{
		Provider: "anthropic",
		ClientID: "c",
		TokenURL: srv.URL,
	}, st, srv.Client())

	resolver := NewCredentialResolver(st, map[string]*OAuthFlow{"anthropic": flow})

	// Create provider and connection
	st.CreateProvider(&store.ProviderRecord{Name: "Anthropic", Type: "anthropic", Enabled: true})
	providers, _ := st.ListProviders()
	provider := providers[0]

	conn := &store.Connection{
		ProviderID:   provider.ID,
		Name:         "test",
		Kind:         "oauth",
		AccessToken:  "at-old",
		RefreshToken: "rt-old",
		ExpiresAt:    time.Now().Add(-time.Hour).Unix(), // expired
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	// Start all goroutines then release them together so they truly race.
	start := make(chan struct{})
	var wg sync.WaitGroup
	const n = 50
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, err := resolver.ResolveKey(provider.ID)
			if err != nil {
				t.Errorf("ResolveKey error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	mu.Lock()
	count := requestCount
	mu.Unlock()
	if count != 1 {
		t.Errorf("token endpoint called %d times, want 1", count)
	}
}

func TestMergePreservesRefreshTokenWhenEmpty(t *testing.T) {
	current := &store.Connection{AccessToken: "at-old", RefreshToken: "rt-old", ExpiresAt: 100}
	refreshed := &store.Connection{AccessToken: "at-new", RefreshToken: "", ExpiresAt: 200}
	merged, err := mergeRefreshedCredentials(current, refreshed)
	if err != nil {
		t.Fatalf("mergeRefreshedCredentials: %v", err)
	}
	if merged.AccessToken != "at-new" {
		t.Errorf("AccessToken = %q, want at-new", merged.AccessToken)
	}
	if merged.RefreshToken != "rt-old" {
		t.Errorf("RefreshToken = %q, want rt-old", merged.RefreshToken)
	}
	if merged.ExpiresAt != 200 {
		t.Errorf("ExpiresAt = %d, want 200", merged.ExpiresAt)
	}
}

func TestMergeProviderSpecificData(t *testing.T) {
	current := &store.Connection{
		AccessToken: "at-old",
		Metadata:    `{"baseUrl":"http://old","extra":"x"}`,
	}
	refreshed := &store.Connection{
		AccessToken: "at-new",
		Metadata:    `{"baseUrl":"http://new"}`,
	}
	merged, err := mergeRefreshedCredentials(current, refreshed)
	if err != nil {
		t.Fatalf("mergeRefreshedCredentials: %v", err)
	}
	var psd map[string]string
	if err := json.Unmarshal([]byte(merged.Metadata), &psd); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if psd["baseUrl"] != "http://new" {
		t.Errorf("baseUrl = %q, want http://new", psd["baseUrl"])
	}
	if psd["extra"] != "x" {
		t.Errorf("extra = %q, want x", psd["extra"])
	}
}

func TestResolveKeyNoConnection(t *testing.T) {
	st := newTestStore(t)
	resolver := NewCredentialResolver(st, nil)

	st.CreateProvider(&store.ProviderRecord{Name: "NoConn", Type: "test", Enabled: true})
	providers, _ := st.ListProviders()
	provider := providers[0]

	_, _, err := resolver.ResolveKey(provider.ID)
	if err == nil {
		t.Fatal("expected error for missing connection")
	}
	if !strings.Contains(err.Error(), "no connection for provider") {
		t.Fatalf("error = %q, want to contain 'no connection for provider'", err.Error())
	}
}

func TestResolveKeyPersistsRefreshed(t *testing.T) {
	st := newTestStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-refreshed", "expires_in": 3600})
	}))
	defer srv.Close()

	flow := NewOAuthFlow(OAuthConfig{Provider: "xai", ClientID: "c", TokenURL: srv.URL}, st, srv.Client())
	resolver := NewCredentialResolver(st, map[string]*OAuthFlow{"xai": flow})

	st.CreateProvider(&store.ProviderRecord{Name: "xAI", Type: "xai", Enabled: true})
	providers, _ := st.ListProviders()
	provider := providers[0]

	conn := &store.Connection{
		ProviderID:   provider.ID,
		Name:         "test",
		Kind:         "oauth",
		AccessToken:  "at-old",
		RefreshToken: "rt-old",
		ExpiresAt:    time.Now().Add(-time.Hour).Unix(),
	}
	st.CreateConnection(conn)

	key, psd, err := resolver.ResolveKey(provider.ID)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}
	if key.Value != "at-refreshed" {
		t.Errorf("key.Value = %q, want at-refreshed", key.Value)
	}

	// Verify persisted
	conns, _ := st.ListConnections()
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if conns[0].AccessToken != "at-refreshed" {
		t.Errorf("persisted AccessToken = %q, want at-refreshed", conns[0].AccessToken)
	}
	if psd == nil {
		t.Errorf("psd is nil")
	}
}

func TestResolveKeyInvalidMetadataErrors(t *testing.T) {
	st := newTestStore(t)
	resolver := NewCredentialResolver(st, nil)

	st.CreateProvider(&store.ProviderRecord{Name: "Test", Type: "test", Enabled: true})
	providers, _ := st.ListProviders()
	provider := providers[0]

	conn := &store.Connection{
		ProviderID:  provider.ID,
		Name:        "test",
		Kind:        "oauth",
		AccessToken: "at",
		Metadata:    `{"invalid`,
	}
	st.CreateConnection(conn)

	_, _, err := resolver.ResolveKey(provider.ID)
	if err == nil {
		t.Fatal("expected error for invalid metadata")
	}
	if !strings.Contains(err.Error(), "parse provider metadata") {
		t.Fatalf("error = %q, want to contain 'parse provider metadata'", err.Error())
	}
}

func TestResolveKeyEmptyMetadataOK(t *testing.T) {
	st := newTestStore(t)
	resolver := NewCredentialResolver(st, nil)

	st.CreateProvider(&store.ProviderRecord{Name: "Test", Type: "test", Enabled: true})
	providers, _ := st.ListProviders()
	provider := providers[0]

	conn := &store.Connection{
		ProviderID:  provider.ID,
		Name:        "test",
		Kind:        "oauth",
		AccessToken: "at",
		Metadata:    "",
	}
	st.CreateConnection(conn)

	key, psd, err := resolver.ResolveKey(provider.ID)
	if err != nil {
		t.Fatalf("ResolveKey: %v", err)
	}
	if key.Value != "at" {
		t.Errorf("key.Value = %q, want at", key.Value)
	}
	if len(psd) != 0 {
		t.Errorf("psd = %v, want empty", psd)
	}
}
