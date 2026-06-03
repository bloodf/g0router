package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAnthropicFlowStartBuildsPKCEAuthURL(t *testing.T) {
	flow := NewAnthropicFlowWithConfig(AnthropicConfig{
		ClientID:    "client-1",
		AuthURL:     "https://auth.example.test/oauth/authorize",
		TokenURL:    "https://auth.example.test/oauth/token",
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"org:create_api_key"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if session.Provider != ProviderID("anthropic") {
		t.Fatalf("provider = %q, want anthropic", session.Provider)
	}
	if session.SessionID == "" {
		t.Fatal("session id is empty")
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	query := authURL.Query()
	if got := query.Get("client_id"); got != "client-1" {
		t.Errorf("client_id = %q", got)
	}
	if got := query.Get("redirect_uri"); got != "http://localhost:54545/oauth/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := query.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q", got)
	}
	if got := query.Get("scope"); got != "org:create_api_key" {
		t.Errorf("scope = %q", got)
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q", got)
	}
	if query.Get("state") == "" {
		t.Error("state is empty")
	}
	if query.Get("code_challenge") == "" {
		t.Error("code challenge is empty")
	}
}

func TestAnthropicFlowExchangePostsCodeAndVerifier(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "org:create_api_key",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	flow := NewAnthropicFlowWithConfig(AnthropicConfig{
		ClientID:    "client-1",
		AuthURL:     "https://auth.example.test/oauth/authorize",
		TokenURL:    server.URL,
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"org:create_api_key"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	before := time.Now()
	token, err := flow.Exchange(context.Background(), session, "callback-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}

	if got := gotForm.Get("grant_type"); got != "authorization_code" {
		t.Errorf("grant_type = %q", got)
	}
	if got := gotForm.Get("client_id"); got != "client-1" {
		t.Errorf("client_id = %q", got)
	}
	if got := gotForm.Get("redirect_uri"); got != "http://localhost:54545/oauth/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := gotForm.Get("code"); got != "callback-code" {
		t.Errorf("code = %q", got)
	}

	verifier := gotForm.Get("code_verifier")
	if verifier == "" {
		t.Fatal("code_verifier is empty")
	}
	sum := sha256.Sum256([]byte(verifier))
	wantChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if got := authURL.Query().Get("code_challenge"); got != wantChallenge {
		t.Errorf("code_challenge = %q, want %q", got, wantChallenge)
	}

	if token.Provider != ProviderID("anthropic") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "access-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh-token" {
		t.Errorf("refresh token = %q", token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("token type = %q", token.TokenType)
	}
	if len(token.Scopes) != 1 || token.Scopes[0] != "org:create_api_key" {
		t.Errorf("scopes = %+v", token.Scopes)
	}
	if !token.ExpiresAt.After(before) {
		t.Errorf("expires at = %v, want after %v", token.ExpiresAt, before)
	}
}

func TestAnthropicFlowRefreshPostsRefreshTokenGrant(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "org:create_api_key",
		})
	}))
	defer server.Close()

	flow := NewAnthropicFlowWithConfig(AnthropicConfig{
		ClientID:   "client-1",
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
	})

	before := time.Now()
	token, err := flow.Refresh(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if got := gotForm.Get("grant_type"); got != "refresh_token" {
		t.Errorf("grant_type = %q, want refresh_token", got)
	}
	if got := gotForm.Get("client_id"); got != "client-1" {
		t.Errorf("client_id = %q, want client-1", got)
	}
	if got := gotForm.Get("refresh_token"); got != "old-refresh" {
		t.Errorf("refresh_token = %q, want old-refresh", got)
	}
	if token.Provider != ProviderID("anthropic") {
		t.Errorf("provider = %q, want anthropic", token.Provider)
	}
	if token.AccessToken != "new-access" || token.RefreshToken != "new-refresh" {
		t.Fatalf("token = %+v, want refreshed credentials", token)
	}
	if !token.ExpiresAt.After(before) {
		t.Fatalf("expires at = %v, want after %v", token.ExpiresAt, before)
	}
}

func TestAnthropicFlowPollUnsupported(t *testing.T) {
	flow := NewAnthropicFlow()

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("anthropic")})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
