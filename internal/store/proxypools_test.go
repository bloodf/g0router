package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func newProxyTestStore(t *testing.T) *Store {
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

func TestProxyPoolCRUD(t *testing.T) {
	st := newProxyTestStore(t)

	created, err := st.CreateProxyPool(&ProxyPool{
		Name:     "US East",
		Protocol: "https",
		Host:     "us-east.proxy.example.com",
		Port:     8080,
		Username: "user1",
		Password: "s3cret",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("created pool has empty ID")
	}
	if created.CreatedAt == 0 || created.UpdatedAt == 0 {
		t.Fatalf("created pool missing timestamps")
	}

	got, err := st.GetProxyPoolByID(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPoolByID: %v", err)
	}
	if got.Name != "US East" || got.Host != "us-east.proxy.example.com" || got.Port != 8080 {
		t.Fatalf("unexpected pool: %+v", got)
	}
	if got.Password != "s3cret" {
		t.Fatalf("password did not round-trip: %q", got.Password)
	}

	list, err := st.ListProxyPools(nil)
	if err != nil {
		t.Fatalf("ListProxyPools: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 pool, got %d", len(list))
	}

	got.Name = "US East 2"
	got.IsActive = false
	if err := st.UpdateProxyPool(got); err != nil {
		t.Fatalf("UpdateProxyPool: %v", err)
	}
	updated, err := st.GetProxyPoolByID(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPoolByID after update: %v", err)
	}
	if updated.Name != "US East 2" || updated.IsActive {
		t.Fatalf("update not persisted: %+v", updated)
	}

	if err := st.DeleteProxyPool(created.ID); err != nil {
		t.Fatalf("DeleteProxyPool: %v", err)
	}
	if _, err := st.GetProxyPoolByID(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
	if err := st.DeleteProxyPool(created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestProxyPoolListActiveFilter(t *testing.T) {
	st := newProxyTestStore(t)

	if _, err := st.CreateProxyPool(&ProxyPool{Name: "active", Host: "a.example.com", IsActive: true}); err != nil {
		t.Fatalf("create active: %v", err)
	}
	if _, err := st.CreateProxyPool(&ProxyPool{Name: "inactive", Host: "b.example.com", IsActive: false}); err != nil {
		t.Fatalf("create inactive: %v", err)
	}

	all, err := st.ListProxyPools(nil)
	if err != nil {
		t.Fatalf("ListProxyPools(nil): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(all))
	}

	active := true
	onlyActive, err := st.ListProxyPools(&active)
	if err != nil {
		t.Fatalf("ListProxyPools(active): %v", err)
	}
	if len(onlyActive) != 1 || !onlyActive[0].IsActive {
		t.Fatalf("expected 1 active pool, got %+v", onlyActive)
	}
}

func TestProxyPoolPasswordEncryptedAtRest(t *testing.T) {
	st := newProxyTestStore(t)
	created, err := st.CreateProxyPool(&ProxyPool{Name: "p", Host: "h.example.com", Password: "topsecret"})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	var raw string
	if err := st.DB().QueryRow("SELECT password_enc FROM proxy_pools WHERE id = ?", created.ID).Scan(&raw); err != nil {
		t.Fatalf("read raw column: %v", err)
	}
	if raw == "" {
		t.Fatalf("password_enc is empty for a non-empty password")
	}
	if raw == "topsecret" {
		t.Fatalf("password stored in cleartext: %q", raw)
	}
}

func TestSetProxyPoolCheck(t *testing.T) {
	st := newProxyTestStore(t)
	created, err := st.CreateProxyPool(&ProxyPool{Name: "p", Host: "h.example.com"})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	if err := st.SetProxyPoolCheck(created.ID, "ok", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("SetProxyPoolCheck: %v", err)
	}
	got, err := st.GetProxyPoolByID(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPoolByID: %v", err)
	}
	if got.LastCheckStatus != "ok" || got.LastCheckAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("check not persisted: status=%q at=%q", got.LastCheckStatus, got.LastCheckAt)
	}
}

func TestCountConnectionsUsingProxyPoolZero(t *testing.T) {
	// The bound (>0) path is asserted in connections_test.go (T-proxywire),
	// where the Connection.ProxyPoolID linkage is added. Here we prove the
	// unbound count is 0 against an empty connections table.
	st := newProxyTestStore(t)
	pool, err := st.CreateProxyPool(&ProxyPool{Name: "p", Host: "h.example.com"})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	n, err := st.CountConnectionsUsingProxyPool(pool.ID)
	if err != nil {
		t.Fatalf("CountConnectionsUsingProxyPool: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 bound connections, got %d", n)
	}
}
