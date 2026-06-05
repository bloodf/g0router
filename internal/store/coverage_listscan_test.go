package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
)

// invalidManifest carries a tool whose InputSchema is malformed JSON, which makes
// json.Marshal of the manifest fail — exercising the encodeJSON error branch of
// the manifest-update methods without any driver fault.
func invalidManifest(clientID string) mcp.Manifest {
	return mcp.Manifest{
		ClientID: clientID,
		Tools:    []mcp.Tool{{Name: "t", InputSchema: json.RawMessage("{invalid")}},
	}
}

func TestUpdateMCPClientManifestEncodeError(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{Name: "enc", Transport: mcp.TransportStdio, Command: strPtr("x")}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if err := s.UpdateMCPClientManifest(client.ID, invalidManifest(client.ID)); err == nil {
		t.Fatal("UpdateMCPClientManifest invalid manifest: want encode error")
	}
}

func TestUpdateMCPInstanceManifestEncodeError(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "enc-inst")
	if err := s.UpdateMCPInstanceManifest(inst.ID, invalidManifest(inst.ID)); err == nil {
		t.Fatal("UpdateMCPInstanceManifest invalid manifest: want encode error")
	}
}

// DeleteMCPClient success path (existing tests only delete a missing client).
func TestDeleteMCPClientSuccess(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{Name: "del", Transport: mcp.TransportStdio, Command: strPtr("x")}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if err := s.DeleteMCPClient(client.ID); err != nil {
		t.Fatalf("DeleteMCPClient: %v", err)
	}
	if _, err := s.GetMCPClient(client.ID); err == nil {
		t.Fatal("client should be gone after delete")
	}
}

// scanMCPInstance headers/env decode-error branches via the List path.
func TestScanMCPInstanceHeadersEnvDecodeError(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "hdr-env-scan")

	// Corrupt headers only.
	if _, err := s.db.Exec("UPDATE mcp_instances SET args='[]', headers=?, env='{}', tool_manifest=NULL WHERE id=?", "{bad", inst.ID); err != nil {
		t.Fatalf("corrupt headers: %v", err)
	}
	if _, err := s.GetMCPInstance(inst.ID); err == nil {
		t.Fatal("GetMCPInstance corrupt headers: want decode error")
	}
	// Corrupt env only.
	if _, err := s.db.Exec("UPDATE mcp_instances SET args='[]', headers='{}', env=?, tool_manifest=NULL WHERE id=?", "{bad", inst.ID); err != nil {
		t.Fatalf("corrupt env: %v", err)
	}
	if _, err := s.GetMCPInstance(inst.ID); err == nil {
		t.Fatal("GetMCPInstance corrupt env: want decode error")
	}
}

// These tests drive the in-loop scan-error branch of the List* methods: a
// corrupt JSON column on a row that the List query returns forces the per-row
// scanX call to fail mid-iteration (distinct from the single-row Get* path).

func TestListConnectionsScanError(t *testing.T) {
	s := openTestStore(t)
	conn := &Connection{Provider: "p", AuthType: AuthTypeAPIKey, IsActive: true}
	if err := s.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if _, err := s.db.Exec("UPDATE connections SET provider_specific_data = ? WHERE id = ?", "{bad", conn.ID); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if _, err := s.ListConnections(); err == nil {
		t.Fatal("ListConnections corrupt row: want scan error")
	}
	if _, err := s.GetConnections("p"); err == nil {
		t.Fatal("GetConnections corrupt row: want scan error")
	}
	if _, err := s.GetActiveConnections("p"); err == nil {
		t.Fatal("GetActiveConnections corrupt row: want scan error")
	}
}

func TestListCombosScanError(t *testing.T) {
	s := openTestStore(t)
	combo := &Combo{Name: "c", Steps: []ComboStep{{Provider: "p", Model: "m"}}, IsActive: true}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	if _, err := s.db.Exec("UPDATE combos SET steps = ? WHERE id = ?", "{bad", combo.ID); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if _, err := s.ListCombos(); err == nil {
		t.Fatal("ListCombos corrupt row: want scan error")
	}
}

func TestListMCPClientsScanError(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{Name: "c", Transport: mcp.TransportStdio, Command: strPtr("x")}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if _, err := s.db.Exec("UPDATE mcp_clients SET args = ? WHERE id = ?", "{bad", client.ID); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if _, err := s.ListMCPClients(); err == nil {
		t.Fatal("ListMCPClients corrupt row: want scan error")
	}
}

func TestListMCPInstancesScanError(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "list-scan")
	if _, err := s.db.Exec("UPDATE mcp_instances SET args = ? WHERE id = ?", "{bad", inst.ID); err != nil {
		t.Fatalf("corrupt: %v", err)
	}
	if _, err := s.ListMCPInstances(); err == nil {
		t.Fatal("ListMCPInstances corrupt row: want scan error")
	}
	if _, err := s.ListActiveMCPInstances(); err == nil {
		t.Fatal("ListActiveMCPInstances corrupt row: want scan error")
	}
}

// scanMCPInstance manifest-decode branch: a corrupt tool_manifest on a listed
// instance exercises the manifest decode path inside the scanner.
func TestScanMCPInstanceManifestDecodeError(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "manifest-scan")
	if _, err := s.db.Exec(
		"UPDATE mcp_instances SET args='[]', headers='{}', env='{}', tool_manifest=? WHERE id=?",
		"{bad", inst.ID,
	); err != nil {
		t.Fatalf("corrupt manifest: %v", err)
	}
	if _, err := s.GetMCPInstance(inst.ID); err == nil {
		t.Fatal("GetMCPInstance corrupt manifest: want decode error")
	}
}

// MCPInstance.Config CWD branch.
func TestMCPInstanceConfigCWD(t *testing.T) {
	s := openTestStore(t)
	inst := &MCPInstance{
		Name:       "cwd-inst",
		ServerKey:  "k",
		LaunchType: mcp.LaunchCommand,
		Transport:  mcp.TransportStdio,
		Command:    strPtr("/bin/echo"),
		CWD:        strPtr("/tmp/work"),
		IsActive:   true,
	}
	if err := s.CreateMCPInstance(inst); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	got, err := s.GetMCPInstance(inst.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	cfg := got.Config()
	if cfg.CWD != "/tmp/work" {
		t.Fatalf("Config CWD = %q, want /tmp/work", cfg.CWD)
	}
}

// GetValidMCPOAuthAccount: a valid (non-expired) account returns successfully,
// exercising the success branch and a non-empty expiry parse.
func TestGetValidMCPOAuthAccountSuccessParsesExpiry(t *testing.T) {
	s := openTestStore(t)
	inst := createOAuthTestInstance(t, s, "valid-parse")
	if err := s.UpsertMCPOAuthAccount(&MCPOAuthAccount{
		InstanceID:   inst.ID,
		AccountLabel: "ok",
		ResourceURI:  "https://res",
		AccessToken:  "tok",
		Scopes:       []string{"a", "b"},
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := s.GetValidMCPOAuthAccount(inst.ID, "https://res")
	if err != nil {
		t.Fatalf("GetValidMCPOAuthAccount: %v", err)
	}
	if got.AccessToken != "tok" || len(got.Scopes) != 2 {
		t.Fatalf("account = %+v", got)
	}
}
