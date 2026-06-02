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

func TestGitHubCopilotFlowStartDeviceCode(t *testing.T) {
	var gotClientID string
	var gotScope string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login/device/code" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		gotClientID = r.Form.Get("client_id")
		gotScope = r.Form.Get("scope")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"device_code": "device-123",
			"user_code": "ABCD-EFGH",
			"verification_uri": "https://github.com/login/device",
			"expires_in": 900,
			"interval": 5
		}`))
	}))
	defer server.Close()

	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{
		ClientID:      "client-123",
		DeviceCodeURL: server.URL + "/login/device/code",
		TokenURL:      server.URL + "/login/oauth/access_token",
		Scopes:        []string{"read:user", "user:email"},
		HTTPClient:    server.Client(),
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("github-copilot") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if gotClientID != "client-123" {
		t.Errorf("client id = %q", gotClientID)
	}
	if gotScope != "read:user user:email" {
		t.Errorf("scope = %q", gotScope)
	}
	if session.Provider != ProviderID("github-copilot") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.SessionID != "device-123" {
		t.Errorf("session id = %q", session.SessionID)
	}
	if session.UserCode != "ABCD-EFGH" {
		t.Errorf("user code = %q", session.UserCode)
	}
	if session.Verification != "https://github.com/login/device" {
		t.Errorf("verification = %q", session.Verification)
	}
	if session.AuthURL != "https://github.com/login/device" {
		t.Errorf("auth url = %q", session.AuthURL)
	}
	if session.ExpiresIn != 900 {
		t.Errorf("expires in = %d", session.ExpiresIn)
	}
	if session.PollInterval != 5 {
		t.Errorf("poll interval = %d", session.PollInterval)
	}
}

func TestGitHubCopilotFlowPollPendingAndSlowDown(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		status PollStatus
	}{
		{name: "pending", body: `{"error": "authorization_pending"}`, status: PollStatusPending},
		{name: "slow down", body: `{"error": "slow_down"}`, status: PollStatusSlowDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Fatalf("parse form: %v", err)
				}
				if r.Form.Get("client_id") != "client-123" {
					t.Errorf("client id = %q", r.Form.Get("client_id"))
				}
				if r.Form.Get("device_code") != "device-123" {
					t.Errorf("device code = %q", r.Form.Get("device_code"))
				}
				if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
					t.Errorf("grant type = %q", r.Form.Get("grant_type"))
				}

				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{
				ClientID:   "client-123",
				TokenURL:   server.URL,
				HTTPClient: server.Client(),
			})

			result, err := flow.Poll(context.Background(), AuthSession{
				Provider:  ProviderID("github-copilot"),
				SessionID: "device-123",
			})
			if err != nil {
				t.Fatalf("poll: %v", err)
			}
			if result.Status != tt.status {
				t.Errorf("status = %q", result.Status)
			}
			if result.Token != nil {
				t.Errorf("token = %+v, want nil", result.Token)
			}
		})
	}
}

func TestGitHubCopilotFlowPollComplete(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "bearer",
			"expires_in":    3600,
			"scope":         "read:user user:email",
		})
	}))
	defer server.Close()

	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{
		ClientID:   "client-123",
		TokenURL:   server.URL,
		HTTPClient: server.Client(),
	})

	result, err := flow.Poll(context.Background(), AuthSession{
		Provider:  ProviderID("github-copilot"),
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
	if result.Token.Provider != ProviderID("github-copilot") {
		t.Errorf("provider = %q", result.Token.Provider)
	}
	if result.Token.AccessToken != "access-token" {
		t.Errorf("access token = %q", result.Token.AccessToken)
	}
	if result.Token.RefreshToken != "refresh-token" {
		t.Errorf("refresh token = %q", result.Token.RefreshToken)
	}
	if result.Token.TokenType != "bearer" {
		t.Errorf("token type = %q", result.Token.TokenType)
	}
	if result.Token.ExpiresAt.Before(now.Add(3590 * time.Second)) {
		t.Errorf("expires at = %v, want about one hour from now", result.Token.ExpiresAt)
	}
	if strings.Join(result.Token.Scopes, " ") != "read:user user:email" {
		t.Errorf("scopes = %+v", result.Token.Scopes)
	}
}

func TestGitHubCopilotFlowExchangeUnsupported(t *testing.T) {
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{})

	_, err := flow.Exchange(context.Background(), AuthSession{}, "callback-code")
	if err == nil {
		t.Fatal("exchange error is nil")
	}
	if !strings.Contains(err.Error(), "exchange") {
		t.Fatalf("error = %q, want exchange context", err.Error())
	}
}
