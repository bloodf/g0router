package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestKiroFlowStartBuildsPKCEAuthURL(t *testing.T) {
	flow := NewKiroFlow(KiroConfig{
		ClientID:    "kiro-client",
		AuthURL:     "https://auth.kiro.dev/oauth/authorize",
		TokenURL:    "https://auth.kiro.dev/oauth/token",
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "profile"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("kiro") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("kiro") {
		t.Errorf("session provider = %q", session.Provider)
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	query := authURL.Query()
	if got := query.Get("client_id"); got != "kiro-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := query.Get("scope"); got != "openid profile" {
		t.Errorf("scope = %q", got)
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q", got)
	}
}

func TestKiroFlowExchangePostsAuthorizationCode(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "kiro-access",
			"refresh_token": "kiro-refresh",
			"token_type":    "Bearer",
			"scope":         "openid profile",
		})
	}))
	defer server.Close()

	flow := NewKiroFlow(KiroConfig{
		ClientID:    "kiro-client",
		AuthURL:     "https://auth.kiro.dev/oauth/authorize",
		TokenURL:    server.URL,
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "profile"},
		HTTPClient:  server.Client(),
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	token, err := flow.Exchange(context.Background(), session, "callback-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}

	if got := gotForm.Get("grant_type"); got != "authorization_code" {
		t.Errorf("grant_type = %q", got)
	}
	if got := gotForm.Get("code"); got != "callback-code" {
		t.Errorf("code = %q", got)
	}
	if gotForm.Get("code_verifier") == "" {
		t.Fatal("code_verifier is empty")
	}
	if token.Provider != ProviderID("kiro") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "kiro-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if strings.Join(token.Scopes, " ") != "openid profile" {
		t.Errorf("scopes = %+v", token.Scopes)
	}
}

func TestKiroFlowPollUnsupported(t *testing.T) {
	flow := NewKiroFlow(KiroConfig{})

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("kiro")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
}
