package store

import (
	"testing"
)

func TestClosedStorePhase14ErrorPaths(t *testing.T) {
	s := closedStore(t)

	checks := []struct {
		name string
		fn   func() error
	}{
		{"ListDisabledModels", func() error { _, err := s.ListDisabledModels(); return err }},
		{"CreateDisabledModel", func() error { _, err := s.CreateDisabledModel("p", "m"); return err }},
		{"DeleteDisabledModel", func() error { return s.DeleteDisabledModel("p", "m") }},
		{"IsModelDisabled", func() error { _, err := s.IsModelDisabled("p", "m"); return err }},
		{"ListCustomModels", func() error { _, err := s.ListCustomModels(); return err }},
		{"CreateCustomModel", func() error { _, err := s.CreateCustomModel("p", "m", "d"); return err }},
		{"GetCustomModel", func() error { _, err := s.GetCustomModel("x"); return err }},
		{"DeleteCustomModel", func() error { return s.DeleteCustomModel("x") }},
		{"ListProxyPools", func() error { _, err := s.ListProxyPools(); return err }},
		{"CreateProxyPool", func() error { _, err := s.CreateProxyPool(ProxyPool{Name: "n", Protocol: "http", Host: "h", Port: 8080}); return err }},
		{"GetProxyPool", func() error { _, err := s.GetProxyPool("x"); return err }},
		{"UpdateProxyPool", func() error { return s.UpdateProxyPool("x", ProxyPool{Name: "n", Protocol: "http", Host: "h", Port: 8080}) }},
		{"DeleteProxyPool", func() error { return s.DeleteProxyPool("x") }},
		{"UpdateConnectionProxyPool", func() error { return s.UpdateConnectionProxyPool("x", nil) }},
		{"GetConnectionProxyPoolID", func() error { _, err := s.GetConnectionProxyPoolID("x"); return err }},
	}

	for _, c := range checks {
		if err := c.fn(); err == nil {
			t.Errorf("%s on closed DB: want error, got nil", c.name)
		}
	}
}

func TestDeleteCustomModelNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.DeleteCustomModel("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for non-existent custom model")
	}
}

func TestDeleteProxyPoolNotFound(t *testing.T) {
	s := openTestStore(t)
	err := s.DeleteProxyPool("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for non-existent proxy pool")
	}
}

func TestGetProxyPoolNotFound(t *testing.T) {
	s := openTestStore(t)
	_, err := s.GetProxyPool("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for non-existent proxy pool")
	}
}

func TestScanProxyPoolNullFields(t *testing.T) {
	s := openTestStore(t)
	pool := ProxyPool{Name: "test", Protocol: "http", Host: "host", Port: 8080}
	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	got, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if got.Password != "" {
		t.Fatalf("password = %q, want empty", got.Password)
	}
}
