package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func newMitmTestStore(t *testing.T) *Store {
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

func TestEnsureMitmToolsSeedsTwoNamedTools(t *testing.T) {
	st := newMitmTestStore(t)
	if err := st.EnsureMitmTools(); err != nil {
		t.Fatalf("EnsureMitmTools: %v", err)
	}
	// Idempotent: a second call must not duplicate rows.
	if err := st.EnsureMitmTools(); err != nil {
		t.Fatalf("EnsureMitmTools (second): %v", err)
	}
	tools, err := st.ListMitmTools()
	if err != nil {
		t.Fatalf("ListMitmTools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("ListMitmTools len = %d, want 2", len(tools))
	}
	// Deterministic order by id: mitm-1 (Request Inspector) then mitm-2.
	if tools[0].ID != "mitm-1" || tools[0].Name != "Request Inspector" {
		t.Fatalf("tools[0] = %+v, want mitm-1 Request Inspector", tools[0])
	}
	if tools[1].ID != "mitm-2" || tools[1].Name != "Response Modifier" {
		t.Fatalf("tools[1] = %+v, want mitm-2 Response Modifier", tools[1])
	}
}

func TestUpsertGetMitmTool(t *testing.T) {
	st := newMitmTestStore(t)
	want := MitmTool{ID: "mitm-9", Name: "Custom", Enabled: true, DNSOverride: "host.local", Status: "active"}
	if err := st.UpsertMitmTool(want); err != nil {
		t.Fatalf("UpsertMitmTool: %v", err)
	}
	got, err := st.GetMitmTool("mitm-9")
	if err != nil {
		t.Fatalf("GetMitmTool: %v", err)
	}
	if got.ID != want.ID || got.Name != want.Name || got.Enabled != want.Enabled ||
		got.DNSOverride != want.DNSOverride || got.Status != want.Status {
		t.Fatalf("GetMitmTool = %+v, want %+v", got, want)
	}
	// Upsert overwrites on conflict.
	want.Name = "Renamed"
	if err := st.UpsertMitmTool(want); err != nil {
		t.Fatalf("UpsertMitmTool (update): %v", err)
	}
	got, _ = st.GetMitmTool("mitm-9")
	if got.Name != "Renamed" {
		t.Fatalf("upsert did not update name: %+v", got)
	}
}

func TestGetMitmToolNotFound(t *testing.T) {
	st := newMitmTestStore(t)
	if _, err := st.GetMitmTool("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetMitmTool unknown id err = %v, want ErrNotFound", err)
	}
}

func TestSetMitmToolEnabledDerivesStatus(t *testing.T) {
	st := newMitmTestStore(t)
	if err := st.EnsureMitmTools(); err != nil {
		t.Fatalf("EnsureMitmTools: %v", err)
	}
	// mitm-2 starts disabled/inactive; enabling derives status=active.
	got, err := st.SetMitmToolEnabled("mitm-2", true)
	if err != nil {
		t.Fatalf("SetMitmToolEnabled: %v", err)
	}
	if !got.Enabled || got.Status != "active" {
		t.Fatalf("enabled tool = %+v, want enabled+active", got)
	}
	// Disabling derives status=inactive.
	got, err = st.SetMitmToolEnabled("mitm-2", false)
	if err != nil {
		t.Fatalf("SetMitmToolEnabled (disable): %v", err)
	}
	if got.Enabled || got.Status != "inactive" {
		t.Fatalf("disabled tool = %+v, want disabled+inactive", got)
	}
	// Unknown id → ErrNotFound.
	if _, err := st.SetMitmToolEnabled("nope", true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetMitmToolEnabled unknown id err = %v, want ErrNotFound", err)
	}
}

func TestMitmGlobalEnableRoundTrips(t *testing.T) {
	st := newMitmTestStore(t)
	// Default (unset) is false.
	enabled, err := st.GetMitmEnabled()
	if err != nil {
		t.Fatalf("GetMitmEnabled (default): %v", err)
	}
	if enabled {
		t.Fatalf("default GetMitmEnabled = true, want false")
	}
	if err := st.SetMitmEnabled(true); err != nil {
		t.Fatalf("SetMitmEnabled(true): %v", err)
	}
	enabled, err = st.GetMitmEnabled()
	if err != nil {
		t.Fatalf("GetMitmEnabled: %v", err)
	}
	if !enabled {
		t.Fatalf("GetMitmEnabled after SetMitmEnabled(true) = false, want true")
	}
	if err := st.SetMitmEnabled(false); err != nil {
		t.Fatalf("SetMitmEnabled(false): %v", err)
	}
	enabled, _ = st.GetMitmEnabled()
	if enabled {
		t.Fatalf("GetMitmEnabled after SetMitmEnabled(false) = true, want false")
	}
}
