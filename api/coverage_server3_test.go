package api

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// TestObserveRequestMetricNilMetrics covers the early-return when metrics is nil.
func TestObserveRequestMetricNilMetrics(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
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
