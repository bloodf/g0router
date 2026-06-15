package store

import (
	"errors"
	"reflect"
	"testing"
)

// TestVKMCPConfigCRUD proves the additive virtual_key_mcp_configs store round-
// trips create→get→list-by-VK→update→delete and reports ErrNotFound on a missing
// id (PAR-BF-MCP-033, D2). Mirrors the mcp_tool_groups store contract.
func TestVKMCPConfigCRUD(t *testing.T) {
	st := newTestStore(t)

	created, err := st.CreateVKMCPConfig(&VKMCPConfig{
		VirtualKeyID:       "vk1",
		MCPClientID:        "exa",
		ToolsToExecute:     []string{"exa-*"},
		ToolsToAutoExecute: []string{"exa-search"},
		ConfigHash:         "hash-1",
	})
	if err != nil {
		t.Fatalf("CreateVKMCPConfig: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("created config has zero id")
	}

	got, err := st.GetVKMCPConfig(created.ID)
	if err != nil {
		t.Fatalf("GetVKMCPConfig: %v", err)
	}
	if got.VirtualKeyID != "vk1" || got.MCPClientID != "exa" {
		t.Fatalf("got = %+v", got)
	}
	if !reflect.DeepEqual(got.ToolsToExecute, []string{"exa-*"}) {
		t.Fatalf("ToolsToExecute = %v", got.ToolsToExecute)
	}
	if !reflect.DeepEqual(got.ToolsToAutoExecute, []string{"exa-search"}) {
		t.Fatalf("ToolsToAutoExecute = %v", got.ToolsToAutoExecute)
	}
	if got.ConfigHash != "hash-1" {
		t.Fatalf("ConfigHash = %q", got.ConfigHash)
	}

	// Update.
	got.ToolsToExecute = []string{"*"}
	got.ConfigHash = "hash-2"
	updated, err := st.UpdateVKMCPConfig(got.ID, got)
	if err != nil {
		t.Fatalf("UpdateVKMCPConfig: %v", err)
	}
	if !reflect.DeepEqual(updated.ToolsToExecute, []string{"*"}) || updated.ConfigHash != "hash-2" {
		t.Fatalf("after update = %+v", updated)
	}

	// Delete.
	if err := st.DeleteVKMCPConfig(created.ID); err != nil {
		t.Fatalf("DeleteVKMCPConfig: %v", err)
	}
	if _, err := st.GetVKMCPConfig(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete err = %v, want ErrNotFound", err)
	}
}

// TestVKMCPConfigManyToMany proves the table is a real junction: one VK scopes N
// clients and one client is scoped by N VKs (D2).
func TestVKMCPConfigManyToMany(t *testing.T) {
	st := newTestStore(t)

	for _, client := range []string{"exa", "fs"} {
		if _, err := st.CreateVKMCPConfig(&VKMCPConfig{VirtualKeyID: "vk1", MCPClientID: client}); err != nil {
			t.Fatalf("create vk1/%s: %v", client, err)
		}
	}
	if _, err := st.CreateVKMCPConfig(&VKMCPConfig{VirtualKeyID: "vk2", MCPClientID: "exa"}); err != nil {
		t.Fatalf("create vk2/exa: %v", err)
	}

	vk1, err := st.ListVKMCPConfigsByVK("vk1")
	if err != nil {
		t.Fatalf("ListVKMCPConfigsByVK vk1: %v", err)
	}
	if len(vk1) != 2 {
		t.Fatalf("vk1 configs = %d, want 2", len(vk1))
	}
	vk2, err := st.ListVKMCPConfigsByVK("vk2")
	if err != nil {
		t.Fatalf("ListVKMCPConfigsByVK vk2: %v", err)
	}
	if len(vk2) != 1 {
		t.Fatalf("vk2 configs = %d, want 1", len(vk2))
	}
}

// TestVKMCPConfigEmptyAndStarPatterns proves empty and ["*"] pattern arrays round-
// trip (the deny-all and allow-all D4 sentinels must survive storage).
func TestVKMCPConfigEmptyAndStarPatterns(t *testing.T) {
	st := newTestStore(t)

	deny, err := st.CreateVKMCPConfig(&VKMCPConfig{VirtualKeyID: "vk1", MCPClientID: "exa", ToolsToExecute: []string{}})
	if err != nil {
		t.Fatalf("create deny: %v", err)
	}
	gotDeny, _ := st.GetVKMCPConfig(deny.ID)
	if gotDeny.ToolsToExecute == nil || len(gotDeny.ToolsToExecute) != 0 {
		t.Fatalf("empty patterns did not round-trip: %v", gotDeny.ToolsToExecute)
	}

	all, err := st.CreateVKMCPConfig(&VKMCPConfig{VirtualKeyID: "vk2", MCPClientID: "fs", ToolsToExecute: []string{"*"}})
	if err != nil {
		t.Fatalf("create all: %v", err)
	}
	gotAll, _ := st.GetVKMCPConfig(all.ID)
	if !reflect.DeepEqual(gotAll.ToolsToExecute, []string{"*"}) {
		t.Fatalf("star patterns did not round-trip: %v", gotAll.ToolsToExecute)
	}
}
