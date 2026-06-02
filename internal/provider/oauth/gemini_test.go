package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGeminiFlowStartBuildsPKCEAuthURL(t *testing.T) {
	flow := NewGeminiFlow(GeminiConfig{
		ClientID:    "gemini-client",
		AuthURL:     "https://accounts.example.test/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.example.test/token",
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "email", "profile"},
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("gemini") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("gemini") {
		t.Errorf("session provider = %q", session.Provider)
	}
	if session.SessionID == "" {
		t.Fatal("session id is empty")
	}

	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	query := authURL.Query()
	if got := query.Get("client_id"); got != "gemini-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := query.Get("redirect_uri"); got != "http://localhost:54545/oauth/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := query.Get("response_type"); got != "code" {
		t.Errorf("response_type = %q", got)
	}
	if got := query.Get("scope"); got != "openid email profile" {
		t.Errorf("scope = %q", got)
	}
	if got := query.Get("access_type"); got != "offline" {
		t.Errorf("access_type = %q", got)
	}
	if got := query.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q", got)
	}
	if query.Get("state") == "" {
		t.Error("state is empty")
	}
	if query.Get("code_challenge") == "" {
		t.Error("code challenge is empty")
	}
}

func TestGeminiFlowExchangePostsCodeAndVerifier(t *testing.T) {
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
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "gemini-access",
			"refresh_token": "gemini-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "openid email profile",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	flow := NewGeminiFlow(GeminiConfig{
		ClientID:    "gemini-client",
		AuthURL:     "https://accounts.example.test/o/oauth2/v2/auth",
		TokenURL:    server.URL,
		RedirectURI: "http://localhost:54545/oauth/callback",
		Scopes:      []string{"openid", "email", "profile"},
		HTTPClient:  server.Client(),
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	authURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	before := time.Now()
	token, err := flow.Exchange(context.Background(), session, "callback-code")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}

	if got := gotForm.Get("grant_type"); got != "authorization_code" {
		t.Errorf("grant_type = %q", got)
	}
	if got := gotForm.Get("client_id"); got != "gemini-client" {
		t.Errorf("client_id = %q", got)
	}
	if got := gotForm.Get("redirect_uri"); got != "http://localhost:54545/oauth/callback" {
		t.Errorf("redirect_uri = %q", got)
	}
	if got := gotForm.Get("code"); got != "callback-code" {
		t.Errorf("code = %q", got)
	}

	verifier := gotForm.Get("code_verifier")
	if verifier == "" {
		t.Fatal("code_verifier is empty")
	}
	sum := sha256.Sum256([]byte(verifier))
	wantChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if got := authURL.Query().Get("code_challenge"); got != wantChallenge {
		t.Errorf("code_challenge = %q, want %q", got, wantChallenge)
	}

	if token.Provider != ProviderID("gemini") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "gemini-access" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "gemini-refresh" {
		t.Errorf("refresh token = %q", token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("token type = %q", token.TokenType)
	}
	if strings.Join(token.Scopes, " ") != "openid email profile" {
		t.Errorf("scopes = %+v", token.Scopes)
	}
	if !token.ExpiresAt.After(before) {
		t.Errorf("expires at = %v, want after %v", token.ExpiresAt, before)
	}
}

func TestGeminiFlowPollUnsupported(t *testing.T) {
	flow := NewGeminiFlow(GeminiConfig{})

	_, err := flow.Poll(context.Background(), AuthSession{Provider: ProviderID("gemini")})
	if err == nil {
		t.Fatal("poll error is nil")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll context", err.Error())
	}
}
