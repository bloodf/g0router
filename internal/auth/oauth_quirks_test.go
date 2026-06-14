package auth

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestAnthropicAuthorizeURLByteIdenticalWithEmptyExtraParams is the additive-only
// regression guard: the anthropic authorize URL must be byte-identical whether the
// config carries a nil ExtraAuthParams (the existing behavior).
func TestAnthropicAuthorizeURLByteIdenticalWithEmptyExtraParams(t *testing.T) {
	prev := randRead
	t.Cleanup(func() { randRead = prev })
	// Deterministic randomness so the two URLs are comparable.
	randRead = func(b []byte) (int, error) {
		for i := range b {
			b[i] = 0x41
		}
		return len(b), nil
	}

	st := newTestStore(t)
	cfg := AnthropicOAuth()
	if cfg.ExtraAuthParams != nil {
		t.Fatal("AnthropicOAuth must not set ExtraAuthParams")
	}
	flow := NewOAuthFlow(cfg, st, nil)
	authURL, _, err := flow.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if strings.Contains(authURL, "access_type") || strings.Contains(authURL, "prompt=consent") {
		t.Fatalf("anthropic authorize URL leaked extra params: %s", authURL)
	}
}

func TestStartWithRedirectAppendsExtraAuthParams(t *testing.T) {
	st := newTestStore(t)
	flow := NewOAuthFlow(OAuthConfig{
		Provider:     "codex",
		ClientID:     "c",
		AuthorizeURL: "https://example.com/authorize",
		TokenURL:     "https://example.com/token",
		RedirectURI:  "http://localhost/cb",
		ExtraAuthParams: map[string]string{
			"originator": "codex_cli_rs",
			"prompt":     "consent",
		},
	}, st, nil)

	authURL, _, err := flow.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Query().Get("originator") != "codex_cli_rs" {
		t.Errorf("originator = %q", parsed.Query().Get("originator"))
	}
	if parsed.Query().Get("prompt") != "consent" {
		t.Errorf("prompt = %q", parsed.Query().Get("prompt"))
	}
	// Existing PKCE params must still be present.
	if parsed.Query().Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q", parsed.Query().Get("code_challenge_method"))
	}
}

func TestRefreshBasicAuthMode(t *testing.T) {
	st := newTestStore(t)
	var gotAuth string
	var gotForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		r.ParseForm()
		gotForm = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-new", "expires_in": 3600})
	}))
	defer srv.Close()

	flow := NewOAuthFlow(OAuthConfig{
		Provider:     "iflow",
		ClientID:     "id",
		ClientSecret: "secret",
		TokenURL:     srv.URL,
		RefreshMode:  "basic",
	}, st, srv.Client())

	tok, err := flow.Refresh("rt-1")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if tok.AccessToken != "at-new" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("id:secret"))
	if gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
	if gotForm.Get("refresh_token") != "rt-1" {
		t.Errorf("refresh_token = %q", gotForm.Get("refresh_token"))
	}
}

func TestRefreshJSONMode(t *testing.T) {
	st := newTestStore(t)
	var gotPath string
	var gotBody map[string]any
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"accessToken": "cl-at", "refreshToken": "cl-rt"},
		})
	}))
	defer srv.Close()

	flow := NewOAuthFlow(OAuthConfig{
		Provider:    "cline",
		TokenURL:    srv.URL + "/token",
		RefreshURL:  srv.URL + "/refresh",
		RefreshMode: "json",
		ExtraAuthParams: map[string]string{
			"client_type": "extension",
		},
	}, st, srv.Client())

	tok, err := flow.Refresh("old-rt")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if gotPath != "/refresh" {
		t.Errorf("path = %q, want /refresh (RefreshURL)", gotPath)
	}
	if !strings.HasPrefix(gotCT, "application/json") {
		t.Errorf("Content-Type = %q", gotCT)
	}
	if gotBody["refreshToken"] != "old-rt" {
		t.Errorf("refreshToken = %v", gotBody["refreshToken"])
	}
	if gotBody["grantType"] != "refresh_token" {
		t.Errorf("grantType = %v", gotBody["grantType"])
	}
	if gotBody["clientType"] != "extension" {
		t.Errorf("clientType = %v", gotBody["clientType"])
	}
	if tok.AccessToken != "cl-at" {
		t.Errorf("AccessToken = %q (want nested data.accessToken)", tok.AccessToken)
	}
	if tok.RefreshToken != "cl-rt" {
		t.Errorf("RefreshToken = %q", tok.RefreshToken)
	}
}

func TestRefreshNoneModeReturnsSentinel(t *testing.T) {
	st := newTestStore(t)
	flow := NewOAuthFlow(OAuthConfig{
		Provider:    "kilocode",
		TokenURL:    "https://example.com/should-not-be-called",
		RefreshMode: "none",
	}, st, nil)

	if _, err := flow.Refresh("anything"); err == nil {
		t.Fatal("Refresh with RefreshMode=none returned nil error, want sentinel")
	}
}

func TestExchangeBase64JSONCodeEncoding(t *testing.T) {
	st := newTestStore(t)
	// The token endpoint must NOT be hit for the base64-json happy path.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("token endpoint hit for base64-json code: %s", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	flow := NewOAuthFlow(OAuthConfig{
		Provider:     "cline",
		AuthorizeURL: "https://example.com/authorize",
		TokenURL:     srv.URL,
		RedirectURI:  "http://localhost/cb",
		CodeEncoding: "base64-json",
	}, st, srv.Client())

	_, state, err := flow.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	tokenData := map[string]any{
		"accessToken":  "cline-access",
		"refreshToken": "cline-refresh",
		"expiresAt":    "2099-01-01T00:00:00Z",
	}
	raw, _ := json.Marshal(tokenData)
	// Trailing junk after the JSON object, mirroring the ref's lastIndexOf('}') trim.
	code := base64.StdEncoding.EncodeToString(append(raw, []byte("\n#sig")...))

	tok, err := flow.Exchange(state, code)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if tok.AccessToken != "cline-access" {
		t.Errorf("AccessToken = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "cline-refresh" {
		t.Errorf("RefreshToken = %q", tok.RefreshToken)
	}
	if tok.ExpiresAt == 0 {
		t.Error("ExpiresAt = 0, want parsed from expiresAt ISO")
	}
}
