package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestOAuthStartBuildsPKCEAuthorizationURL(t *testing.T) {
	flow, err := BuildOAuthStartFlow(OAuthStartConfig{
		InstanceID:        "inst-1",
		AuthorizationURL:  "https://auth.example/authorize",
		ResourceURI:       "https://mcp.example",
		RedirectURI:       "http://localhost:3000/api/mcp/oauth/callback",
		ExpirationSeconds: 600,
	})
	if err != nil {
		t.Fatalf("BuildOAuthStartFlow: %v", err)
	}
	if flow.State == "" || flow.CodeVerifierSecret == "" {
		t.Fatalf("flow = %+v, want state and verifier", flow)
	}
	if flow.State == flow.CodeVerifierSecret {
		t.Fatal("state and PKCE verifier must be separate values")
	}

	parsed, err := url.Parse(flow.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("state") != flow.State {
		t.Fatalf("state query = %q, want flow state", query.Get("state"))
	}
	if query.Get("resource") != "https://mcp.example" {
		t.Fatalf("resource query = %q, want resource", query.Get("resource"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("challenge method = %q, want S256", query.Get("code_challenge_method"))
	}
	if query.Get("code_challenge") != pkceChallenge(flow.CodeVerifierSecret) {
		t.Fatalf("challenge = %q, want verifier challenge", query.Get("code_challenge"))
	}
	redirect, err := url.Parse(query.Get("redirect_uri"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if decodedInstanceIDForOAuthTest(t, redirect.Query().Get("instance_id")) != "inst-1" {
		t.Fatalf("redirect instance_id = %q, want recoverable inst-1", redirect.Query().Get("instance_id"))
	}
}

func TestOAuthStartIncludesClientID(t *testing.T) {
	flow, err := BuildOAuthStartFlow(OAuthStartConfig{
		InstanceID:        "inst-1",
		AuthorizationURL:  "https://auth.example/authorize",
		ResourceURI:       "https://mcp.example",
		RedirectURI:       "http://localhost:3000/api/mcp/oauth/callback",
		ClientID:          "client-123",
		ExpirationSeconds: 600,
	})
	if err != nil {
		t.Fatalf("BuildOAuthStartFlow: %v", err)
	}

	parsed, err := url.Parse(flow.AuthorizationURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if parsed.Query().Get("client_id") != "client-123" {
		t.Fatalf("client_id query = %q, want client-123", parsed.Query().Get("client_id"))
	}
	if flow.ClientID != "client-123" {
		t.Fatalf("flow client id = %q, want client-123", flow.ClientID)
	}
}

func TestOAuthEngineCompletesCallbackForMatchingInstance(t *testing.T) {
	store := newFakeOAuthStore()
	var tokenForm url.Values
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Fatalf("token path = %q, want /token", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		tokenForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"expires_in":    3600,
			"scope":         "read write",
			"account_label": "work",
			"sub":           "subject-1",
			"email":         "work@example.com",
			"iss":           "https://auth.example",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	flow := OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost:3000/api/mcp/oauth/callback?instance_id=inst-1",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}
	flow.AuthorizationURL = tokenServer.URL + "/authorize"
	if err := store.CreateFlow(flow); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	account, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if account.InstanceID != "inst-1" || account.AccountLabel != "work" || account.AccessToken != "access-token" || account.RefreshToken != "refresh-token" {
		t.Fatalf("account = %+v, want exchanged token for inst-1/work", account)
	}
	if tokenForm.Get("grant_type") != "authorization_code" ||
		tokenForm.Get("code") != "ok" ||
		tokenForm.Get("code_verifier") != "verifier" ||
		tokenForm.Get("redirect_uri") != flow.RedirectURI ||
		tokenForm.Get("resource") != flow.ResourceURI {
		t.Fatalf("token form = %+v, want auth-code exchange with verifier and resource", tokenForm)
	}
	if store.accounts[0].AccessToken != "access-token" || store.accounts[0].AuthMetadata["token_endpoint"] != tokenServer.URL+"/token" {
		t.Fatalf("stored account = %+v, want real token and token endpoint metadata", store.accounts[0])
	}
	if _, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1"); err == nil {
		t.Fatal("state should be single-use")
	}
}

func TestOAuthEnginePostsClientCredentialsWhenFlowProvidesThem(t *testing.T) {
	store := newFakeOAuthStore()
	var tokenForm url.Values
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		tokenForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token",
			"expires_in":   3600,
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	if err := store.CreateFlow(OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost:3000/api/mcp/oauth/callback?instance_id=inst-1",
		AuthorizationURL:   tokenServer.URL + "/authorize",
		ResourceURI:        "https://mcp.example",
		ClientID:           "client-123",
		ClientSecret:       "client-secret",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	_, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if tokenForm.Get("client_id") != "client-123" {
		t.Fatalf("client_id form = %q, want client-123", tokenForm.Get("client_id"))
	}
	if tokenForm.Get("client_secret") != "client-secret" {
		t.Fatal("client_secret form was not posted from consumed flow")
	}
}

func TestOAuthEngineRejectsMismatchedInstanceState(t *testing.T) {
	store := newFakeOAuthStore()
	engine := NewOAuthEngine(store, OAuthHTTPClient(nil))
	if err := store.CreateFlow(OAuthFlow{InstanceID: "inst-1", State: "state-1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	_, err := engine.CompleteCallback(context.Background(), "inst-2", "https://callback.example?code=ok&state=state-1")
	if err == nil {
		t.Fatal("mismatched instance should fail")
	}
}

func TestOAuthEngineUsesSelectedInstanceAccountLabel(t *testing.T) {
	store := newFakeOAuthStore()
	store.accountLabels["inst-1"] = "selected-work"
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-token",
			"expires_in":   3600,
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	if err := store.CreateFlow(OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost/callback?instance_id=inst-1",
		AuthorizationURL:   tokenServer.URL + "/authorize",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	account, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if account.AccountLabel != "selected-work" {
		t.Fatalf("account label = %q, want selected-work", account.AccountLabel)
	}
}

func TestOAuthEnginePrefersSelectedInstanceAccountLabelOverTokenAccountLabel(t *testing.T) {
	store := newFakeOAuthStore()
	store.accountLabels["inst-1"] = "selected-work"
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access-token",
			"expires_in":    3600,
			"account_label": "token-work",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	if err := store.CreateFlow(OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost/callback?instance_id=inst-1",
		AuthorizationURL:   tokenServer.URL + "/authorize",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	account, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if account.AccountLabel != "selected-work" {
		t.Fatalf("account label = %q, want selected-work", account.AccountLabel)
	}
	if store.accounts[0].AccountLabel != "selected-work" {
		t.Fatalf("stored account label = %q, want selected-work", store.accounts[0].AccountLabel)
	}
}

func TestOAuthEngineRequiresRealTokenEndpoint(t *testing.T) {
	store := newFakeOAuthStore()
	authServer := httptest.NewServer(http.NotFoundHandler())
	defer authServer.Close()

	engine := NewOAuthEngine(store, authServer.Client())
	if err := store.CreateFlow(OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost/callback?instance_id=inst-1",
		AuthorizationURL:   authServer.URL + "/login",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	_, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")

	if !errors.Is(err, errOAuthTokenEndpointUnavailable) {
		t.Fatalf("err = %v, want token endpoint unavailable", err)
	}
	if len(store.accounts) != 0 {
		t.Fatalf("stored accounts = %+v, want none", store.accounts)
	}
}

func TestOAuthEngineDiscoversTokenEndpointFromAuthorizationServerMetadata(t *testing.T) {
	store := newFakeOAuthStore()
	var tokenForm url.Values
	var tokenEndpoint string
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{"token_endpoint": tokenEndpoint}); err != nil {
				t.Fatalf("Encode metadata: %v", err)
			}
		case "/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			tokenForm = r.PostForm
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{
				"access_token": "metadata-access-token",
				"expires_in":   3600,
			}); err != nil {
				t.Fatalf("Encode token: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer authServer.Close()
	tokenEndpoint = authServer.URL + "/oauth/token"

	engine := NewOAuthEngine(store, authServer.Client())
	if err := store.CreateFlow(OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier-from-store",
		RedirectURI:        "http://localhost/callback?instance_id=inst-1",
		AuthorizationURL:   authServer.URL + "/login",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	account, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")
	if err != nil {
		t.Fatalf("CompleteCallback: %v", err)
	}
	if account.AccessToken != "metadata-access-token" || account.AuthMetadata["token_endpoint"] != tokenEndpoint {
		t.Fatalf("account = %+v, want metadata token endpoint and access token", account)
	}
	if tokenForm.Get("code_verifier") != "verifier-from-store" || tokenForm.Get("resource") != "https://mcp.example" {
		t.Fatalf("token form = %+v, want stored verifier and resource", tokenForm)
	}
}

func TestDiscoverOAuthAuthorizationURLFromProtectedResourceMetadata(t *testing.T) {
	var authorizationEndpoint string
	resourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp", resource_metadata="`+resourceServerURLForOAuthTest(r)+`/oauth-resource"`)
			w.WriteHeader(http.StatusUnauthorized)
		case "/oauth-resource":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"authorization_servers": []string{resourceServerURLForOAuthTest(r)}}); err != nil {
				t.Fatalf("Encode protected resource metadata: %v", err)
			}
		case "/.well-known/oauth-authorization-server":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]string{"authorization_endpoint": authorizationEndpoint}); err != nil {
				t.Fatalf("Encode authorization metadata: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer resourceServer.Close()
	authorizationEndpoint = resourceServer.URL + "/oauth/authorize"

	got, err := DiscoverOAuthAuthorizationURL(context.Background(), resourceServer.Client(), resourceServer.URL+"/mcp")

	if err != nil {
		t.Fatalf("DiscoverOAuthAuthorizationURL: %v", err)
	}
	if got != authorizationEndpoint {
		t.Fatalf("authorization URL = %q, want %q", got, authorizationEndpoint)
	}
}

func TestOAuthEngineRejectsRedirectingTokenEndpointWithoutFollowing(t *testing.T) {
	store := newFakeOAuthStore()
	var redirectHit bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			http.Redirect(w, r, "/redirect-token", http.StatusTemporaryRedirect)
		case "/redirect-token":
			redirectHit = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "redirected-token",
				"expires_in":   3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	if err := store.CreateFlow(OAuthFlow{
		InstanceID:         "inst-1",
		State:              "state-1",
		CodeVerifierSecret: "verifier",
		RedirectURI:        "http://localhost/callback?instance_id=inst-1",
		AuthorizationURL:   tokenServer.URL + "/authorize",
		ResourceURI:        "https://mcp.example",
		ExpiresAt:          time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("CreateFlow: %v", err)
	}

	_, err := engine.CompleteCallback(context.Background(), "inst-1", "https://callback.example?code=ok&state=state-1")

	if !errors.Is(err, errOAuthTokenEndpointUnavailable) {
		t.Fatalf("err = %v, want token endpoint unavailable", err)
	}
	if redirectHit {
		t.Fatal("token redirect target was followed")
	}
	if len(store.accounts) != 0 {
		t.Fatalf("stored accounts = %+v, want none", store.accounts)
	}
}

func TestOAuthEngineAddsBearerAndProtocolHeaders(t *testing.T) {
	var gotAuth string
	var gotProtocol string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotProtocol = r.Header.Get("MCP-Protocol-Version")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	engine := NewOAuthEngine(newFakeOAuthStore(), server.Client())
	err := engine.AuthorizeRequest(&http.Request{Header: http.Header{}}, OAuthAccount{
		InstanceID:  "inst-1",
		AccessToken: "token-1",
		ExpiresAt:   time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("AuthorizeRequest: %v", err)
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	if err := engine.AuthorizeRequest(req, OAuthAccount{AccessToken: "token-1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("AuthorizeRequest req: %v", err)
	}
	if _, err := server.Client().Do(req); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotAuth != "Bearer token-1" {
		t.Fatalf("Authorization = %q, want bearer", gotAuth)
	}
	if gotProtocol == "" {
		t.Fatal("MCP protocol header is empty")
	}
}

func TestOAuthEngineRequiresReauthForExpiredOrWrongResource(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), OAuthHTTPClient(nil))
	req, _ := http.NewRequest(http.MethodGet, "https://mcp.example", nil)

	err := engine.AuthorizeRequest(req, OAuthAccount{AccessToken: "token", ExpiresAt: time.Now().Add(-time.Minute)})
	if err != ErrReauthRequired {
		t.Fatalf("expired err = %v, want ErrReauthRequired", err)
	}
	err = engine.AuthorizeRequest(req, OAuthAccount{AccessToken: "token", ResourceURI: "https://other.example", ExpiresAt: time.Now().Add(time.Hour)})
	if err != ErrReauthRequired {
		t.Fatalf("wrong resource err = %v, want ErrReauthRequired", err)
	}
}

func TestOAuthEngineRefreshRejectsRedirectingTokenEndpointWithoutFollowing(t *testing.T) {
	store := newFakeOAuthStore()
	var redirectHit bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			http.Redirect(w, r, "/redirect-token", http.StatusTemporaryRedirect)
		case "/redirect-token":
			redirectHit = true
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "redirected-refresh-token",
				"expires_in":   3600,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	_, err := engine.RefreshAccount(context.Background(), OAuthAccount{
		InstanceID:   "inst-1",
		AccountLabel: "work",
		ResourceURI:  "https://mcp.example",
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
		AuthMetadata: map[string]string{"token_endpoint": tokenServer.URL + "/token"},
	})

	if !errors.Is(err, errOAuthTokenEndpointUnavailable) {
		t.Fatalf("err = %v, want token endpoint unavailable", err)
	}
	if redirectHit {
		t.Fatal("refresh token redirect target was followed")
	}
	if len(store.accounts) != 0 {
		t.Fatalf("stored accounts = %+v, want no refresh persistence", store.accounts)
	}
}

func TestOAuthEngineRefreshesSameInstanceAccount(t *testing.T) {
	store := newFakeOAuthStore()
	var tokenForm url.Values
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		tokenForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"expires_in":   1200,
			"scope":        "read",
		}); err != nil {
			t.Fatalf("Encode: %v", err)
		}
	}))
	defer tokenServer.Close()

	engine := NewOAuthEngine(store, tokenServer.Client())
	refreshed, err := engine.RefreshAccount(context.Background(), OAuthAccount{
		InstanceID:   "inst-1",
		AccountLabel: "work",
		ResourceURI:  "https://mcp.example",
		Scopes:       []string{"old"},
		AccessToken:  "old-access",
		RefreshToken: "old-refresh",
		ExpiresAt:    time.Now().Add(-time.Minute),
		AuthMetadata: map[string]string{"token_endpoint": tokenServer.URL},
	})
	if err != nil {
		t.Fatalf("RefreshAccount: %v", err)
	}
	if refreshed.InstanceID != "inst-1" || refreshed.AccountLabel != "work" || refreshed.AccessToken != "new-access" || refreshed.RefreshToken != "old-refresh" {
		t.Fatalf("refreshed = %+v, want same account with new access and retained refresh", refreshed)
	}
	if tokenForm.Get("grant_type") != "refresh_token" || tokenForm.Get("refresh_token") != "old-refresh" || tokenForm.Get("resource") != "https://mcp.example" {
		t.Fatalf("token form = %+v, want refresh grant with resource", tokenForm)
	}
	if len(store.accounts) != 1 || store.accounts[0].InstanceID != "inst-1" || store.accounts[0].AccountLabel != "work" {
		t.Fatalf("stored accounts = %+v, want one updated same account", store.accounts)
	}
}

func TestStdioCredentialsReturnRedactedEnv(t *testing.T) {
	env := StdioCredentialEnv(OAuthAccount{AccessToken: "token", RefreshToken: "refresh"})

	if env.Actual["MCP_ACCESS_TOKEN"] != "token" {
		t.Fatalf("actual token = %q, want token", env.Actual["MCP_ACCESS_TOKEN"])
	}
	if env.Redacted["MCP_ACCESS_TOKEN"] != RedactedValue {
		t.Fatalf("redacted token = %q, want redacted", env.Redacted["MCP_ACCESS_TOKEN"])
	}
	if env.Redacted["MCP_REFRESH_TOKEN"] != RedactedValue {
		t.Fatalf("redacted refresh = %q, want redacted", env.Redacted["MCP_REFRESH_TOKEN"])
	}
}

type fakeOAuthStore struct {
	flows         map[string]OAuthFlow
	accounts      []OAuthAccount
	accountLabels map[string]string
}

func newFakeOAuthStore() *fakeOAuthStore {
	return &fakeOAuthStore{flows: make(map[string]OAuthFlow), accountLabels: make(map[string]string)}
}

func (s *fakeOAuthStore) CreateFlow(flow OAuthFlow) error {
	s.flows[flow.InstanceID+"|"+flow.State] = flow
	return nil
}

func (s *fakeOAuthStore) ConsumeFlow(instanceID, state string) (OAuthFlow, error) {
	key := instanceID + "|" + state
	flow, ok := s.flows[key]
	if !ok {
		return OAuthFlow{}, ErrOAuthFlowNotFound
	}
	delete(s.flows, key)
	return flow, nil
}

func (s *fakeOAuthStore) SaveAccount(account OAuthAccount) error {
	s.accounts = append(s.accounts, account)
	return nil
}

func (s *fakeOAuthStore) AccountLabelForInstance(instanceID string) (string, error) {
	return s.accountLabels[instanceID], nil
}

func decodedInstanceIDForOAuthTest(t *testing.T, value string) string {
	t.Helper()
	if strings.HasPrefix(value, "b64:") {
		decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, "b64:"))
		if err != nil {
			t.Fatalf("decode instance id: %v", err)
		}
		return string(decoded)
	}
	return value
}

func resourceServerURLForOAuthTest(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
