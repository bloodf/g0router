package store

import (
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
)

// Corrupting persisted JSON/timestamp columns lets us drive the scan-time
// error branches (decode/parse) that the happy paths never reach.

func TestScanMCPOAuthAccountErrors(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "scan-acct")
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   inst.ID,
		AccountLabel: "d",
		ResourceURI:  "https://res",
		AccessToken:  "t",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// Corrupt scopes JSON -> decode error.
	if _, err := s.db.Exec("UPDATE mcp_oauth_accounts SET scopes = ? WHERE instance_id = ?", "{bad", inst.ID); err != nil {
		t.Fatalf("corrupt scopes: %v", err)
	}
	if _, err := s.ListMCPOAuthAccounts(inst.ID); err == nil {
		t.Fatal("ListMCPOAuthAccounts corrupt scopes: want error")
	}
	if _, err := s.GetValidMCPOAuthAccount(inst.ID, "https://res"); err == nil {
		t.Fatal("GetValidMCPOAuthAccount corrupt scopes: want error")
	}

	// Fix scopes, corrupt metadata.
	if _, err := s.db.Exec("UPDATE mcp_oauth_accounts SET scopes = ?, auth_metadata = ? WHERE instance_id = ?", "[]", "{bad", inst.ID); err != nil {
		t.Fatalf("corrupt metadata: %v", err)
	}
	if _, err := s.ListMCPOAuthAccounts(inst.ID); err == nil {
		t.Fatal("ListMCPOAuthAccounts corrupt metadata: want error")
	}

	// Fix metadata, corrupt expiry.
	if _, err := s.db.Exec("UPDATE mcp_oauth_accounts SET auth_metadata = ?, expires_at = ? WHERE instance_id = ?", "{}", "not-a-time", inst.ID); err != nil {
		t.Fatalf("corrupt expiry: %v", err)
	}
	if _, err := s.ListMCPOAuthAccounts(inst.ID); err == nil {
		t.Fatal("ListMCPOAuthAccounts corrupt expiry: want error")
	}
}

func TestScanMCPOAuthFlowExpiryError(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "scan-flow")
	flow := &MCPOAuthFlow{
		InstanceID:         inst.ID,
		State:              "st",
		CodeVerifierSecret: "v",
		ExpiresAt:          time.Now().Add(time.Hour),
	}
	if err := s.CreateMCPOAuthFlow(flow); err != nil {
		t.Fatalf("CreateMCPOAuthFlow: %v", err)
	}
	if _, err := s.db.Exec("UPDATE mcp_oauth_flows SET expires_at = ? WHERE instance_id = ?", "not-a-time", inst.ID); err != nil {
		t.Fatalf("corrupt expiry: %v", err)
	}
	if _, err := s.ConsumeMCPOAuthFlow(inst.ID, "st"); err == nil {
		t.Fatal("ConsumeMCPOAuthFlow corrupt expiry: want error")
	}
}

func TestScanOAuthSessionExpiryError(t *testing.T) {
	s := openTestStore(t)
	session := &OAuthSession{Provider: "p", State: "st", ExpiresAt: time.Now().Add(time.Hour)}
	if err := s.CreateOAuthSession(session); err != nil {
		t.Fatalf("CreateOAuthSession: %v", err)
	}
	if _, err := s.db.Exec("UPDATE oauth_sessions SET expires_at = ? WHERE id = ?", "not-a-time", session.ID); err != nil {
		t.Fatalf("corrupt expiry: %v", err)
	}
	if _, err := s.GetOAuthSession("st"); err == nil {
		t.Fatal("GetOAuthSession corrupt expiry: want error")
	}
}

func TestScanMCPInstanceDecodeErrors(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "scan-inst")

	for _, col := range []string{"args", "headers", "env", "tool_manifest"} {
		if _, err := s.db.Exec("UPDATE mcp_instances SET "+col+" = '[]', headers = '{}', env = '{}', tool_manifest = NULL WHERE id = ?", inst.ID); err != nil {
			t.Fatalf("reset: %v", err)
		}
		if _, err := s.db.Exec("UPDATE mcp_instances SET "+col+" = ? WHERE id = ?", "{bad", inst.ID); err != nil {
			t.Fatalf("corrupt %s: %v", col, err)
		}
		if _, err := s.GetMCPInstance(inst.ID); err == nil {
			t.Fatalf("GetMCPInstance corrupt %s: want error", col)
		}
	}
}

func TestScanMCPClientArgsEnvDecodeError(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{Name: "scan-cli", Transport: mcp.TransportStdio, Command: strPtr("x")}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if _, err := s.db.Exec("UPDATE mcp_clients SET args = ? WHERE id = ?", "{bad", client.ID); err != nil {
		t.Fatalf("corrupt args: %v", err)
	}
	if _, err := s.GetMCPClient(client.ID); err == nil {
		t.Fatal("GetMCPClient corrupt args: want error")
	}
	if _, err := s.db.Exec("UPDATE mcp_clients SET args = '[]', env = ? WHERE id = ?", "{bad", client.ID); err != nil {
		t.Fatalf("corrupt env: %v", err)
	}
	if _, err := s.GetMCPClient(client.ID); err == nil {
		t.Fatal("GetMCPClient corrupt env: want error")
	}
}

func TestUsageScanTimestampError(t *testing.T) {
	s := openTestStore(t)
	if err := s.LogRequest(&RequestLogEntry{RequestID: "r", Provider: "p", Model: "m", AuthType: "api_key"}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}
	if _, err := s.db.Exec("UPDATE request_log SET timestamp = ? WHERE request_id = ?", "not-a-time", "r"); err != nil {
		t.Fatalf("corrupt timestamp: %v", err)
	}
	if _, err := s.GetUsage(UsageFilter{}); err == nil {
		t.Fatal("GetUsage corrupt timestamp: want error")
	}
}
