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

func TestXAIFlowStartBuildsPKCEAuthURL(t *testing.T) {
	flow := NewXAIFlow(XAIConfig{
		ClientID:    "xai-client",
		AuthURL:     "https://accounts.x.ai/oauth/authorize",
		TokenURL:    "https://accounts.x.ai/oauth/token",
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "profile"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("xai") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("xai") {
		t.Errorf("session provider = %q", session.Provider)
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	query := authURL.Query()
	if got := query.Get("client_id"); got != "xai-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := query.Get("redirect_uri"); got != "http://localhost:54545/oauth/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := query.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q", got)
	}
	if got := query.Get("scope"); got != "openid profile" {
		t.Errorf("scope = %q", got)
	}
	if query.Get("state") == "" {
		t.Error("state is empty")
	}
	if query.Get("code_challenge") == "" {
		t.Error("code challenge is empty")
	}
}

func TestXAIFlowExchangePostsAuthorizationCode(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "xai-access",
			"refresh_token": "xai-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "openid profile",
		})
	}))
	defer server.Close()

	flow := NewXAIFlow(XAIConfig{
		ClientID:    "xai-client",
		AuthURL:     "https://accounts.x.ai/oauth/authorize",
		TokenURL:    server.URL,
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "profile"},
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
	if got := gotForm.Get("client_id"); got != "xai-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := gotForm.Get("code"); got != "callback-code" {
		t.Errorf("code = %q", got)
	}
	if gotForm.Get("code_verifier") == "" {
		t.Fatal("code_verifier is empty")
	}
	if token.Provider != ProviderID("xai") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "xai-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "xai-refresh" {
		t.Errorf("refresh token = %q", token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("token type = %q", token.TokenType)
	}
	if strings.Join(token.Scopes, " ") != "openid profile" {
		t.Errorf("scopes = %+v", token.Scopes)
	}
	if !token.ExpiresAt.After(before) {
		t.Errorf("expires at = %v, want after %v", token.ExpiresAt, before)
	}
}

func TestXAIFlowPollUnsupported(t *testing.T) {
	flow := NewXAIFlow(XAIConfig{})

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("xai")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
}
