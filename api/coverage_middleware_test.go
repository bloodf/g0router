package api

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
)

// TestClassifySourceIPNilReturnsPublic exercises the ip==nil branch (line 163)
// in classifySourceIP.
func TestClassifySourceIPNilReturnsPublic(t *testing.T) {
	got := classifySourceIP(nil)
	if got != "public" {
		t.Fatalf("classifySourceIP(nil) = %q, want public", got)
	}
}

// TestClassifySourceIPNilParsed exercises using a nil net.IP (zero value).
func TestClassifySourceIPNilNetIP(t *testing.T) {
	var ip net.IP // nil
	got := classifySourceIP(ip)
	if got != "public" {
		t.Fatalf("classifySourceIP(nil net.IP) = %q, want public", got)
	}
}

// TestApplyMiddlewareSourceNotAllowedReturns403 exercises the !sourceAllowed
// branch in applyMiddleware (lines 59-62): a /v1/* request from localhost
// when AllowedSources is "tailscale" only returns 403.
func TestApplyMiddlewareSourceNotAllowedReturns403(t *testing.T) {
	s := newAPITestStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.AllowedSources = []string{"tailscale"} // local/lan not in allowed list
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s, RequireAPIKey: false})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	resp, err := httpClient().Get(base + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("AllowedSources restricted: got %d, want 403", resp.StatusCode)
	}
}

// TestApplyMiddlewareOptionsPreflightReturns204 exercises the MethodOptions
// branch in applyMiddleware (lines 54-57).
func TestApplyMiddlewareOptionsPreflightReturns204(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	req, _ := http.NewRequest(http.MethodOptions, base+"/v1/chat/completions", nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("OPTIONS = %d, want 204", resp.StatusCode)
	}
}

// TestNewServerInferenceEngineAsConnectionRefresher exercises the
// InferenceEngine.(ConnectionRefresher) type assertion in NewServer (line 124-126).
func TestNewServerInferenceEngineAsConnectionRefresher(t *testing.T) {
	engine := &fakeRefreshEngine{}
	srv := NewServer(ServerConfig{InferenceEngine: engine})
	if srv.connRefresher == nil {
		t.Fatal("connRefresher should be set when InferenceEngine implements ConnectionRefresher")
	}
}

// fakeRefreshEngine satisfies handlers.InferenceEngine + ConnectionRefresher.
type fakeRefreshEngine struct{}

func (f *fakeRefreshEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, nil
}
func (f *fakeRefreshEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}
func (f *fakeRefreshEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return nil, nil
}
func (f *fakeRefreshEngine) RefreshExpiringConnections(ctx context.Context, now time.Time) []proxy.RefreshOutcome {
	return nil
}

// TestStartLogRetentionNilRunFallsBack exercises the `run == nil` branch in
// StartLogRetention (lines 144-147) by setting runRetention to nil before
// calling StartLogRetention.
func TestStartLogRetentionNilRunFallsBack(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)
	seedLog(t, s, "stale-run", time.Now().UTC().Add(-30*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.runRetention = nil // forces fallback to runLogRetentionOnce

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartLogRetention(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for {
		entries, err := s.GetUsage(store.UsageFilter{})
		if err != nil {
			t.Fatalf("GetUsage: %v", err)
		}
		if len(entries) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("nil run fallback did not delete old log")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestStartConnectionRefreshNilRunFallsBack exercises the `run == nil` branch in
// StartConnectionRefresh (lines 208-210) by setting runConnectionRefresh to nil.
func TestStartConnectionRefreshNilRunFallsBack(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)
	srv.runConnectionRefresh = nil // forces fallback to runConnectionRefreshOnce
	srv.connectionRefreshInterval = 5 * time.Millisecond

	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c3", Provider: "openai", Name: "nil-run", Failed: true, Reason: "x"}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartConnectionRefresh(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for notifier.count() < 1 {
		if time.Now().After(deadline) {
			t.Fatal("nil run fallback did not notify")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
}
