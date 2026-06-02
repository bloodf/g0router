package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAntigravityFlowStartBuildsGoogleAuthURL(t *testing.T) {
	flow := NewAntigravityFlow(AntigravityConfig{
		ClientID:    "antigravity-client",
		AuthURL:     "https://accounts.example.test/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.example.test/token",
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "email"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("antigravity") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("antigravity") {
		t.Errorf("session provider = %q", session.Provider)
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	query := authURL.Query()
	if got := query.Get("client_id"); got != "antigravity-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := query.Get("scope"); got != "openid email" {
		t.Errorf("scope = %q", got)
	}
	if got := query.Get("prompt"); got != "consent" {
		t.Errorf("prompt = %q", got)
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q", got)
	}
}

func TestAntigravityFlowExchangePostsCode(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "antigravity-access",
			"refresh_token": "antigravity-refresh",
			"token_type":    "Bearer",
			"expires_in":    1800,
			"scope":         "openid email",
		})
	}))
	defer server.Close()

	flow := NewAntigravityFlow(AntigravityConfig{
		ClientID:    "antigravity-client",
		AuthURL:     "https://accounts.example.test/o/oauth2/v2/auth",
		TokenURL:    server.URL,
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "email"},
		HTTPClient:  server.Client(),
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	before := time.Now()
	token, err := flow.Exchange(context.Background(), session, "callback-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}

	if got := gotForm.Get("grant_type"); got != "authorization_code" {
		t.Errorf("grant_type = %q", got)
	}
	if got := gotForm.Get("client_id"); got != "antigravity-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := gotForm.Get("code"); got != "callback-code" {
		t.Errorf("code = %q", got)
	}
	if gotForm.Get("code_verifier") == "" {
		t.Error("code_verifier is empty")
	}

	if token.Provider != ProviderID("antigravity") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "antigravity-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "antigravity-refresh" {
		t.Errorf("refresh token = %q", token.RefreshToken)
	}
	if token.ExpiresAt.Before(before.Add(1790 * time.Second)) {
		t.Errorf("expires at = %v, want about thirty minutes from now", token.ExpiresAt)
	}
}

func TestAntigravityFlowPollUnsupported(t *testing.T) {
	flow := NewAntigravityFlow(AntigravityConfig{})

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("antigravity")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
