package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func badStatusServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad"}`, http.StatusBadRequest)
	}))
}

func okTokenServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"a","refresh_token":"r","token_type":"bearer","expires_in":3600,"scope":"x y"}`))
	}))
}

// --- shared helpers ---

func TestSplitScopes(t *testing.T) {
	if splitScopes("") != nil {
		t.Fatal("empty scope should be nil")
	}
	got := splitScopes("a  b\tc")
	if len(got) != 3 {
		t.Fatalf("scopes = %v", got)
	}
}

func TestSessionIDParsers(t *testing.T) {
	parsers := map[string]func(string) (string, string, error){
		"anthropic": parseAnthropicSessionID,
		"google":    parseGoogleSessionID,
		"cursor":    parseCursorSessionID,
		"callback":  parseCallbackSessionID,
	}
	for name, p := range parsers {
		if a, b, err := p("uuid.verifier"); err != nil || a != "uuid" || b != "verifier" {
			t.Fatalf("%s parse valid: %s/%s/%v", name, a, b, err)
		}
		for _, bad := range []string{"", "noseparator", ".verifier", "uuid."} {
			if _, _, err := p(bad); err == nil {
				t.Fatalf("%s parse %q: want error", name, bad)
			}
		}
	}
}

// --- anthropic ---

func TestAnthropicExchangeErrors(t *testing.T) {
	flow := NewAnthropicFlowWithConfig(AnthropicConfig{ClientID: "c"})
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: "other"}, "code"); err == nil {
		t.Fatal("provider mismatch: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID()}, ""); err == nil {
		t.Fatal("empty code: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "bad"}, "code"); err == nil {
		t.Fatal("bad session id: want error")
	}
}

func TestAnthropicExchangeBadStatusAndDecode(t *testing.T) {
	bad := badStatusServer()
	defer bad.Close()
	flow := NewAnthropicFlowWithConfig(AnthropicConfig{ClientID: "c", TokenURL: bad.URL, HTTPClient: bad.Client()})
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "s.v"}, "code"); err == nil {
		t.Fatal("bad status: want error")
	}

	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	defer missing.Close()
	flow2 := NewAnthropicFlowWithConfig(AnthropicConfig{ClientID: "c", TokenURL: missing.URL, HTTPClient: missing.Client()})
	if _, err := flow2.Exchange(context.Background(), AuthSession{Provider: flow2.ProviderID(), SessionID: "s.v"}, "code"); err == nil {
		t.Fatal("missing access token: want error")
	}
}

func TestAnthropicExchangeSuccess(t *testing.T) {
	server := okTokenServer()
	defer server.Close()
	flow := NewAnthropicFlowWithConfig(AnthropicConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	got, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "s.v"}, "code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if got.AccessToken != "a" || got.ExpiresAt.IsZero() || len(got.Scopes) != 2 {
		t.Fatalf("token = %+v", got)
	}
}

func TestAnthropicStartProducesSessionID(t *testing.T) {
	flow := NewAnthropicFlowWithConfig(AnthropicConfig{ClientID: "c"})
	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !strings.Contains(session.SessionID, ".") {
		t.Fatalf("session id = %q", session.SessionID)
	}
}

// --- gemini (google flow) ---

func TestGeminiExchangeErrorsAndSuccess(t *testing.T) {
	flow := NewGeminiFlow(GeminiConfig{ClientID: "c"})
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: "other"}, "code"); err == nil {
		t.Fatal("provider mismatch: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID()}, ""); err == nil {
		t.Fatal("empty code: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "bad"}, "code"); err == nil {
		t.Fatal("bad session id: want error")
	}

	server := okTokenServer()
	defer server.Close()
	flow2 := NewGeminiFlow(GeminiConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	got, err := flow2.Exchange(context.Background(), AuthSession{Provider: flow2.ProviderID(), SessionID: "s.v"}, "code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if got.AccessToken != "a" {
		t.Fatalf("token = %+v", got)
	}
}

// --- xai (callback flow) ---

func TestXAIExchangeErrorsAndSuccess(t *testing.T) {
	flow := NewXAIFlow(XAIConfig{ClientID: "c"})
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: "other"}, "code"); err == nil {
		t.Fatal("provider mismatch: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID()}, ""); err == nil {
		t.Fatal("empty code: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "bad"}, "code"); err == nil {
		t.Fatal("bad session id: want error")
	}
	server := okTokenServer()
	defer server.Close()
	flow2 := NewXAIFlow(XAIConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	if _, err := flow2.Exchange(context.Background(), AuthSession{Provider: flow2.ProviderID(), SessionID: "s.v"}, "code"); err != nil {
		t.Fatalf("Exchange: %v", err)
	}
}

// --- codex device flow ---

func TestCodexStartError(t *testing.T) {
	bad := badStatusServer()
	defer bad.Close()
	flow := NewCodexFlow(CodexFlowConfig{ClientID: "c", DeviceCodeURL: bad.URL, HTTPClient: bad.Client()})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("Start: want error")
	}
}

func TestCodexPollStatusMapping(t *testing.T) {
	cases := map[string]PollStatus{
		"authorization_pending": PollStatusPending,
		"slow_down":             PollStatusSlowDown,
		"expired_token":         PollStatusExpired,
		"access_denied":         PollStatusDenied,
		"unknown_code":          PollStatusPending,
	}
	for code, want := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(oauthError{Code: code})
		}))
		flow := NewCodexFlow(CodexFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
		res, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"})
		server.Close()
		if err != nil {
			t.Fatalf("%s poll: %v", code, err)
		}
		if res.Status != want {
			t.Fatalf("%s: status = %q, want %q", code, res.Status, want)
		}
	}
}

func TestCodexPollMissingAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"bearer"}`))
	}))
	defer server.Close()
	flow := NewCodexFlow(CodexFlowConfig{ClientID: "c", TokenURL: server.URL, HTTPClient: server.Client()})
	if _, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"}); err == nil {
		t.Fatal("missing access token: want error")
	}
}

func TestCodexPollNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	flow := NewCodexFlow(CodexFlowConfig{ClientID: "c", TokenURL: url, HTTPClient: client})
	if _, err := flow.Poll(context.Background(), AuthSession{SessionID: "d"}); err == nil {
		t.Fatal("network error: want error")
	}
}

func TestOAuthErrorString(t *testing.T) {
	if (oauthError{Code: "x", Description: "y"}).Error() != "x: y" {
		t.Fatal("with description")
	}
	if (oauthError{Code: "x"}).Error() != "x" {
		t.Fatal("without description")
	}
}

// --- kimi device flow ---

func TestKimiStartAndPollErrors(t *testing.T) {
	bad := badStatusServer()
	defer bad.Close()
	flow := NewKimiFlow(KimiFlowConfig{ClientID: "c", DeviceCodeURL: bad.URL, TokenURL: bad.URL, HTTPClient: bad.Client()})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("Start: want error")
	}
	// Poll with pending oauth error.
	pending := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(oauthError{Code: "authorization_pending"})
	}))
	defer pending.Close()
	flow2 := NewKimiFlow(KimiFlowConfig{ClientID: "c", TokenURL: pending.URL, HTTPClient: pending.Client()})
	res, err := flow2.Poll(context.Background(), AuthSession{SessionID: "d"})
	if err != nil || res.Status != PollStatusPending {
		t.Fatalf("poll pending: status=%q err=%v", res.Status, err)
	}
}

func TestKimiStartUsesVerificationURIFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"device_code":"d","user_code":"u","verification_uri":"https://v","expires_in":900,"interval":5}`))
	}))
	defer server.Close()
	flow := NewKimiFlow(KimiFlowConfig{ClientID: "c", DeviceCodeURL: server.URL, HTTPClient: server.Client()})
	session, err := flow.Start(context.Background())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if session.AuthURL != "https://v" {
		t.Fatalf("auth url = %q, want fallback", session.AuthURL)
	}
}

// --- cursor specifics ---

func TestCursorExchangeAndPollErrors(t *testing.T) {
	flow := NewCursorFlow()
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: "other"}, "c"); err == nil {
		t.Fatal("exchange provider mismatch: want error")
	}
	if _, err := flow.Exchange(context.Background(), AuthSession{Provider: flow.ProviderID()}, "c"); err == nil {
		t.Fatal("exchange unsupported: want error")
	}
	if _, err := flow.Poll(context.Background(), AuthSession{Provider: "other"}); err == nil {
		t.Fatal("poll provider mismatch: want error")
	}
	if _, err := flow.Poll(context.Background(), AuthSession{Provider: flow.ProviderID(), SessionID: "bad"}); err == nil {
		t.Fatal("poll bad session: want error")
	}
}

func TestCursorRefreshErrors(t *testing.T) {
	flow := NewCursorFlow()
	if _, err := flow.Refresh(context.Background(), "  "); err == nil {
		t.Fatal("empty refresh: want error")
	}
	bad := badStatusServer()
	defer bad.Close()
	flow2 := NewCursorFlowWithConfig(CursorConfig{RefreshURL: bad.URL, HTTPClient: bad.Client()})
	if _, err := flow2.Refresh(context.Background(), "tok"); err == nil {
		t.Fatal("bad status refresh: want error")
	}
}

func TestCursorRefreshSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accessToken":"acc","refreshToken":"ref"}`))
	}))
	defer server.Close()
	flow := NewCursorFlowWithConfig(CursorConfig{RefreshURL: server.URL, HTTPClient: server.Client()})
	got, err := flow.Refresh(context.Background(), "tok")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got.AccessToken != "acc" || got.RefreshToken != "ref" {
		t.Fatalf("token = %+v", got)
	}
}

func TestCursorStartUUIDError(t *testing.T) {
	flow := NewCursorFlowWithConfig(CursorConfig{NewUUID: func() (string, error) { return "  ", nil }})
	if _, err := flow.Start(context.Background()); err == nil {
		t.Fatal("empty uuid: want error")
	}
}

func TestNewCursorUUIDProducesUUID(t *testing.T) {
	id, err := newCursorUUID()
	if err != nil {
		t.Fatalf("newCursorUUID: %v", err)
	}
	if len(id) != 36 || strings.Count(id, "-") != 4 {
		t.Fatalf("uuid = %q", id)
	}
}

func TestCursorTokenExpiryFromJWT(t *testing.T) {
	exp := time.Now().Add(2 * time.Hour).Unix()
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"exp":` + itoa(exp) + `}`))
	jwt := "header." + payload + ".sig"
	got := cursorTokenExpiry(jwt, time.Now())
	// Should be exp minus 5 minutes, roughly two hours out.
	if got.Before(time.Now().Add(time.Hour)) {
		t.Fatalf("expiry = %v, want ~2h out", got)
	}

	// Non-JWT token falls back to now+1h.
	fallback := cursorTokenExpiry("notajwt", time.Now())
	if fallback.Before(time.Now().Add(50 * time.Minute)) {
		t.Fatalf("fallback expiry = %v", fallback)
	}
	// Malformed base64 payload falls back.
	bad := cursorTokenExpiry("a.!!!.c", time.Now())
	if bad.Before(time.Now().Add(50 * time.Minute)) {
		t.Fatalf("bad payload expiry = %v", bad)
	}
}

func itoa(v int64) string {
	return strings.TrimSpace(jsonNumber(v))
}

func jsonNumber(v int64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
