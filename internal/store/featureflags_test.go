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

func TestFeatureFlagListSeededOnly(t *testing.T) {
	st := newTestStore(t)
	flags, err := st.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	// The migration seeds semantic_cache (bf-core-2, D8) and vk_mandatory
	// (bf-gov-4, D1); no other flags are seeded in a fresh store.
	if len(flags) != 2 {
		t.Fatalf("len(flags) = %d, want 2 (seeded semantic_cache + vk_mandatory)", len(flags))
	}
	seeded := map[string]bool{}
	for _, f := range flags {
		seeded[f.Key] = true
	}
	if !seeded["semantic_cache"] || !seeded["vk_mandatory"] {
		t.Fatalf("seeded flag keys = %v, want semantic_cache + vk_mandatory", seeded)
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
	// 2 inserted + the migration-seeded semantic_cache (bf-core-2, D8) and
	// vk_mandatory (bf-gov-4, D1) flags.
	if len(flags) != 4 {
		t.Fatalf("len(flags) = %d, want 4 (2 inserted + 2 seeded)", len(flags))
	}
	byKey := map[string]*FeatureFlag{}
	for _, f := range flags {
		byKey[f.Key] = f
	}
	mcp := byKey["mcp_gateway"]
	if mcp == nil || mcp.ID != id1 {
		t.Fatalf("mcp_gateway flag = %+v", mcp)
	}
	if !mcp.Enabled {
		t.Fatalf("mcp_gateway.Enabled = false, want true")
	}
	if rtk := byKey["rtk_compression"]; rtk == nil || rtk.Enabled {
		t.Fatalf("rtk_compression flag = %+v, want present + disabled", rtk)
	}
	if mcp.CreatedAt == "" {
		t.Fatalf("mcp_gateway.CreatedAt empty")
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

func TestIsFeatureEnabledTrue(t *testing.T) {
	st := newTestStore(t)
	insertTestFeatureFlag(t, st, "on_flag", "", true)
	got, err := st.IsFeatureEnabled("on_flag")
	if err != nil {
		t.Fatalf("IsFeatureEnabled: %v", err)
	}
	if !got {
		t.Fatal("IsFeatureEnabled(on_flag) = false, want true")
	}
}

func TestIsFeatureEnabledFalse(t *testing.T) {
	st := newTestStore(t)
	insertTestFeatureFlag(t, st, "off_flag", "", false)
	got, err := st.IsFeatureEnabled("off_flag")
	if err != nil {
		t.Fatalf("IsFeatureEnabled: %v", err)
	}
	if got {
		t.Fatal("IsFeatureEnabled(off_flag) = true, want false")
	}
}

// TestIsFeatureEnabledMissing verifies fail-OFF: a key with no row reports
// (false, nil) so the hook stays a clean no-op (D8).
func TestIsFeatureEnabledMissing(t *testing.T) {
	st := newTestStore(t)
	got, err := st.IsFeatureEnabled("nonexistent")
	if err != nil {
		t.Fatalf("IsFeatureEnabled missing err = %v, want nil (fail-OFF)", err)
	}
	if got {
		t.Fatal("IsFeatureEnabled(missing) = true, want false (fail-OFF)")
	}
}

// TestSemanticCacheFlagSeeded verifies the semantic_cache flag row is seeded by
// the migration (OFF by default) so the admin toggle has a row to flip (D8).
func TestSemanticCacheFlagSeeded(t *testing.T) {
	st := newTestStore(t)
	flags, err := st.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	var found *FeatureFlag
	for _, f := range flags {
		if f.Key == "semantic_cache" {
			found = f
			break
		}
	}
	if found == nil {
		t.Fatal("semantic_cache flag not seeded")
	}
	if found.Enabled {
		t.Fatal("semantic_cache flag seeded enabled=true, want false (OFF by default)")
	}
}

// TestVKMandatoryFlagSeeded verifies the vk_mandatory flag row is seeded by the
// migration OFF by default (bf-gov-4, D1) — the backward-compat guarantee: a
// fresh/upgraded store starts with mandatory mode OFF so an absent VK is allowed.
func TestVKMandatoryFlagSeeded(t *testing.T) {
	st := newTestStore(t)
	enabled, err := st.IsFeatureEnabled("vk_mandatory")
	if err != nil {
		t.Fatalf("IsFeatureEnabled(vk_mandatory): %v", err)
	}
	if enabled {
		t.Fatal("vk_mandatory flag seeded enabled=true, want false (OFF by default)")
	}
	flags, err := st.ListFeatureFlags()
	if err != nil {
		t.Fatalf("ListFeatureFlags: %v", err)
	}
	var found *FeatureFlag
	for _, f := range flags {
		if f.Key == "vk_mandatory" {
			found = f
			break
		}
	}
	if found == nil {
		t.Fatal("vk_mandatory flag not seeded")
	}
}
