package store

import (
	"errors"
	"testing"
)

func TestMCPToolGroupCreateGetListUpdateDelete(t *testing.T) {
	st := newTestStore(t)

	// List empty.
	groups, err := st.ListMCPToolGroups()
	if err != nil {
		t.Fatalf("ListMCPToolGroups empty: %v", err)
	}
	if len(groups) != 0 {
		t.Fatalf("len(groups) = %d, want 0", len(groups))
	}

	// Create.
	created, err := st.CreateMCPToolGroup(&MCPToolGroup{
		Name:     "File Operations",
		ToolIDs:  []string{"read_file", "write_file"},
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}
	if created.ID == 0 {
		t.Fatalf("created.ID = 0, want non-zero numeric id")
	}
	if created.Name != "File Operations" || len(created.ToolIDs) != 2 {
		t.Fatalf("created = %+v", created)
	}
	if !created.IsActive {
		t.Fatalf("created.IsActive = false, want true")
	}
	if created.CreatedAt == "" {
		t.Fatalf("created.CreatedAt empty, want ISO-8601 timestamp")
	}

	// Get.
	got, err := st.GetMCPToolGroup(created.ID)
	if err != nil {
		t.Fatalf("GetMCPToolGroup: %v", err)
	}
	if got.Name != "File Operations" || got.ToolIDs[0] != "read_file" {
		t.Fatalf("got = %+v", got)
	}

	// List has one.
	groups, err = st.ListMCPToolGroups()
	if err != nil {
		t.Fatalf("ListMCPToolGroups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}

	// Update (toggle is_active off + rename).
	updated, err := st.UpdateMCPToolGroup(created.ID, &MCPToolGroup{
		Name:     "File Ops",
		ToolIDs:  []string{"read_file"},
		IsActive: false,
	})
	if err != nil {
		t.Fatalf("UpdateMCPToolGroup: %v", err)
	}
	if updated.Name != "File Ops" || updated.IsActive {
		t.Fatalf("updated = %+v", updated)
	}
	if len(updated.ToolIDs) != 1 {
		t.Fatalf("updated.ToolIDs = %v, want 1 element", updated.ToolIDs)
	}

	// Persisted.
	got, err = st.GetMCPToolGroup(created.ID)
	if err != nil {
		t.Fatalf("GetMCPToolGroup after update: %v", err)
	}
	if got.IsActive || got.Name != "File Ops" {
		t.Fatalf("persisted = %+v", got)
	}

	// Delete.
	if err := st.DeleteMCPToolGroup(created.ID); err != nil {
		t.Fatalf("DeleteMCPToolGroup: %v", err)
	}
	if _, err := st.GetMCPToolGroup(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMCPToolGroup after delete err = %v, want ErrNotFound", err)
	}
}

func TestMCPToolGroupGetMissing(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.GetMCPToolGroup(999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMCPToolGroup missing err = %v, want ErrNotFound", err)
	}
}

func TestMCPToolGroupUpdateMissing(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.UpdateMCPToolGroup(404, &MCPToolGroup{Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("UpdateMCPToolGroup missing err = %v, want ErrNotFound", err)
	}
}

func TestMCPToolGroupDeleteMissing(t *testing.T) {
	st := newTestStore(t)
	if err := st.DeleteMCPToolGroup(404); !errors.Is(err, ErrNotFound) {
		t.Fatalf("DeleteMCPToolGroup missing err = %v, want ErrNotFound", err)
	}
}
