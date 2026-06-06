package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestStartTunnelHealthNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	srv.StartTunnelHealth(ctx)
	cancel()
}

func TestStartTunnelHealthDefaultInterval(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	srv.tunnelHealthInterval = 0 // triggers default

	ctx, cancel := context.WithCancel(context.Background())
	srv.StartTunnelHealth(ctx)
	time.Sleep(5 * time.Millisecond)
	cancel()
	time.Sleep(5 * time.Millisecond) // let goroutine exit
}

func TestStartTunnelHealthTickerFires(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	srv.tunnelHealthInterval = 2 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	srv.StartTunnelHealth(ctx)

	// Wait long enough for the ticker to fire at least once
	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(5 * time.Millisecond) // let goroutine exit
}

func TestRunTunnelHealthOnceNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	srv.runTunnelHealthOnce()
}

func TestRunTunnelHealthOnceListError(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	s.Close()

	srv := NewServer(ServerConfig{Store: s})
	srv.runTunnelHealthOnce()
}

func TestRunTunnelHealthOnceDisabledTunnel(t *testing.T) {
	s := newAPITestStore(t)
	if err := s.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "cloudflare",
		IsEnabled: false,
		URL:       "http://127.0.0.1:1",
		Status:    "inactive",
	}); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runTunnelHealthOnce()
}

func TestRunTunnelHealthOnceNon200Status(t *testing.T) {
	s := newAPITestStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	if err := s.UpsertTunnelConfig(store.TunnelConfig{
		Type:      "cloudflare",
		IsEnabled: true,
		URL:       ts.URL,
		Status:    "active",
	}); err != nil {
		t.Fatalf("UpsertTunnelConfig: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runTunnelHealthOnce()

	cfg, err := s.GetTunnelConfig("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnelConfig: %v", err)
	}
	if cfg.Status != "error" {
		t.Fatalf("status = %q, want error", cfg.Status)
	}
}

func TestStartProxyPoolHealthNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	srv.StartProxyPoolHealth(ctx)
	cancel()
}

func TestStartProxyPoolHealthDefaultInterval(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	srv.proxyPoolHealthInterval = 0 // triggers default

	ctx, cancel := context.WithCancel(context.Background())
	srv.StartProxyPoolHealth(ctx)
	time.Sleep(5 * time.Millisecond)
	cancel()
	time.Sleep(5 * time.Millisecond) // let goroutine exit
}

func TestStartProxyPoolHealthTickerFires(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	srv.proxyPoolHealthInterval = 2 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	srv.StartProxyPoolHealth(ctx)

	time.Sleep(10 * time.Millisecond)
	cancel()
	time.Sleep(5 * time.Millisecond) // let goroutine exit
}

// TestRunTunnelHealthOnceUpdateErrorOnSuccess exercises the UpdateTunnelStatus
// error-log branch when the health check succeeds but the store update fails.
func TestRunTunnelHealthOnceUpdateErrorOnSuccess(t *testing.T) {
	s := newAPITestStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay the response so the store can be closed while the request is in flight.
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
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

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.Close()
	}()

	// Must not panic even when UpdateTunnelStatus errors.
	srv.runTunnelHealthOnce()
}

// TestRunTunnelHealthOnceUpdateErrorOnHTTPError exercises the UpdateTunnelStatus
// error-log branch inside the HTTP-error path.
func TestRunTunnelHealthOnceUpdateErrorOnHTTPError(t *testing.T) {
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

	go func() {
		// The unreachable check returns almost instantly; close the store
		// right after runTunnelHealthOnce starts.
		time.Sleep(5 * time.Millisecond)
		s.Close()
	}()

	// Must not panic even when UpdateTunnelStatus errors.
	srv.runTunnelHealthOnce()
}

func TestRunProxyPoolHealthOnceNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	srv.runProxyPoolHealthOnce()
}

func TestRunProxyPoolHealthOnceListError(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	s.Close()

	srv := NewServer(ServerConfig{Store: s})
	srv.runProxyPoolHealthOnce()
}
