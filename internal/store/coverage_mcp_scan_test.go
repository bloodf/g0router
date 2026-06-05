package store

import (
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
)

// TestListMCPClientsScanBadArgsJSON exercises the decodeJSON error in
// scanMCPClient (args field) by inserting a client with malformed JSON args.
func TestListMCPClientsScanBadArgsJSON(t *testing.T) {
	s := openTestStore(t)

	// Insert directly with bad JSON in args column.
	if _, err := s.db.Exec(`
		INSERT INTO mcp_clients (id, name, transport, args, is_active, health_status, created_at)
		VALUES ('bad-client', 'bad-client', 'stdio', 'not-valid-json', 1, 'unknown', datetime('now'))
	`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	// ListMCPClients should fail when decoding malformed args.
	_, err := s.ListMCPClients()
	if err == nil {
		t.Fatal("ListMCPClients with bad args JSON should error")
	}
}

// TestListMCPInstancesScanBadArgsJSON exercises the decodeJSON error in
// scanMCPInstance by inserting an instance with malformed JSON in args.
func TestListMCPInstancesScanBadArgsJSON(t *testing.T) {
	s := openTestStore(t)

	// Insert directly with bad JSON in args column.
	if _, err := s.db.Exec(`
		INSERT INTO mcp_instances (id, name, server_key, launch_type, transport, args, is_active, health_status, created_at, updated_at)
		VALUES ('bad-inst', 'bad-inst', 'key1', 'command', 'stdio', 'bad-json[', 1, 'unknown', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	_, err := s.ListMCPInstances()
	if err == nil {
		t.Fatal("ListMCPInstances with bad args JSON should error")
	}
}

// TestGetMCPClientBadArgsJSON exercises the decodeJSON error in GetMCPClient.
func TestGetMCPClientBadArgsJSON(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.db.Exec(`
		INSERT INTO mcp_clients (id, name, transport, args, is_active, health_status, created_at)
		VALUES ('bad-get', 'bad-get', 'stdio', '{invalid', 1, 'unknown', datetime('now'))
	`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	_, err := s.GetMCPClient("bad-get")
	if err == nil {
		t.Fatal("GetMCPClient with bad args JSON should error")
	}
}

// TestListCombosDecodeStepsBadJSON exercises the json.Unmarshal error in
// scanCombo (line 189-190): a combo with invalid JSON in steps column.
func TestListCombosDecodeStepsBadJSON(t *testing.T) {
	s := openTestStore(t)

	if _, err := s.db.Exec(`
		INSERT INTO combos (id, name, steps, strategy, is_active, created_at, updated_at)
		VALUES ('bad-combo', 'bad-combo', 'not-json', 'fallback', 1, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	_, err := s.ListCombos()
	if err == nil {
		t.Fatal("ListCombos with bad steps JSON should error")
	}
}

// Ensure mcp import is used.
var _ = mcp.TransportStdio
