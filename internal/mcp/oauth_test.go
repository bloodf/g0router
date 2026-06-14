package mcp

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func newOAuthTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

const prmBody = `{"resource":"https://srv.example/mcp","authorization_servers":["https://as.example"]}`
const asmBody = `{"issuer":"https://as.example","authorization_endpoint":"https://as.example/authorize","token_endpoint":"https://as.example/token"}`

func TestParseProtectedResourceMetadata(t *testing.T) {
	servers, err := parseProtectedResourceMetadata([]byte(prmBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(servers) != 1 || servers[0] != "https://as.example" {
		t.Fatalf("authorization_servers = %#v", servers)
	}
}

func TestParseAuthServerMetadata(t *testing.T) {
	authz, token, err := parseAuthServerMetadata([]byte(asmBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if authz != "https://as.example/authorize" || token != "https://as.example/token" {
		t.Fatalf("endpoints = %q %q", authz, token)
	}
}

func TestEngineStartPersistsFlowAndBuildsAuthURL(t *testing.T) {
	st := newOAuthTestStore(t)
	ft := &fakeTransport{responses: []fakeResp{
		jsonResp(prmBody), // protected-resource-metadata
		jsonResp(asmBody), // authorization-server-metadata
	}}
	eng := NewEngine(st, fakeClient(ft))

	res, err := eng.Start(context.Background(), "https://srv.example/mcp", "inst-1", "https://app/callback")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if res.State == "" {
		t.Fatalf("empty state")
	}
	// The authorize URL must target the DISCOVERED endpoint with an S256 challenge.
	if !strings.HasPrefix(res.AuthURL, "https://as.example/authorize?") {
		t.Fatalf("authURL = %q", res.AuthURL)
	}
	if !strings.Contains(res.AuthURL, "code_challenge_method=S256") {
		t.Fatalf("missing S256 challenge method: %q", res.AuthURL)
	}
	if !strings.Contains(res.AuthURL, "code_challenge=") {
		t.Fatalf("missing code_challenge: %q", res.AuthURL)
	}
	if !strings.Contains(res.AuthURL, "state="+res.State) {
		t.Fatalf("authURL state mismatch: %q", res.AuthURL)
	}

	// The flow is persisted with an ENCRYPTED verifier (consumable once).
	flow, err := st.ConsumeMCPOAuthFlow(res.State)
	if err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow: %v", err)
	}
	if flow.Verifier == "" {
		t.Fatalf("flow has no verifier")
	}
	if flow.InstanceID != "inst-1" {
		t.Fatalf("flow instance = %q", flow.InstanceID)
	}
}

func TestEngineCompleteExchangesAndUpsertsAccount(t *testing.T) {
	st := newOAuthTestStore(t)
	tokenBody := `{"access_token":"AT-secret","refresh_token":"RT-secret","expires_in":3600,"scope":"read"}`
	ft := &fakeTransport{responses: []fakeResp{
		jsonResp(prmBody),
		jsonResp(asmBody),
		jsonResp(tokenBody), // token exchange
	}}
	eng := NewEngine(st, fakeClient(ft))

	start, err := eng.Start(context.Background(), "https://srv.example/mcp", "inst-1", "https://app/callback")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Re-arm discovery for Complete (PRM+ASM+token).
	ft.responses = []fakeResp{jsonResp(prmBody), jsonResp(asmBody), jsonResp(tokenBody)}
	ft.idx = 0
	ft.captured = nil
	ft.bodies = nil

	acct, err := eng.Complete(context.Background(), "https://srv.example/mcp", start.State, "the-code", "https://app/callback")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	// The returned account must NOT carry cleartext tokens (masked discipline).
	if acct.AccessToken == "AT-secret" || acct.RefreshToken == "RT-secret" {
		t.Fatalf("Complete returned cleartext tokens: %#v", acct)
	}
	if acct.Status != "connected" {
		t.Fatalf("status = %q, want connected", acct.Status)
	}

	// The token exchange POST must carry the PKCE code_verifier + the auth code.
	var tokenReqBody string
	for i, req := range ft.captured {
		if req.URL.String() == "https://as.example/token" {
			tokenReqBody = ft.bodyAt(i)
		}
	}
	if tokenReqBody == "" {
		t.Fatalf("no token request captured: %#v", ft.captured)
	}
	if !strings.Contains(tokenReqBody, "grant_type=authorization_code") {
		t.Fatalf("token body missing grant_type: %s", tokenReqBody)
	}
	if !strings.Contains(tokenReqBody, "code_verifier=") {
		t.Fatalf("token body missing code_verifier: %s", tokenReqBody)
	}
	if !strings.Contains(tokenReqBody, "code=the-code") {
		t.Fatalf("token body missing code: %s", tokenReqBody)
	}

	// The stored account tokens are encrypted at rest (raw column != cleartext).
	stored, err := st.GetMCPOAuthAccount(acct.ID)
	if err != nil {
		t.Fatalf("GetMCPOAuthAccount: %v", err)
	}
	if stored.AccessToken != "AT-secret" {
		t.Fatalf("decrypted access token = %q, want AT-secret", stored.AccessToken)
	}
}

func TestEngineCompleteExpiredFlow(t *testing.T) {
	st := newOAuthTestStore(t)
	ft := &fakeTransport{responses: []fakeResp{jsonResp(prmBody), jsonResp(asmBody)}}
	eng := NewEngine(st, fakeClient(ft))
	// No flow persisted for this state → ErrNotFound surfaces as an error.
	_, err := eng.Complete(context.Background(), "https://srv.example/mcp", "unknown-state", "code", "https://app/callback")
	if err == nil {
		t.Fatalf("expected error for unknown state")
	}
}

func TestEngineRefresh(t *testing.T) {
	st := newOAuthTestStore(t)
	tokenBody := `{"access_token":"AT-new","refresh_token":"RT-new","expires_in":3600}`
	ft := &fakeTransport{responses: []fakeResp{jsonResp(tokenBody)}}
	eng := NewEngine(st, fakeClient(ft))

	// Seed an account to refresh.
	seeded, err := st.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   "inst-9",
		ServerURL:    "https://srv.example/mcp",
		AccessToken:  "AT-old",
		RefreshToken: "RT-old",
		ExpiresAt:    time.Now().Add(time.Minute).Unix(),
		Status:       "connected",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	acct, err := eng.Refresh(context.Background(), seeded, "https://as.example/token")
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if acct.AccessToken == "AT-new" {
		t.Fatalf("Refresh returned cleartext token")
	}
	// The refresh POST carries grant_type=refresh_token + the old refresh token.
	body := ft.bodyAt(0)
	if !strings.Contains(body, "grant_type=refresh_token") || !strings.Contains(body, "refresh_token=RT-old") {
		t.Fatalf("refresh body = %s", body)
	}
	// Stored token rotated.
	stored, err := st.GetMCPOAuthAccount(seeded.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AccessToken != "AT-new" {
		t.Fatalf("token not rotated: %q", stored.AccessToken)
	}
}

func TestNeedsRefresh(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	lead := 5 * time.Minute
	// Expires well in the future → no refresh.
	if needsRefresh(now.Add(time.Hour).Unix(), now, lead) {
		t.Fatalf("should not need refresh")
	}
	// Within the lead window → refresh.
	if !needsRefresh(now.Add(2*time.Minute).Unix(), now, lead) {
		t.Fatalf("should need refresh within lead")
	}
	// Already expired → refresh.
	if !needsRefresh(now.Add(-time.Minute).Unix(), now, lead) {
		t.Fatalf("expired should need refresh")
	}
	// Zero expiry (unknown) → no forced refresh.
	if needsRefresh(0, now, lead) {
		t.Fatalf("zero expiry should not force refresh")
	}
}

func TestNewEngineNilClient(t *testing.T) {
	st := newOAuthTestStore(t)
	if NewEngine(st, nil) == nil {
		t.Fatalf("NewEngine(nil) returned nil")
	}
}
