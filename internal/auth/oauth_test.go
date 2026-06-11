package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGeminiOAuthConfig(t *testing.T) {
	cfg := GeminiOAuth()
	if cfg.Provider != "gemini" {
		t.Errorf("Provider = %q, want gemini", cfg.Provider)
	}
	if cfg.ClientID == "" {
		t.Error("ClientID is empty")
	}
	if cfg.ClientID != GeminiOAuth().ClientID {
		t.Error("ClientID not stable across constructor calls")
	}
	if cfg.ClientSecret == "" {
		t.Error("ClientSecret is empty")
	}
	if cfg.ClientSecret != GeminiOAuth().ClientSecret {
		t.Error("ClientSecret not stable across constructor calls")
	}
	if !strings.HasPrefix(cfg.ClientSecret, "GOCSPX-") {
		t.Errorf("ClientSecret missing expected prefix")
	}
	if !strings.HasSuffix(cfg.ClientSecret, "-geV6Cu5clXFsxl") {
		t.Errorf("ClientSecret missing expected suffix")
	}
	if len(cfg.ClientSecret) != 35 {
		t.Errorf("ClientSecret length = %d, want 35", len(cfg.ClientSecret))
	}
	if cfg.AuthorizeURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Errorf("AuthorizeURL = %q, want https://accounts.google.com/o/oauth2/v2/auth", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Errorf("TokenURL = %q, want https://oauth2.googleapis.com/token", cfg.TokenURL)
	}
	if len(cfg.Scopes) == 0 {
		t.Error("Scopes is empty")
	}
}

func TestXaiOAuthConfig(t *testing.T) {
	cfg := XaiOAuth()
	if cfg.Provider != "xai" {
		t.Errorf("Provider = %q, want xai", cfg.Provider)
	}
	wantClientID := "b1a00492-073a-47ea-816f-4c329264a828"
	if cfg.ClientID != wantClientID {
		t.Errorf("ClientID = %q, want %q", cfg.ClientID, wantClientID)
	}
	if cfg.ClientSecret != "" {
		t.Errorf("ClientSecret = %q, want empty", cfg.ClientSecret)
	}
	if cfg.AuthorizeURL != "https://auth.x.ai/oauth2/authorize" {
		t.Errorf("AuthorizeURL = %q, want https://auth.x.ai/oauth2/authorize", cfg.AuthorizeURL)
	}
	if cfg.TokenURL != "https://auth.x.ai/oauth2/token" {
		t.Errorf("TokenURL = %q, want https://auth.x.ai/oauth2/token", cfg.TokenURL)
	}
	if len(cfg.Scopes) == 0 {
		t.Error("Scopes is empty")
	}
}

func TestRefreshLeadTable(t *testing.T) {
	tests := []struct {
		provider string
		want     time.Duration
	}{
		{"anthropic", 4 * time.Hour},
		{"gemini", 5 * time.Minute},
		{"xai", 5 * time.Minute},
		{"unknown", 5 * time.Minute},
	}
	for _, tt := range tests {
		got := refreshLead(tt.provider)
		if got != tt.want {
			t.Errorf("refreshLead(%q) = %v, want %v", tt.provider, got, tt.want)
		}
	}
}

func TestOAuthFlowExchangeWithClientSecret(t *testing.T) {
	st := newTestStore(t)
	var lastForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		lastForm = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-123", "refresh_token": "rt-456", "expires_in": 3600})
	}))
	defer srv.Close()

	cfg := OAuthConfig{
		Provider:     "gemini",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		AuthorizeURL: "https://example.com/authorize",
		TokenURL:     srv.URL,
		RedirectURI:  "http://localhost/cb",
	}
	flow := NewOAuthFlow(cfg, st, srv.Client())

	authURL, state, err := flow.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if authURL == "" || state == "" {
		t.Fatal("empty authURL or state")
	}

	tok, err := flow.Exchange(state, "good-code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if tok.AccessToken != "at-123" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if lastForm.Get("client_secret") != "client-secret" {
		t.Errorf("exchange client_secret = %q, want client-secret", lastForm.Get("client_secret"))
	}
	if lastForm.Get("redirect_uri") != "http://localhost/cb" {
		t.Errorf("exchange redirect_uri = %q", lastForm.Get("redirect_uri"))
	}
}

func TestOAuthFlowRefreshWithClientSecret(t *testing.T) {
	st := newTestStore(t)
	var lastForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		lastForm = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-refreshed", "expires_in": 3600})
	}))
	defer srv.Close()

	cfg := OAuthConfig{
		Provider:     "gemini",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		TokenURL:     srv.URL,
	}
	flow := NewOAuthFlow(cfg, st, srv.Client())

	tok, err := flow.Refresh("rt-1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tok.AccessToken != "at-refreshed" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if lastForm.Get("client_secret") != "client-secret" {
		t.Errorf("refresh client_secret = %q, want client-secret", lastForm.Get("client_secret"))
	}
	if lastForm.Get("client_id") != "client-id" {
		t.Errorf("refresh client_id = %q, want client-id", lastForm.Get("client_id"))
	}
}

func TestOAuthFlowStartWithRedirect(t *testing.T) {
	st := newTestStore(t)
	flow := NewOAuthFlow(OAuthConfig{
		Provider:     "xai",
		ClientID:     "c",
		AuthorizeURL: "https://example.com/authorize",
		TokenURL:     "https://example.com/token",
		RedirectURI:  "http://default/cb",
	}, st, nil)

	authURL, state, err := flow.StartWithRedirect("http://override/cb")
	if err != nil {
		t.Fatalf("StartWithRedirect: %v", err)
	}
	if state == "" {
		t.Fatal("empty state")
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	if parsed.Query().Get("redirect_uri") != "http://override/cb" {
		t.Errorf("redirect_uri = %q, want http://override/cb", parsed.Query().Get("redirect_uri"))
	}
}

func TestOAuthFlowExchangeWithRedirect(t *testing.T) {
	st := newTestStore(t)
	var lastForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		lastForm = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-123", "refresh_token": "rt-456", "expires_in": 3600})
	}))
	defer srv.Close()

	cfg := OAuthConfig{
		Provider:     "xai",
		ClientID:     "c",
		AuthorizeURL: "https://example.com/authorize",
		TokenURL:     srv.URL,
		RedirectURI:  "http://default/cb",
	}
	flow := NewOAuthFlow(cfg, st, srv.Client())

	_, state, err := flow.StartWithRedirect("http://override/cb")
	if err != nil {
		t.Fatalf("StartWithRedirect: %v", err)
	}

	tok, err := flow.ExchangeWithRedirect(state, "good-code", "http://override/cb")
	if err != nil {
		t.Fatalf("ExchangeWithRedirect: %v", err)
	}
	if tok.AccessToken != "at-123" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if lastForm.Get("redirect_uri") != "http://override/cb" {
		t.Errorf("exchange redirect_uri = %q, want http://override/cb", lastForm.Get("redirect_uri"))
	}
}
