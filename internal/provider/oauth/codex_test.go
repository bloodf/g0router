package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCodexFlowStartDeviceCode(t *testing.T) {
	var gotPath string
	var gotClientID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotClientID = r.Form.Get("client_id")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"device_code": "device-123",
			"user_code": "ABCD-EFGH",
			"verification_uri": "https://auth.openai.com/activate",
			"verification_uri_complete": "https://auth.openai.com/activate?user_code=ABCD-EFGH",
			"expires_in": 900,
			"interval": 7
		}`))
	}))
	defer server.Close()

	flow := NewCodexFlow(CodexFlowConfig{
		ClientID:      "client-123",
		DeviceCodeURL: server.URL + "/oauth/device/code",
		TokenURL:      server.URL + "/oauth/token",
		HTTPClient:    server.Client(),
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("codex") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if gotPath != "/oauth/device/code" {
		t.Errorf("path = %q", gotPath)
	}
	if gotClientID != "client-123" {
		t.Errorf("client id = %q", gotClientID)
	}
	if session.Provider != ProviderID("codex") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.SessionID != "device-123" {
		t.Errorf("session id = %q", session.SessionID)
	}
	if session.UserCode != "ABCD-EFGH" {
		t.Errorf("user code = %q", session.UserCode)
	}
	if session.Verification != "https://auth.openai.com/activate" {
		t.Errorf("verification = %q", session.Verification)
	}
	if session.AuthURL != "https://auth.openai.com/activate?user_code=ABCD-EFGH" {
		t.Errorf("auth url = %q", session.AuthURL)
	}
	if session.ExpiresIn != 900 {
		t.Errorf("expires in = %d", session.ExpiresIn)
	}
	if session.PollInterval != 7 {
		t.Errorf("poll interval = %d", session.PollInterval)
	}
}

func TestCodexFlowPollPending(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("device_code") != "device-123" {
			t.Errorf("device code = %q", r.Form.Get("device_code"))
		}
		if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
			t.Errorf("grant type = %q", r.Form.Get("grant_type"))
		}

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "authorization_pending"}`))
	}))
	defer server.Close()

	flow := NewCodexFlow(CodexFlowConfig{
		ClientID:   "client-123",
		TokenURL:   server.URL + "/oauth/token",
		HTTPClient: server.Client(),
	})

	result, err := flow.Poll(context.Background(), AuthSession{
		Provider:  ProviderID("codex"),
		SessionID: "device-123",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.Status != PollStatusPending {
		t.Errorf("status = %q", result.Status)
	}
	if result.Token != nil {
		t.Errorf("token = %+v, want nil", result.Token)
	}
}

func TestCodexFlowPollComplete(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "openid profile offline_access",
		})
	}))
	defer server.Close()

	flow := NewCodexFlow(CodexFlowConfig{
		ClientID:   "client-123",
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
	})

	result, err := flow.Poll(context.Background(), AuthSession{
		Provider:  ProviderID("codex"),
		SessionID: "device-123",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}

	if result.Status != PollStatusComplete {
		t.Fatalf("status = %q", result.Status)
	}
	if result.Token == nil {
		t.Fatal("token is nil")
	}
	if result.Token.Provider != ProviderID("codex") {
		t.Errorf("provider = %q", result.Token.Provider)
	}
	if result.Token.AccessToken != "access-token" {
		t.Errorf("access token = %q", result.Token.AccessToken)
	}
	if result.Token.RefreshToken != "refresh-token" {
		t.Errorf("refresh token = %q", result.Token.RefreshToken)
	}
	if result.Token.TokenType != "Bearer" {
		t.Errorf("token type = %q", result.Token.TokenType)
	}
	if result.Token.ExpiresAt.Before(now.Add(3590 * time.Second)) {
		t.Errorf("expires at = %v, want about one hour from now", result.Token.ExpiresAt)
	}
	if strings.Join(result.Token.Scopes, " ") != "openid profile offline_access" {
		t.Errorf("scopes = %+v", result.Token.Scopes)
	}
}

func TestCodexFlowExchangeUnsupported(t *testing.T) {
	flow := NewCodexFlow(CodexFlowConfig{})

	_, err := flow.Exchange(context.Background(), AuthSession{}, "callback-code")
	if err == nil {
		t.Fatal("exchange error is nil")
	}
}
