package store

import (
	"errors"
	"testing"
	"time"
)

func TestMCPOAuthAccountEncryptedAtRest(t *testing.T) {
	st := newMCPTestStore(t)

	const accessTok = "sk-access-PLAINTEXT-12345"
	const refreshTok = "rt-refresh-PLAINTEXT-67890"

	acct := &MCPOAuthAccount{
		InstanceID:   "inst-1",
		ServerURL:    "https://mcp.example.com",
		AccessToken:  accessTok,
		RefreshToken: refreshTok,
		ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		Scope:        "read write",
		Status:       "connected",
	}
	saved, err := st.UpsertMCPOAuthAccount(acct)
	if err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}
	if saved.ID == "" {
		t.Fatalf("empty account ID")
	}

	// Read back decrypts the tokens for the OAuth engine's internal use.
	got, err := st.GetMCPOAuthAccount(saved.ID)
	if err != nil {
		t.Fatalf("GetMCPOAuthAccount: %v", err)
	}
	if got.AccessToken != accessTok || got.RefreshToken != refreshTok {
		t.Fatalf("tokens not round-tripped: %+v", got)
	}
	if got.Status != "connected" || got.Scope != "read write" {
		t.Fatalf("metadata not round-tripped: %+v", got)
	}

	// The raw *_enc columns MUST NOT contain the cleartext (encrypted at rest).
	var accessEnc, refreshEnc string
	row := st.DB().QueryRow(
		"SELECT access_token_enc, refresh_token_enc FROM mcp_oauth_accounts WHERE id = ?", saved.ID)
	if err := row.Scan(&accessEnc, &refreshEnc); err != nil {
		t.Fatalf("scan raw enc columns: %v", err)
	}
	if accessEnc == accessTok || accessEnc == "" {
		t.Fatalf("access_token_enc not encrypted at rest: %q", accessEnc)
	}
	if refreshEnc == refreshTok || refreshEnc == "" {
		t.Fatalf("refresh_token_enc not encrypted at rest: %q", refreshEnc)
	}
	if contains(accessEnc, accessTok) || contains(refreshEnc, refreshTok) {
		t.Fatalf("cleartext leaked into *_enc column")
	}
}

func TestMCPOAuthAccountByInstanceAndList(t *testing.T) {
	st := newMCPTestStore(t)

	a1, err := st.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID: "inst-A", ServerURL: "https://a", AccessToken: "tA", Status: "connected",
	})
	if err != nil {
		t.Fatalf("upsert a1: %v", err)
	}
	if _, err := st.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID: "inst-B", ServerURL: "https://b", AccessToken: "tB", Status: "connected",
	}); err != nil {
		t.Fatalf("upsert a2: %v", err)
	}

	byInst, err := st.GetMCPOAuthAccountByInstance("inst-A")
	if err != nil {
		t.Fatalf("GetMCPOAuthAccountByInstance: %v", err)
	}
	if byInst.ID != a1.ID || byInst.AccessToken != "tA" {
		t.Fatalf("by-instance mismatch: %+v", byInst)
	}

	list, err := st.ListMCPOAuthAccounts()
	if err != nil {
		t.Fatalf("ListMCPOAuthAccounts: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list len = %d, want 2", len(list))
	}

	if err := st.DeleteMCPOAuthAccount(a1.ID); err != nil {
		t.Fatalf("DeleteMCPOAuthAccount: %v", err)
	}
	if _, err := st.GetMCPOAuthAccount(a1.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete err = %v, want ErrNotFound", err)
	}
}

func TestMCPOAuthFlowConsumeAndExpire(t *testing.T) {
	st := newMCPTestStore(t)

	const verifier = "pkce-verifier-PLAINTEXT-abcdef"
	flow := &MCPOAuthFlow{
		State:       "state-123",
		InstanceID:  "inst-1",
		ServerURL:   "https://mcp.example.com",
		Verifier:    verifier,
		RedirectURI: "https://cb",
		ExpiresAt:   time.Now().Add(time.Hour).Unix(),
	}
	if err := st.CreateMCPOAuthFlow(flow); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}

	// Verifier encrypted at rest.
	var verifierEnc string
	if err := st.DB().QueryRow(
		"SELECT verifier_enc FROM mcp_oauth_flows WHERE state = ?", "state-123").Scan(&verifierEnc); err != nil {
		t.Fatalf("scan verifier_enc: %v", err)
	}
	if verifierEnc == verifier || verifierEnc == "" {
		t.Fatalf("verifier_enc not encrypted at rest: %q", verifierEnc)
	}

	// Consume returns + deletes, decrypting the verifier.
	got, err := st.ConsumeMCPOAuthFlow("state-123")
	if err != nil {
		t.Fatalf("ConsumeMCPOAuthFlow: %v", err)
	}
	if got.Verifier != verifier || got.InstanceID != "inst-1" {
		t.Fatalf("consume mismatch: %+v", got)
	}
	// Second consume → gone.
	if _, err := st.ConsumeMCPOAuthFlow("state-123"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second consume err = %v, want ErrNotFound", err)
	}

	// Expired flow → ErrNotFound on consume (and deleted).
	if err := st.CreateMCPOAuthFlow(&MCPOAuthFlow{
		State: "expired", Verifier: "v", ExpiresAt: time.Now().Add(-time.Hour).Unix(),
	}); err != nil {
		t.Fatalf("create expired flow: %v", err)
	}
	if _, err := st.ConsumeMCPOAuthFlow("expired"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired consume err = %v, want ErrNotFound", err)
	}
}

func contains(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
