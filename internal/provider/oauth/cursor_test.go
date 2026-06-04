package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCursorFlowStartBuildsOMPLoginDeepControlURL(t *testing.T) {
	flow := NewCursorFlowWithConfig(CursorConfig{
		LoginURL: "https://cursor.example/loginDeepControl",
		NewUUID:  func() (string, error) { return "00000000-0000-4000-8000-000000000001", nil },
	})

	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	if flow.ProviderID() != ProviderID("cursor") {
		t.Errorf("provider id = %q", flow.ProviderID())
	}
	if session.Provider != ProviderID("cursor") {
		t.Fatalf("provider = %q, want cursor", session.Provider)
	}
	if session.UserCode != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("user code = %q", session.UserCode)
	}
	if session.Verification != session.AuthURL {
		t.Fatalf("verification = %q, want auth url", session.Verification)
	}
	if session.PollInterval != 1 {
		t.Fatalf("poll interval = %d, want 1", session.PollInterval)
	}

	uuid, verifier, err := parseCursorSessionID(session.SessionID)
	if err != nil {
		t.Fatalf("parse session id: %v", err)
	}
	if uuid != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("session uuid = %q", uuid)
	}
	if verifier == "" {
		t.Fatal("verifier is empty")
	}

	loginURL, err := url.Parse(session.AuthURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	query := loginURL.Query()
	if got := query.Get("uuid"); got != uuid {
		t.Errorf("uuid = %q, want %q", got, uuid)
	}
	if got := query.Get("challenge"); got != codeChallenge(verifier) {
		t.Errorf("challenge = %q, want %q", got, codeChallenge(verifier))
	}
	if got := query.Get("mode"); got != "login" {
		t.Errorf("mode = %q, want login", got)
	}
	if got := query.Get("redirectTarget"); got != "cli" {
		t.Errorf("redirectTarget = %q, want cli", got)
	}
	for _, forbidden := range []string{"client_id", "redirect_uri", "response_type", "state", "code_challenge"} {
		if query.Get(forbidden) != "" {
			t.Errorf("%s = %q, want absent", forbidden, query.Get(forbidden))
		}
	}
}

func TestCursorFlowPollPendingOn404(t *testing.T) {
	var gotUUID string
	var gotVerifier string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		gotUUID = r.URL.Query().Get("uuid")
		gotVerifier = r.URL.Query().Get("verifier")
		http.NotFound(w, r)
	}))
	defer server.Close()

	flow := NewCursorFlowWithConfig(CursorConfig{
		LoginURL:   "https://cursor.example/loginDeepControl",
		PollURL:    server.URL + "/auth/poll",
		HTTPClient: server.Client(),
		NewUUID:    func() (string, error) { return "00000000-0000-4000-8000-000000000002", nil },
	})
	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	_, verifier, err := parseCursorSessionID(session.SessionID)
	if err != nil {
		t.Fatalf("parse session id: %v", err)
	}

	result, err := flow.Poll(context.Background(), session)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.Status != PollStatusPending {
		t.Fatalf("status = %q, want pending", result.Status)
	}
	if gotUUID != session.UserCode {
		t.Fatalf("uuid = %q, want %q", gotUUID, session.UserCode)
	}
	if gotVerifier != verifier {
		t.Fatalf("verifier = %q, want %q", gotVerifier, verifier)
	}
}

func TestCursorFlowPollCompleteStoresAccessRefreshAndExpiry(t *testing.T) {
	expiresAt := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	accessToken := jwtWithExpiry(expiresAt)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"accessToken":  accessToken,
			"refreshToken": "cursor-refresh-token",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	flow := NewCursorFlowWithConfig(CursorConfig{
		LoginURL:   "https://cursor.example/loginDeepControl",
		PollURL:    server.URL + "/auth/poll",
		HTTPClient: server.Client(),
		NewUUID:    func() (string, error) { return "00000000-0000-4000-8000-000000000003", nil },
	})
	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	result, err := flow.Poll(context.Background(), session)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	if result.Status != PollStatusComplete {
		t.Fatalf("status = %q, want complete", result.Status)
	}
	if result.Token == nil {
		t.Fatal("token is nil")
	}
	if result.Token.Provider != ProviderID("cursor") {
		t.Errorf("provider = %q", result.Token.Provider)
	}
	if result.Token.AccessToken != accessToken {
		t.Errorf("access token = %q", result.Token.AccessToken)
	}
	if result.Token.RefreshToken != "cursor-refresh-token" {
		t.Errorf("refresh token = %q", result.Token.RefreshToken)
	}
	if result.Token.TokenType != "Bearer" {
		t.Errorf("token type = %q", result.Token.TokenType)
	}
	wantExpiry := expiresAt.Add(-5 * time.Minute)
	if !result.Token.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("expires at = %v, want %v", result.Token.ExpiresAt, wantExpiry)
	}
}

func TestCursorFlowRefreshUsesOMPExchangeUserAPIKey(t *testing.T) {
	var gotAuthorization string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		gotAuthorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"accessToken":  "refreshed-access-token",
			"refreshToken": "refreshed-refresh-token",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	flow := NewCursorFlowWithConfig(CursorConfig{
		RefreshURL: server.URL + "/auth/exchange_user_api_key",
		HTTPClient: server.Client(),
	})

	token, err := flow.Refresh(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if gotAuthorization != "Bearer old-refresh-token" {
		t.Fatalf("authorization = %q", gotAuthorization)
	}
	if len(gotBody) != 0 {
		t.Fatalf("body = %+v, want empty JSON object", gotBody)
	}
	if token.Provider != ProviderID("cursor") {
		t.Errorf("provider = %q", token.Provider)
	}
	if token.AccessToken != "refreshed-access-token" {
		t.Errorf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "refreshed-refresh-token" {
		t.Errorf("refresh token = %q", token.RefreshToken)
	}
	if token.TokenType != "Bearer" {
		t.Errorf("token type = %q", token.TokenType)
	}
	if !token.ExpiresAt.After(time.Now().Add(59 * time.Minute)) {
		t.Errorf("expires at = %v, want fallback near one hour", token.ExpiresAt)
	}
}

func TestCursorFlowExchangeUnsupported(t *testing.T) {
	flow := NewCursorFlow()

	_, err := flow.Exchange(context.Background(), AuthSession{Provider: ProviderID("cursor"), SessionID: "uuid.verifier"}, "callback-code")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "poll") {
		t.Fatalf("error = %q, want poll guidance", err.Error())
	}
}

func jwtWithExpiry(expiresAt time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"exp":` + strconv.FormatInt(expiresAt.Unix(), 10) + `}`))
	return header + "." + payload + "."
}
