package store

import (
	"errors"
	"testing"
)

func TestMCPToolGroupLifecycle(t *testing.T) {
	s := openTestStore(t)

	// Create
	group, err := s.CreateMCPToolGroup("tools-a", []string{"t1", "t2"}, true)
	if err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}
	if group.ID == 0 {
		t.Error("ID should be set")
	}
	if group.Name != "tools-a" {
		t.Errorf("Name = %q, want tools-a", group.Name)
	}
	if len(group.ToolIDs) != 2 || group.ToolIDs[0] != "t1" {
		t.Errorf("ToolIDs = %v, want [t1 t2]", group.ToolIDs)
	}
	if !group.IsActive {
		t.Error("expected IsActive true")
	}

	// Get
	got, err := s.GetMCPToolGroup(group.ID)
	if err != nil {
		t.Fatalf("GetMCPToolGroup: %v", err)
	}
	if got.Name != group.Name {
		t.Errorf("got.Name = %q, want %q", got.Name, group.Name)
	}

	// Get by name
	byName, err := s.GetMCPToolGroupByName(group.Name)
	if err != nil {
		t.Fatalf("GetMCPToolGroupByName: %v", err)
	}
	if byName.ID != group.ID {
		t.Errorf("byName.ID = %d, want %d", byName.ID, group.ID)
	}

	// List
	list, err := s.ListMCPToolGroups()
	if err != nil {
		t.Fatalf("ListMCPToolGroups: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d, want 1", len(list))
	}

	// Update
	if err := s.UpdateMCPToolGroup(group.ID, "tools-b", []string{"t3"}, false); err != nil {
		t.Fatalf("UpdateMCPToolGroup: %v", err)
	}
	updated, err := s.GetMCPToolGroup(group.ID)
	if err != nil {
		t.Fatalf("GetMCPToolGroup after update: %v", err)
	}
	if updated.Name != "tools-b" {
		t.Errorf("Name = %q, want tools-b", updated.Name)
	}
	if updated.IsActive {
		t.Error("expected IsActive false")
	}

	// GetByName inactive returns not found
	if _, err := s.GetMCPToolGroupByName("tools-b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMCPToolGroupByName inactive: expected ErrNotFound, got %v", err)
	}

	// Delete
	if err := s.DeleteMCPToolGroup(group.ID); err != nil {
		t.Fatalf("DeleteMCPToolGroup: %v", err)
	}
	if _, err := s.GetMCPToolGroup(group.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMCPToolGroup after delete: expected ErrNotFound, got %v", err)
	}
}

func TestCreateMCPToolGroupDuplicateName(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.CreateMCPToolGroup("dup", []string{"t1"}, true); err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}
	if _, err := s.CreateMCPToolGroup("dup", []string{"t2"}, true); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestUpdateMCPToolGroupDuplicateName(t *testing.T) {
	s := openTestStore(t)
	g1, _ := s.CreateMCPToolGroup("a", []string{"t1"}, true)
	if _, err := s.CreateMCPToolGroup("b", []string{"t2"}, true); err != nil {
		t.Fatalf("CreateMCPToolGroup: %v", err)
	}
	if err := s.UpdateMCPToolGroup(g1.ID, "b", []string{"t1"}, true); !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("expected ErrDuplicateName, got %v", err)
	}
}

func TestUpdateMCPToolGroupNotFound(t *testing.T) {
	s := openTestStore(t)
	if err := s.UpdateMCPToolGroup(9999, "x", []string{"t1"}, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteMCPToolGroupNotFound(t *testing.T) {
	s := openTestStore(t)
	if err := s.DeleteMCPToolGroup(9999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetMCPToolGroupByNameNotFound(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.GetMCPToolGroupByName("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
