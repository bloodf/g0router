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

func TestXiaomiFlowStartBuildsOAuthURL(t *testing.T) {
	flow := NewXiaomiFlow(XiaomiConfig{
		ClientID:    "xiaomi-client",
		AuthURL:     "https://account.xiaomi.example/oauth/authorize",
		TokenURL:    "https://account.xiaomi.example/oauth/token",
		RedirectURI: "http://localhost:54545/oauth/xiaomi/callback",
		Scopes:      []string{"openid", "profile"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if flow.ProviderID() != ProviderID("xiaomi") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("xiaomi") {
		t.Errorf("session provider = %q", session.Provider)
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	query := authURL.Query()
	if got := query.Get("client_id"); got != "xiaomi-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := query.Get("redirect_uri"); got != "http://localhost:54545/oauth/xiaomi/callback" {
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
}

func TestXiaomiFlowExchangePostsCode(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "xiaomi-access",
			"refresh_token": "xiaomi-refresh",
			"token_type":    "Bearer",
			"expires_in":    1800,
			"scope":         "openid profile",
		})
	}))
	defer server.Close()

	flow := NewXiaomiFlow(XiaomiConfig{
		ClientID:    "xiaomi-client",
		AuthURL:     "https://account.xiaomi.example/oauth/authorize",
		TokenURL:    server.URL,
		RedirectURI: "http://localhost:54545/oauth/xiaomi/callback",
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
	if got := gotForm.Get("client_id"); got != "xiaomi-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := gotForm.Get("redirect_uri"); got != "http://localhost:54545/oauth/xiaomi/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := gotForm.Get("code"); got != "callback-code" {
		t.Errorf("code = %q", got)
	}
	if token.Provider != ProviderID("xiaomi") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "xiaomi-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if strings.Join(token.Scopes, " ") != "openid profile" {
		t.Errorf("scopes = %+v", token.Scopes)
	}
}

func TestXiaomiFlowPollUnsupported(t *testing.T) {
	flow := NewXiaomiFlow(XiaomiConfig{})

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("xiaomi")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
