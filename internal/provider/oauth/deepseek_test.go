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

func TestDeepSeekFlowProviderID(t *testing.T) {
	flow := NewDeepSeekFlow(DeepSeekConfig{})

	if flow.ProviderID() != ProviderID("deepseek") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
}

func TestDeepSeekFlowExchangePostsPasswordGrant(t *testing.T) {
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
			"access_token":  "deepseek-access",
			"refresh_token": "deepseek-refresh",
			"token_type":    "Bearer",
			"expires_in":    1800,
			"scope":         "chat",
		})
	}))
	defer server.Close()

	flow := NewDeepSeekFlow(DeepSeekConfig{
		ClientID:   "deepseek-client",
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
	})

	before := time.Now()
	token, err := flow.Exchange(context.Background(), AuthSession{
		Provider:  ProviderID("deepseek"),
		SessionID: "user@example.test",
	}, "password-123")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}

	if got := gotForm.Get("grant_type"); got != "password" {
		t.Errorf("grant_type = %q", got)
	}
	if got := gotForm.Get("client_id"); got != "deepseek-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := gotForm.Get("username"); got != "user@example.test" {
		t.Errorf("username = %q", got)
	}
	if got := gotForm.Get("password"); got != "password-123" {
		t.Errorf("password = %q", got)
	}
	if token.Provider != ProviderID("deepseek") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "deepseek-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "deepseek-refresh" {
		t.Errorf("refresh token = %q", token.RefreshToken)
	}
	if token.ExpiresAt.Before(before.Add(1790 * time.Second)) {
		t.Errorf("expires at = %v, want about 30 minutes from now", token.ExpiresAt)
	}
	if strings.Join(token.Scopes, " ") != "chat" {
		t.Errorf("scopes = %+v", token.Scopes)
	}
}

func TestDeepSeekFlowExchangeRequiresUsernameAndPassword(t *testing.T) {
	flow := NewDeepSeekFlow(DeepSeekConfig{})

	_, err := flow.Exchange(context.Background(), AuthSession{
		Provider: ProviderID("deepseek"),
	}, "password-123")
	if err == nil {
		t.Fatal("missing username error is nil")
	}

	_, err = flow.Exchange(context.Background(), AuthSession{
		Provider:  ProviderID("deepseek"),
		SessionID: "user@example.test",
	}, "")
	if err == nil {
		t.Fatal("missing password error is nil")
	}
}

func TestDeepSeekFlowStartAndPollUnsupported(t *testing.T) {
	flow := NewDeepSeekFlow(DeepSeekConfig{})

	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("start error is nil")
	}
	if _, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("deepseek")}); err == nil {
		t.Fatal("poll error is nil")
	}
}
