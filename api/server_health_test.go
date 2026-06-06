package api

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestRunTunnelHealthOnceUpdatesActiveForReachable(t *testing.T) {
	s := newAPITestStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if err := s.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "cloudflare",
		IsEnabled: true,
		URL:       ts.URL,
		Status:    "inactive",
	}); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runTunnelHealthOnce()

	cfg, err := s.GetTunnelConfig("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if cfg.Status != "active" {
		t.Fatalf("status = %q, want active", cfg.Status)
	}
	if cfg.LastError != "" {
		t.Fatalf("last_error = %q, want empty", cfg.LastError)
	}
}

func TestRunTunnelHealthOnceUpdatesErrorForUnreachable(t *testing.T) {
	s := newAPITestStore(t)
	if err := s.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "tailscale",
		IsEnabled: true,
		URL:       "http://127.0.0.1:1",
		Status:    "inactive",
	}); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runTunnelHealthOnce()

	cfg, err := s.GetTunnelConfig("tailscale")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if cfg.Status != "error" {
		t.Fatalf("status = %q, want error", cfg.Status)
	}
	if cfg.LastError == "" {
		t.Fatal("last_error should be set for unreachable tunnel")
	}
}

func TestRunProxyPoolHealthOnceUpdatesOkForReachable(t *testing.T) {
	s := newAPITestStore(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	tcpAddr := ln.Addr().(*net.TCPAddr)
	pool, err := s.CreateProxyPool(store.ProxyPool{
		Name:     "test-pool",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     tcpAddr.Port,
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runProxyPoolHealthOnce()

	updated, err := s.GetProxyPool(pool.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if updated.LastCheckStatus != "ok" {
		t.Fatalf("last_check_status = %q, want ok", updated.LastCheckStatus)
	}
}

func TestRunProxyPoolHealthOnceUpdatesErrorForUnreachable(t *testing.T) {
	s := newAPITestStore(t)
	pool, err := s.CreateProxyPool(store.ProxyPool{
		Name:     "test-pool",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     1,
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runProxyPoolHealthOnce()

	updated, err := s.GetProxyPool(pool.ID)
	if err != nil {
		t.Fatalf("GetProxyPool: %v", err)
	}
	if updated.LastCheckStatus != "error" {
		t.Fatalf("last_check_status = %q, want error", updated.LastCheckStatus)
	}
}

func TestStartTunnelHealthRunsAtStartupAndStopsOnCancel(t *testing.T) {
	s := newAPITestStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if err := s.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "cloudflare",
		IsEnabled: true,
		URL:       ts.URL,
		Status:    "inactive",
	}); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.tunnelHealthInterval = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartTunnelHealth(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for {
		cfg, err := s.GetTunnelConfig("cloudflare")
		if err != nil {
			t.Fatalf("GetTunnelConfig: %v", err)
		}
		if cfg.Status == "active" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("startup tunnel health pass did not update status; status = %q", cfg.Status)
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()
}

func TestStartProxyPoolHealthRunsAtStartupAndStopsOnCancel(t *testing.T) {
	s := newAPITestStore(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	tcpAddr := ln.Addr().(*net.TCPAddr)
	pool, err := s.CreateProxyPool(store.ProxyPool{
		Name:     "test-pool",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     tcpAddr.Port,
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.proxyPoolHealthInterval = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartProxyPoolHealth(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for {
		updated, err := s.GetProxyPool(pool.ID)
		if err != nil {
			t.Fatalf("GetProxyPool: %v", err)
		}
		if updated.LastCheckStatus == "ok" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("startup proxy pool health pass did not update status; status = %q", updated.LastCheckStatus)
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()
}
