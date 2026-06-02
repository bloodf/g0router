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

func TestKimiFlowStartDeviceCode(t *testing.T) {
	var gotClientID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotClientID = r.Form.Get("client_id")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"device_code": "kimi-device",
			"user_code": "KIMI-CODE",
			"verification_uri": "https://kimi.example/device",
			"verification_uri_complete": "https://kimi.example/device?user_code=KIMI-CODE",
			"expires_in": 600,
			"interval": 3
		}`))
	}))
	defer server.Close()

	flow := NewKimiFlow(KimiFlowConfig{
		ClientID:      "kimi-client",
		DeviceCodeURL: server.URL,
		TokenURL:      server.URL + "/token",
		HTTPClient:    server.Client(),
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if flow.ProviderID() != ProviderID("kimi") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if gotClientID != "kimi-client" {
		t.Errorf("client id = %q", gotClientID)
	}
	if session.Provider != ProviderID("kimi") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.AuthURL != "https://kimi.example/device?user_code=KIMI-CODE" {
		t.Errorf("auth url = %q", session.AuthURL)
	}
	if session.SessionID != "kimi-device" {
		t.Errorf("session id = %q", session.SessionID)
	}
	if session.UserCode != "KIMI-CODE" {
		t.Errorf("user code = %q", session.UserCode)
	}
	if session.PollInterval != 3 {
		t.Errorf("poll interval = %d", session.PollInterval)
	}
}

func TestKimiFlowPollComplete(t *testing.T) {
	var gotForm url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotForm = r.PostForm

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "kimi-access",
			"refresh_token": "kimi-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "chat models",
		})
	}))
	defer server.Close()

	flow := NewKimiFlow(KimiFlowConfig{
		ClientID:   "kimi-client",
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
	})

	result, err := flow.Poll(context.Background(), AuthSession{
		Provider:  ProviderID("kimi"),
		SessionID: "kimi-device",
	})
	if err != nil {
		t.Fatalf("poll: %v", err)
	}

	if got := gotForm.Get("client_id"); got != "kimi-client" {
		t.Errorf("client id = %q", got)
	}
	if got := gotForm.Get("device_code"); got != "kimi-device" {
		t.Errorf("device code = %q", got)
	}
	if got := gotForm.Get("grant_type"); got != "urn:ietf:params:oauth:grant-type:device_code" {
		t.Errorf("grant type = %q", got)
	}
	if result.Status != PollStatusComplete {
		t.Fatalf("status = %q", result.Status)
	}
	if result.Token == nil {
		t.Fatal("token is nil")
	}
	if result.Token.Provider != ProviderID("kimi") {
		t.Errorf("provider = %q", result.Token.Provider)
	}
	if result.Token.AccessToken != "kimi-access" {
		t.Errorf("access token = %q", result.Token.AccessToken)
	}
	if strings.Join(result.Token.Scopes, " ") != "chat models" {
		t.Errorf("scopes = %+v", result.Token.Scopes)
	}
}

func TestKimiFlowExchangeUnsupported(t *testing.T) {
	flow := NewKimiFlow(KimiFlowConfig{})

	_, err := flow.Exchange(context.Background(), AuthSession{Provider: ProviderID("kimi")}, "callback-code")
	if err == nil {
		t.Fatal("exchange error is nil")
	}
	if !strings.Contains(err.Error(), "exchange") {
		t.Fatalf("error = %q, want exchange context", err.Error())
	}
}
