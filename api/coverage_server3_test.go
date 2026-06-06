package api

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
)

// TestObserveRequestMetricNilMetrics covers the early-return when metrics is nil.
func TestObserveRequestMetricNilMetrics(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
	srv.metrics = nil // override the default collector
	// nil metrics → should return without panic.
	srv.observeRequestMetric(requestLogMetadata{}, nil, nil, 200, time.Second)
}

// TestResponseCacheForNonPositiveTTL covers the ttl <= 0 branch.
func TestResponseCacheForNonPositiveTTL(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
	if got := srv.responseCacheFor(0); got != nil {
		t.Fatalf("expected nil for ttl=0, got %v", got)
	}
	if got := srv.responseCacheFor(-1 * time.Second); got != nil {
		t.Fatalf("expected nil for negative ttl, got %v", got)
	}
}

// TestRecordAuditIfMutationAppendError covers the log.Printf branch when
// AppendAudit returns an error.
func TestRecordAuditIfMutationAppendError(t *testing.T) {
	s, err := store.NewStore(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	srv := NewServer(ServerConfig{Port: 0, Store: s})
	ctx := makeCtxWithHeaders("POST", "/api/settings", nil)
	ctx.SetStatusCode(200)
	srv.recordAuditIfMutation(ctx)
}

// TestStartLogRetentionNilRunFallsBack covers the run==nil branch in StartLogRetention.
func TestStartLogRetentionNilRunFallsBack(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)
	seedLog(t, s, "stale", time.Now().UTC().Add(-30*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.runRetention = nil

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

// TestStartConnectionRefreshNilRunFallsBack covers the run==nil branch in StartConnectionRefresh.
func TestStartConnectionRefreshNilRunFallsBack(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)
	srv.runConnectionRefresh = nil
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

// TestRunProxyPoolHealthOnceInactivePoolSkipped covers the !pool.IsActive branch.
func TestRunProxyPoolHealthOnceInactivePoolSkipped(t *testing.T) {
	s := newAPITestStore(t)
	if _, err := s.CreateProxyPool(store.ProxyPool{
		Name:     "inactive",
		Protocol: "http",
		Host:     "127.0.0.1",
		Port:     9999,
		IsActive: false,
	}); err != nil {
		t.Fatalf("CreateProxyPool: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.runProxyPoolHealthOnce()
}

// TestRoutesMethodNotAllowed covers uncovered requireMethod and switch-default branches.
func TestRoutesMethodNotAllowed(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{Store: s})

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/guardrails/test"},
		{http.MethodPost, "/api/models/custom/1"},
		{http.MethodPut, "/api/chat-sessions"},
		{http.MethodPatch, "/api/chat-sessions/1"},
		{http.MethodGet, "/api/alert-channels/1/test"},
		{http.MethodGet, "/api/auth/password"},
	}

	for _, tc := range tests {
		req, _ := http.NewRequest(tc.method, baseURL+tc.path, nil)
		resp, err := httpClient().Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s = %d, want 405", tc.method, tc.path, resp.StatusCode)
		}
	}
}
