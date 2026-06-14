package platform

import (
	"errors"
	"net"
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// publicResolver resolves any host to a public IP so the SSRF guard allows it
// deterministically without touching DNS.
func publicResolver(host string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil
}

func newProxyService(t *testing.T) (*ProxyPoolService, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return NewProxyPoolService(st), st
}

func TestTestConnectivityReachable(t *testing.T) {
	svc, st := newProxyService(t)
	pool, err := st.CreateProxyPool(&store.ProxyPool{Name: "p", Protocol: "http", Host: "proxy.example.com", Port: 8080, IsActive: true})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	svc.SetResolver(publicResolver)
	svc.SetProber(func(proxyURL, target string) (int, error) {
		return 42, nil
	})
	res, err := svc.TestConnectivity(pool.ID)
	if err != nil {
		t.Fatalf("TestConnectivity: %v", err)
	}
	if !res.OK || res.LatencyMs != 42 || res.Status != "ok" {
		t.Fatalf("expected reachable result, got %+v", res)
	}

	// Persisted on the pool.
	got, _ := st.GetProxyPoolByID(pool.ID)
	if got.LastCheckStatus != "ok" || got.LastCheckAt == "" {
		t.Fatalf("check not persisted: %+v", got)
	}
}

func TestTestConnectivityUnreachable(t *testing.T) {
	svc, st := newProxyService(t)
	pool, err := st.CreateProxyPool(&store.ProxyPool{Name: "p", Protocol: "http", Host: "proxy.example.com", Port: 8080, IsActive: true})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	svc.SetResolver(publicResolver)
	svc.SetProber(func(proxyURL, target string) (int, error) {
		return 0, errors.New("connection refused")
	})
	res, err := svc.TestConnectivity(pool.ID)
	if err != nil {
		t.Fatalf("TestConnectivity returned error: %v", err)
	}
	if res.OK || res.Status != "error" {
		t.Fatalf("expected unreachable result, got %+v", res)
	}
	got, _ := st.GetProxyPoolByID(pool.ID)
	if got.LastCheckStatus != "error" {
		t.Fatalf("error status not persisted: %+v", got)
	}
}

func TestTestConnectivitySSRFRefused(t *testing.T) {
	svc, st := newProxyService(t)
	// A proxy host pointing at a private address must be refused before dialing.
	pool, err := st.CreateProxyPool(&store.ProxyPool{Name: "p", Protocol: "http", Host: "10.0.0.5", Port: 8080, IsActive: true})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}
	probed := false
	svc.SetProber(func(proxyURL, target string) (int, error) {
		probed = true
		return 1, nil
	})
	res, err := svc.TestConnectivity(pool.ID)
	if err != nil {
		t.Fatalf("TestConnectivity: %v", err)
	}
	if res.OK {
		t.Fatalf("expected SSRF refusal, got ok=true")
	}
	if res.Status != "blocked" {
		t.Fatalf("expected status=blocked, got %q", res.Status)
	}
	if probed {
		t.Fatalf("prober was called for a blocked target (SSRF guard bypassed)")
	}
}
