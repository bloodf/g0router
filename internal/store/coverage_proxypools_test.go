package store

import (
	"testing"
)

func TestUpdateProxyPoolStatus(t *testing.T) {
	s := openTestStore(t)
	s.SetEncKey("test-key-for-proxy-pools")

	pool := ProxyPool{
		Name:     "us-east-1",
		Protocol: "http",
		Host:     "proxy.example.com",
		Port:     8080,
		IsActive: true,
	}

	created, err := s.CreateProxyPool(pool)
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	if err := s.UpdateProxyPoolStatus(created.ID, "ok", ""); err != nil {
		t.Fatalf("UpdateProxyPoolStatus: %v", err)
	}

	got, err := s.GetProxyPool(created.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if got.LastCheckStatus != "ok" {
		t.Fatalf("last_check_status = %q, want ok", got.LastCheckStatus)
	}
}
