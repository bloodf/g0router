package store

import (
	"errors"
	"testing"
)

// insertTestFeatureFlag adds a flag row directly (the store exposes no create
// method — flags are seeded out of band; the surface is list + get + toggle).
func insertTestFeatureFlag(t *testing.T, st *Store, key, description string, enabled bool) int64 {
	t.Helper()
	res, err := st.db.Exec(
		"INSERT INTO feature_flags (key, enabled, description, created_at) VALUES (?, ?, ?, ?)",
		key, boolToInt(enabled), description, "2026-06-14T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert feature flag: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func TestFeatureFlagListEmpty(t *testing.T) {
	st := newTestStore(t)
	flags, err := st.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	if len(flags) != 0 {
		t.Fatalf("len(flags) = %d, want 0", len(flags))
	}
}

func TestFeatureFlagListAndGet(t *testing.T) {
	st := newTestStore(t)
	id1 := insertTestFeatureFlag(t, st, "mcp_gateway", "Enable MCP gateway", true)
	insertTestFeatureFlag(t, st, "rtk_compression", "Enable RTK compression", false)

	flags, err := st.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("len(flags) = %d, want 2", len(flags))
	}
	// ORDER BY id ASC.
	if flags[0].ID != id1 || flags[0].Key != "mcp_gateway" {
		t.Fatalf("flags[0] = %+v", flags[0])
	}
	if !flags[0].Enabled {
		t.Fatalf("flags[0].Enabled = false, want true")
	}
	if flags[1].Enabled {
		t.Fatalf("flags[1].Enabled = true, want false")
	}
	if flags[0].CreatedAt == "" {
		t.Fatalf("flags[0].CreatedAt empty")
	}

	got, err := st.GetFeatureFlagByID(id1)
	if err != nil {
		t.Fatalf("GetFeatureFlagByID: %v", err)
	}
	if got.Key != "mcp_gateway" || got.Description != "Enable MCP gateway" {
		t.Fatalf("got = %+v", got)
	}
}

func TestFeatureFlagGetMissing(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.GetFeatureFlagByID(99); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetFeatureFlagByID missing err = %v, want ErrNotFound", err)
	}
}

func TestFeatureFlagSetEnabled(t *testing.T) {
	st := newTestStore(t)
	id := insertTestFeatureFlag(t, st, "new_dashboard", "New React dashboard", false)

	updated, err := st.SetFeatureFlagEnabled(id, true)
	if err != nil {
		t.Fatalf("SetFeatureFlagEnabled: %v", err)
	}
	if !updated.Enabled {
		t.Fatalf("updated.Enabled = false, want true")
	}
	if updated.ID != id || updated.Key != "new_dashboard" {
		t.Fatalf("updated = %+v", updated)
	}

	// Persisted.
	got, err := st.GetFeatureFlagByID(id)
	if err != nil {
		t.Fatalf("GetFeatureFlagByID after toggle: %v", err)
	}
	if !got.Enabled {
		t.Fatalf("persisted Enabled = false, want true")
	}

	// Toggle back off.
	updated, err = st.SetFeatureFlagEnabled(id, false)
	if err != nil {
		t.Fatalf("SetFeatureFlagEnabled off: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("updated.Enabled = true, want false")
	}
}

func TestFeatureFlagSetEnabledMissing(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.SetFeatureFlagEnabled(404, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetFeatureFlagEnabled missing err = %v, want ErrNotFound", err)
	}
}
