package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func makeTestIDToken(claims map[string]any) string {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(claims)
	h := base64.RawURLEncoding.EncodeToString(hb)
	p := base64.RawURLEncoding.EncodeToString(pb)
	mac := hmac.New(sha256.New, []byte("test-key"))
	mac.Write([]byte(h + "." + p))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return h + "." + p + "." + sig
}

func TestDiscoveryFetch(t *testing.T) {
	doc := map[string]any{
		"issuer":                 "https://idp.example.com",
		"authorization_endpoint": "https://idp.example.com/authorize",
		"token_endpoint":         "https://idp.example.com/token",
		"jwks_uri":               "https://idp.example.com/jwks",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Errorf("discovery path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(doc)
	}))
	defer srv.Close()

	discovery, err := FetchOIDCDiscovery(srv.URL, nil)
	if err != nil {
		t.Fatalf("FetchOIDCDiscovery: %v", err)
	}
	if discovery.AuthorizationEndpoint != doc["authorization_endpoint"] {
		t.Errorf("AuthorizationEndpoint = %q", discovery.AuthorizationEndpoint)
	}
	if discovery.TokenEndpoint != doc["token_endpoint"] {
		t.Errorf("TokenEndpoint = %q", discovery.TokenEndpoint)
	}
	if discovery.JWKSURI != doc["jwks_uri"] {
		t.Errorf("JWKSURI = %q", discovery.JWKSURI)
	}
}

func TestAuthURLContainsPKCEAndState(t *testing.T) {
	pair, err := CreateOIDCPKCEPair()
	if err != nil {
		t.Fatalf("CreateOIDCPKCEPair: %v", err)
	}
	state, err := CreateOIDCState()
	if err != nil {
		t.Fatalf("CreateOIDCState: %v", err)
	}
	nonce, err := CreateOIDCNonce()
	if err != nil {
		t.Fatalf("CreateOIDCNonce: %v", err)
	}

	authURL := BuildOIDCAuthorizationURL(OIDCAuthURLParams{
		AuthorizationEndpoint: "https://idp.example.com/authorize",
		ClientID:              "client-id",
		RedirectURI:           "https://app.example.com/api/auth/oidc/callback",
		Scopes:                "openid profile email",
		State:                 state,
		Nonce:                 nonce,
		CodeChallenge:         pair.Challenge,
	})

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}
	q := parsed.Query()
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("client_id") != "client-id" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://app.example.com/api/auth/oidc/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("scope") != "openid profile email" {
		t.Errorf("scope = %q", q.Get("scope"))
	}
	if q.Get("state") != state {
		t.Errorf("state = %q, want %q", q.Get("state"), state)
	}
	if q.Get("nonce") != nonce {
		t.Errorf("nonce = %q, want %q", q.Get("nonce"), nonce)
	}
	if q.Get("code_challenge") != pair.Challenge {
		t.Errorf("code_challenge = %q, want %q", q.Get("code_challenge"), pair.Challenge)
	}
	if q.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q", q.Get("code_challenge_method"))
	}

	// Challenge must be S256(verifier).
	if pair.Challenge != pkceChallenge(pair.Verifier) {
		t.Error("challenge is not S256(verifier)")
	}
}

func TestExchangeSendsVerifierAndSecret(t *testing.T) {
	var lastForm url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		lastForm = r.PostForm
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at-1",
			"id_token":     makeTestIDToken(map[string]any{"nonce": "n1"}),
		})
	}))
	defer srv.Close()

	data, err := ExchangeOIDCCode(OIDCCodeExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		Code:          "code-1",
		RedirectURI:   "https://app.example.com/cb",
		CodeVerifier:  "verifier-1",
	}, nil)
	if err != nil {
		t.Fatalf("ExchangeOIDCCode: %v", err)
	}
	if lastForm.Get("grant_type") != "authorization_code" {
		t.Errorf("grant_type = %q", lastForm.Get("grant_type"))
	}
	if lastForm.Get("client_id") != "client-id" {
		t.Errorf("client_id = %q", lastForm.Get("client_id"))
	}
	if lastForm.Get("client_secret") != "client-secret" {
		t.Errorf("client_secret = %q", lastForm.Get("client_secret"))
	}
	if lastForm.Get("code") != "code-1" {
		t.Errorf("code = %q", lastForm.Get("code"))
	}
	if lastForm.Get("redirect_uri") != "https://app.example.com/cb" {
		t.Errorf("redirect_uri = %q", lastForm.Get("redirect_uri"))
	}
	if lastForm.Get("code_verifier") != "verifier-1" {
		t.Errorf("code_verifier = %q", lastForm.Get("code_verifier"))
	}
	if data["access_token"] != "at-1" {
		t.Errorf("access_token = %v", data["access_token"])
	}

	// Without a client secret the field must not be sent.
	lastForm = nil
	_, err = ExchangeOIDCCode(OIDCCodeExchangeParams{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
		Code:          "code-2",
		RedirectURI:   "https://app.example.com/cb",
		CodeVerifier:  "verifier-2",
	}, nil)
	if err != nil {
		t.Fatalf("ExchangeOIDCCode without secret: %v", err)
	}
	if lastForm.Get("client_secret") != "" {
		t.Errorf("client_secret sent when empty: %q", lastForm.Get("client_secret"))
	}
}

func TestNonceMismatchRejected(t *testing.T) {
	tok := makeTestIDToken(map[string]any{"nonce": "expected-nonce"})
	if err := VerifyOIDCNonce(tok, "expected-nonce"); err != nil {
		t.Fatalf("valid nonce rejected: %v", err)
	}
	if err := VerifyOIDCNonce(tok, "wrong-nonce"); err == nil {
		t.Fatal("wrong nonce accepted")
	}
}

func TestStateMismatchRejected(t *testing.T) {
	if err := ValidateOIDCState("stored-state", "stored-state"); err != nil {
		t.Fatalf("matching state rejected: %v", err)
	}
	if err := ValidateOIDCState("stored-state", "returned-state"); err == nil {
		t.Fatal("mismatched state accepted")
	}
	if err := ValidateOIDCState("stored-state", ""); err == nil {
		t.Fatal("empty returned state accepted")
	}
}

func TestProbeNoSecretSkips(t *testing.T) {
	res, err := ProbeOIDCClientSecret("https://idp.example.com/token", "client-id", "", "https://app.example.com/cb", nil)
	if err != nil {
		t.Fatalf("ProbeOIDCClientSecret: %v", err)
	}
	if res.Tested {
		t.Errorf("tested = %v, want false", res.Tested)
	}
	if res.Valid != nil {
		t.Errorf("valid = %v, want nil", res.Valid)
	}
	wantMsg := "No client secret was provided, so secret validation was skipped."
	if res.Message != wantMsg {
		t.Errorf("message = %q, want %q", res.Message, wantMsg)
	}
}

func TestProbeInvalidCodeClassification(t *testing.T) {
	cases := []struct {
		name      string
		errorCode string
		wantValid bool
	}{
		{"invalid_client rejects secret", "invalid_client", false},
		{"unauthorized_client rejects secret", "unauthorized_client", false},
		{"invalid_grant accepts secret", "invalid_grant", true},
		{"invalid_code accepts secret", "invalid_code", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.ParseForm()
				if r.PostForm.Get("code") != "__oidc_test_invalid_code__" {
					t.Errorf("code = %q", r.PostForm.Get("code"))
				}
				if r.PostForm.Get("code_verifier") != "__oidc_test_invalid_verifier__" {
					t.Errorf("code_verifier = %q", r.PostForm.Get("code_verifier"))
				}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": tc.errorCode})
			}))
			defer srv.Close()

			res, err := ProbeOIDCClientSecret(srv.URL, "client-id", "client-secret", "https://app.example.com/cb", nil)
			if err != nil {
				t.Fatalf("ProbeOIDCClientSecret: %v", err)
			}
			if !res.Tested {
				t.Fatalf("tested = %v, want true", res.Tested)
			}
			if res.Valid == nil {
				t.Fatal("valid is nil")
			}
			if *res.Valid != tc.wantValid {
				t.Errorf("valid = %v, want %v", *res.Valid, tc.wantValid)
			}
		})
	}
}

func TestProbeUnexpectedErrorUnknown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "server_error", "error_description": "something else"})
	}))
	defer srv.Close()

	res, err := ProbeOIDCClientSecret(srv.URL, "client-id", "client-secret", "https://app.example.com/cb", nil)
	if err != nil {
		t.Fatalf("ProbeOIDCClientSecret: %v", err)
	}
	if !res.Tested {
		t.Fatal("expected tested=true")
	}
	if res.Valid != nil {
		t.Errorf("valid = %v, want nil", res.Valid)
	}
}

func TestProbeOKAcceptsSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at-probe",
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	res, err := ProbeOIDCClientSecret(srv.URL, "client-id", "client-secret", "https://app.example.com/cb", nil)
	if err != nil {
		t.Fatalf("ProbeOIDCClientSecret: %v", err)
	}
	if !res.Tested || res.Valid == nil || !*res.Valid {
		t.Errorf("result = %+v", res)
	}
	if !strings.Contains(res.Message, "accepted") {
		t.Errorf("message = %q", res.Message)
	}
}
