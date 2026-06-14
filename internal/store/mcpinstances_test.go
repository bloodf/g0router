package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func newMCPTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestMCPClientCRUD(t *testing.T) {
	st := newMCPTestStore(t)

	created, err := st.CreateMCPClient(&MCPClient{
		Name:   "exa",
		Type:   "default",
		Config: map[string]any{"transport": "http", "oauth": false},
	})
	if err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("CreateMCPClient: empty ID")
	}
	if created.Config["transport"] != "http" {
		t.Fatalf("config not round-tripped: %+v", created.Config)
	}

	got, err := st.GetMCPClient(created.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	if got.Name != "exa" || got.Type != "default" {
		t.Fatalf("GetMCPClient = %+v", got)
	}
	if got.Config["oauth"] != false {
		t.Fatalf("config bool not round-tripped: %+v", got.Config)
	}

	list, err := st.ListMCPClients()
	if err != nil {
		t.Fatalf("ListMCPClients: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListMCPClients len = %d, want 1", len(list))
	}

	// Upsert (update existing by ID).
	got.Name = "exa-renamed"
	if err := st.UpsertMCPClient(got); err != nil {
		t.Fatalf("UpsertMCPClient: %v", err)
	}
	reread, _ := st.GetMCPClient(got.ID)
	if reread.Name != "exa-renamed" {
		t.Fatalf("upsert did not update name: %+v", reread)
	}

	if err := st.DeleteMCPClient(got.ID); err != nil {
		t.Fatalf("DeleteMCPClient: %v", err)
	}
	if _, err := st.GetMCPClient(got.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete, GetMCPClient err = %v, want ErrNotFound", err)
	}
}

func TestMCPInstanceCRUDAndStatus(t *testing.T) {
	st := newMCPTestStore(t)

	created, err := st.CreateMCPInstance(&MCPInstance{
		Name:      "browsermcp",
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@browsermcp/mcp@latest"},
		Env:       map[string]string{"NODE_ENV": "production"},
		Status:    "stopped",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("empty ID")
	}

	got, err := st.GetMCPInstance(created.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance: %v", err)
	}
	if len(got.Args) != 2 || got.Args[1] != "@browsermcp/mcp@latest" {
		t.Fatalf("args not round-tripped: %+v", got.Args)
	}
	if got.Env["NODE_ENV"] != "production" {
		t.Fatalf("env not round-tripped: %+v", got.Env)
	}

	list, err := st.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListMCPInstances len = %d, want 1", len(list))
	}

	// Status transition.
	if err := st.SetMCPInstanceStatus(created.ID, "running"); err != nil {
		t.Fatalf("SetMCPInstanceStatus: %v", err)
	}
	got, _ = st.GetMCPInstance(created.ID)
	if got.Status != "running" {
		t.Fatalf("status = %q, want running", got.Status)
	}

	// Update mutable fields.
	got.Name = "browsermcp-2"
	if err := st.UpdateMCPInstance(got); err != nil {
		t.Fatalf("UpdateMCPInstance: %v", err)
	}
	reread, _ := st.GetMCPInstance(got.ID)
	if reread.Name != "browsermcp-2" {
		t.Fatalf("update did not persist name: %+v", reread)
	}

	if err := st.DeleteMCPInstance(created.ID); err != nil {
		t.Fatalf("DeleteMCPInstance: %v", err)
	}
	if _, err := st.GetMCPInstance(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete, err = %v, want ErrNotFound", err)
	}
}
