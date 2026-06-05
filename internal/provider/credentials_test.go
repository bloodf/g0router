package provider

import (
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/store"
)

func TestConnectionFromOAuthTokenNormalizesCodexToOpenAI(t *testing.T) {
	conn := ConnectionFromOAuthToken(oauth.TokenResult{
		Provider:     oauth.ProviderID("codex"),
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "bearer",
		ExpiresAt:    time.Unix(1700000000, 0),
	}, "work")

	if conn.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", conn.Provider)
	}
	if conn.AuthType != store.AuthTypeOAuth {
		t.Fatalf("auth type = %q, want oauth", conn.AuthType)
	}
	if conn.AccessToken == nil || *conn.AccessToken != "access" {
		t.Fatalf("access token = %v, want access", conn.AccessToken)
	}
	if conn.RefreshToken == nil || *conn.RefreshToken != "refresh" {
		t.Fatalf("refresh token = %v, want refresh", conn.RefreshToken)
	}
	if conn.ExpiresAt == nil || *conn.ExpiresAt != 1700000000 {
		t.Fatalf("expires at = %v, want unix timestamp", conn.ExpiresAt)
	}
	if conn.ProviderSpecificData["oauth_provider"] != "codex" {
		t.Fatalf("provider data = %+v, want oauth_provider codex", conn.ProviderSpecificData)
	}
}

func TestConnectionFromOAuthTokenNormalizesGitLabToGitLabDuo(t *testing.T) {
	conn := ConnectionFromOAuthToken(oauth.TokenResult{
		Provider:     oauth.ProviderID("gitlab"),
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "bearer",
		ExpiresAt:    time.Unix(1700000000, 0),
	}, "work")

	if conn.Provider != "gitlab-duo" {
		t.Fatalf("provider = %q, want gitlab-duo", conn.Provider)
	}
	if conn.ProviderSpecificData["oauth_provider"] != "gitlab" {
		t.Fatalf("provider data = %+v, want oauth_provider gitlab", conn.ProviderSpecificData)
	}
}

func TestConnectionFromOAuthTokenStoresAPIKeyFlowsAsAPIKeys(t *testing.T) {
	conn := ConnectionFromOAuthToken(oauth.TokenResult{
		Provider:    oauth.ProviderID("minimax"),
		AccessToken: "api-key",
		TokenType:   "api_key",
	}, "")

	if conn.AuthType != store.AuthTypeAPIKey {
		t.Fatalf("auth type = %q, want api_key", conn.AuthType)
	}
	if conn.APIKey == nil || *conn.APIKey != "api-key" {
		t.Fatalf("api key = %v, want api-key", conn.APIKey)
	}
	if conn.AccessToken != nil || conn.RefreshToken != nil || conn.ExpiresAt != nil {
		t.Fatalf("oauth credential fields should be nil: access=%v refresh=%v expires=%v", conn.AccessToken, conn.RefreshToken, conn.ExpiresAt)
	}
}
