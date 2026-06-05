package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// tokenServer returns a server that responds to the token endpoint with a
// valid refreshed token payload.
func tokenServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-refresh","token_type":"bearer","expires_in":3600,"scope":"a b"}`))
	}))
}

type refreshFlow interface {
	Refresh(ctx context.Context, refreshToken string) (TokenResult, error)
}

func TestProviderRefreshSuccess(t *testing.T) {
	server := tokenServer(t)
	defer server.Close()
	client := server.Client()
	url := server.URL

	flows := map[string]refreshFlow{
		"github":      NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"gitlab":      NewGitLabFlow(GitLabConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"antigravity": NewAntigravityFlow(AntigravityConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"kimi":        NewKimiFlow(KimiFlowConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"kiro":        NewKiroFlow(KiroConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"xiaomi":      NewXiaomiFlow(XiaomiConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"deepseek":    NewDeepSeekFlow(DeepSeekConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
		"xai":         NewXAIFlow(XAIConfig{ClientID: "c", TokenURL: url, HTTPClient: client}),
	}

	for name, flow := range flows {
		t.Run(name, func(t *testing.T) {
			got, err := flow.Refresh(context.Background(), "old-refresh")
			if err != nil {
				t.Fatalf("Refresh: %v", err)
			}
			if got.AccessToken != "new-access" {
				t.Fatalf("access token = %q", got.AccessToken)
			}
			if got.RefreshToken != "new-refresh" {
				t.Fatalf("refresh token = %q", got.RefreshToken)
			}
			if got.ExpiresAt.IsZero() {
				t.Fatal("expires at not set")
			}
		})
	}
}

func TestProviderRefreshEmptyToken(t *testing.T) {
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c"})
	if _, err := flow.Refresh(context.Background(), ""); err == nil {
		t.Fatal("empty refresh token: want error")
	}
}

func TestRefreshTokenGrantBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid_grant", http.StatusBadRequest)
	}))
	defer server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	if _, err := flow.Refresh(context.Background(), "old"); err == nil {
		t.Fatal("bad status: want error")
	}
}

func TestRefreshTokenGrantDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	if _, err := flow.Refresh(context.Background(), "old"); err == nil {
		t.Fatal("decode error: want error")
	}
}

func TestRefreshTokenGrantMissingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refresh_token":"r","token_type":"bearer"}`))
	}))
	defer server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	if _, err := flow.Refresh(context.Background(), "old"); err == nil {
		t.Fatal("missing access token: want error")
	}
}

func TestRefreshTokenGrantDoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow := NewGitHubCopilotFlow(GitHubCopilotFlowConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := flow.Refresh(context.Background(), "old"); err == nil {
		t.Fatal("client.Do error: want error")
	}
}
