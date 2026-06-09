package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestNewTokenRandFailure(t *testing.T) {
	prev := randRead
	t.Cleanup(func() { randRead = prev })

	randRead = func(b []byte) (int, error) {
		return 0, errors.New("simulated rand failure")
	}

	st := newTestStore(t)
	sessions := NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	if _, err := sessions.Login("admin", "123456"); err == nil {
		t.Fatal("Login returned nil error with failing randRead")
	}
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("hunter2!")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if strings.Contains(hash, "hunter2!") {
		t.Fatal("hash contains plaintext password")
	}
	if !VerifyPassword(hash, "hunter2!") {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("wrong password accepted")
	}
	if VerifyPassword("garbage-not-a-hash", "hunter2!") {
		t.Fatal("garbage hash accepted")
	}

	// Salted: same password hashes differently.
	hash2, err := HashPassword("hunter2!")
	if err != nil {
		t.Fatalf("HashPassword second: %v", err)
	}
	if hash == hash2 {
		t.Fatal("two hashes of same password are identical (no salt?)")
	}
}

func TestSeedAdminCreatesOnlyWhenEmpty(t *testing.T) {
	st := newTestStore(t)
	sessions := NewSessions(st, time.Hour)

	created, err := sessions.SeedAdmin("admin", "123456")
	if err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	if !created {
		t.Fatal("SeedAdmin did not create user in empty store")
	}

	// Second seed is a no-op.
	created, err = sessions.SeedAdmin("admin", "other-password")
	if err != nil {
		t.Fatalf("SeedAdmin second: %v", err)
	}
	if created {
		t.Fatal("SeedAdmin created a user when users already exist")
	}

	// Original password still valid.
	if _, err := sessions.Login("admin", "123456"); err != nil {
		t.Fatalf("login after reseed attempt: %v", err)
	}
}

func TestSessionLoginValidateLogout(t *testing.T) {
	st := newTestStore(t)
	sessions := NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	token, err := sessions.Login("admin", "123456")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}

	user, err := sessions.Validate(token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("Validate user = %q", user.Username)
	}

	if _, err := sessions.Login("admin", "bad"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bad password err = %v, want ErrInvalidCredentials", err)
	}
	if _, err := sessions.Login("nobody", "123456"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown user err = %v, want ErrInvalidCredentials", err)
	}
	if _, err := sessions.Validate("bogus-token"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("bogus token err = %v, want ErrUnauthorized", err)
	}

	if err := sessions.Logout(token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := sessions.Validate(token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("after logout err = %v, want ErrUnauthorized", err)
	}
}

func TestSessionExpiry(t *testing.T) {
	st := newTestStore(t)
	sessions := NewSessions(st, -time.Second) // already expired on creation
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	token, err := sessions.Login("admin", "123456")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if _, err := sessions.Validate(token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expired session err = %v, want ErrUnauthorized", err)
	}
}

// fakeTokenServer emulates an OAuth token endpoint with real HTTP behavior.
func fakeTokenServer(t *testing.T) (*httptest.Server, *url.Values) {
	t.Helper()
	var lastForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		lastForm = r.PostForm

		switch r.PostForm.Get("grant_type") {
		case "authorization_code":
			if r.PostForm.Get("code") != "good-code" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "at-123",
				"refresh_token": "rt-456",
				"expires_in":    3600,
			})
		case "refresh_token":
			if r.PostForm.Get("refresh_token") != "rt-456" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "at-789",
				"refresh_token": "rt-456",
				"expires_in":    3600,
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &lastForm
}

func TestOAuthFlowStartExchangeRefresh(t *testing.T) {
	st := newTestStore(t)
	srv, lastForm := fakeTokenServer(t)

	cfg := OAuthConfig{
		Provider:     "anthropic",
		ClientID:     "client-abc",
		AuthorizeURL: "https://example.com/oauth/authorize",
		TokenURL:     srv.URL + "/token",
		RedirectURI:  "http://localhost:20128/api/oauth/anthropic/callback",
		Scopes:       []string{"inference"},
	}
	flow := NewOAuthFlow(cfg, st, srv.Client())

	authURL, state, err := flow.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if state == "" {
		t.Fatal("empty state")
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := parsed.Query()
	if q.Get("client_id") != "client-abc" || q.Get("state") != state {
		t.Fatalf("auth url query = %v", q)
	}
	if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
		t.Fatalf("missing PKCE challenge in %v", q)
	}
	if q.Get("response_type") != "code" {
		t.Fatalf("response_type = %q", q.Get("response_type"))
	}

	tok, err := flow.Exchange(state, "good-code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if tok.AccessToken != "at-123" || tok.RefreshToken != "rt-456" {
		t.Fatalf("token = %+v", tok)
	}
	if tok.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("ExpiresAt = %d not in the future", tok.ExpiresAt)
	}
	if lastForm.Get("code_verifier") == "" {
		t.Fatal("exchange did not send PKCE code_verifier")
	}
	if lastForm.Get("client_id") != "client-abc" {
		t.Fatalf("exchange client_id = %q", lastForm.Get("client_id"))
	}

	// State is single-use.
	if _, err := flow.Exchange(state, "good-code"); err == nil {
		t.Fatal("state reuse accepted")
	}

	refreshed, err := flow.Refresh("rt-456")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if refreshed.AccessToken != "at-789" {
		t.Fatalf("refreshed = %+v", refreshed)
	}
}

func TestOAuthExchangeRejectsUnknownState(t *testing.T) {
	st := newTestStore(t)
	srv, _ := fakeTokenServer(t)

	flow := NewOAuthFlow(OAuthConfig{Provider: "anthropic", ClientID: "c", AuthorizeURL: "https://example.com/a", TokenURL: srv.URL}, st, srv.Client())
	if _, err := flow.Exchange("never-issued", "good-code"); err == nil {
		t.Fatal("unknown state accepted")
	}
}

func TestOAuthExchangeSurfacesProviderError(t *testing.T) {
	st := newTestStore(t)
	srv, _ := fakeTokenServer(t)

	flow := NewOAuthFlow(OAuthConfig{Provider: "anthropic", ClientID: "c", AuthorizeURL: "https://example.com/a", TokenURL: srv.URL}, st, srv.Client())
	_, state, err := flow.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := flow.Exchange(state, "bad-code"); err == nil {
		t.Fatal("bad code accepted")
	}
}

func TestAnthropicOAuthDefaults(t *testing.T) {
	cfg := AnthropicOAuth()
	if cfg.Provider != "anthropic" {
		t.Fatalf("Provider = %q", cfg.Provider)
	}
	if cfg.ClientID == "" || cfg.AuthorizeURL == "" || cfg.TokenURL == "" {
		t.Fatalf("incomplete config: %+v", cfg)
	}
}
