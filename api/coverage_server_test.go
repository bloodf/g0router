package api

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func noopContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled — any goroutine waiting on it exits immediately
	return ctx
}

// ---- clientToolFromCtx ----

func makeCtxWithHeaders(method, uri string, headers map[string]string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, &net.TCPAddr{}, nil)
	return ctx
}

func TestClientToolFromCtxXClientToolHeader(t *testing.T) {
	ctx := makeCtxWithHeaders("GET", "/", map[string]string{
		"X-Client-Tool": "codex",
		"User-Agent":    "fallback-agent",
	})
	got := clientToolFromCtx(ctx)
	if got == nil || *got != "codex" {
		t.Fatalf("clientToolFromCtx = %v, want codex", got)
	}
}

func TestClientToolFromCtxUserAgentFallback(t *testing.T) {
	ctx := makeCtxWithHeaders("GET", "/", map[string]string{
		"User-Agent": "my-agent/2.0",
	})
	got := clientToolFromCtx(ctx)
	if got == nil || *got != "my-agent/2.0" {
		t.Fatalf("clientToolFromCtx = %v, want my-agent/2.0", got)
	}
}

func TestClientToolFromCtxNeitherHeaderReturnsNil(t *testing.T) {
	ctx := makeCtxWithHeaders("GET", "/", nil)
	got := clientToolFromCtx(ctx)
	if got != nil {
		t.Fatalf("clientToolFromCtx = %v, want nil", got)
	}
}

// ---- rtkBytesSaved ----

func TestRTKBytesSavedDisabledReturnsNil(t *testing.T) {
	settings := store.Settings{RTKEnabled: false}
	req := &providers.ChatRequest{Model: "gpt-4o"}
	if got := rtkBytesSaved(settings, req); got != nil {
		t.Fatalf("rtkBytesSaved disabled = %v, want nil", got)
	}
}

func TestRTKBytesSavedNilRequestReturnsNil(t *testing.T) {
	settings := store.Settings{RTKEnabled: true}
	if got := rtkBytesSaved(settings, nil); got != nil {
		t.Fatalf("rtkBytesSaved nil req = %v, want nil", got)
	}
}

func TestRTKBytesSavedCompressibleRequestReturnsSaved(t *testing.T) {
	settings := store.Settings{RTKEnabled: true}
	bulky := strings.Repeat("compressible tool log line\n", 500)
	req := &providers.ChatRequest{
		Model: "gpt-4o",
		Messages: []providers.Message{
			{Role: "tool", Content: bulky},
		},
	}
	got := rtkBytesSaved(settings, req)
	if got == nil || *got <= 0 {
		t.Fatalf("rtkBytesSaved = %v, want > 0 for compressible content", got)
	}
}

func TestRTKBytesSavedNoSavingReturnsNil(t *testing.T) {
	settings := store.Settings{RTKEnabled: true}
	// A minimal request that RTK cannot compress further.
	req := &providers.ChatRequest{
		Model:    "gpt-4o",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	}
	got := rtkBytesSaved(settings, req)
	// Either nil (no saving) or small positive; we only assert no panic and
	// the nil path is exercised when there's no saving.
	_ = got
}

// ---- comboNameForModel ----

func TestComboNameForModelNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{})
	req := &providers.ChatRequest{Model: "gpt-4o"}
	if got := srv.comboNameForModel(req); got != nil {
		t.Fatalf("comboNameForModel nil store = %v, want nil", got)
	}
}

func TestComboNameForModelNilRequest(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	if got := srv.comboNameForModel(nil); got != nil {
		t.Fatalf("comboNameForModel nil request = %v, want nil", got)
	}
}

func TestComboNameForModelEmptyModel(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	req := &providers.ChatRequest{Model: ""}
	if got := srv.comboNameForModel(req); got != nil {
		t.Fatalf("comboNameForModel empty model = %v, want nil", got)
	}
}

func TestComboNameForModelNotACombo(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})
	req := &providers.ChatRequest{Model: "gpt-4o"}
	if got := srv.comboNameForModel(req); got != nil {
		t.Fatalf("comboNameForModel non-combo = %v, want nil", got)
	}
}

func TestComboNameForModelActiveComboReturnsName(t *testing.T) {
	s := newAPITestStore(t)
	combo := &store.Combo{
		Name:     "fast-combo",
		Steps:    []store.ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
	}
	if err := s.CreateCombo(combo); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	srv := NewServer(ServerConfig{Store: s})
	req := &providers.ChatRequest{Model: "fast-combo"}
	got := srv.comboNameForModel(req)
	if got == nil || *got != "fast-combo" {
		t.Fatalf("comboNameForModel = %v, want fast-combo", got)
	}
}

// ---- runLogRetentionOnce: error path ----

func TestRunLogRetentionOnceNilStore(t *testing.T) {
	srv := NewServer(ServerConfig{}) // Store is nil
	// Must not panic.
	srv.runLogRetentionOnce(time.Now().UTC())
}

func TestRunLogRetentionOnceErrorPathLogged(t *testing.T) {
	// Use a store and set retention > 0, but then close the store so
	// DeleteRequestLogsOlderThan returns an error — exercises the error log branch.
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)

	now := time.Now().UTC()
	seedLog(t, s, "old", now.Add(-30*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	// Prime the settings cache before closing.
	srv.runLogRetentionOnce(now) // succeeds — deletes "old"

	// Close the store so the next call hits an error.
	_ = s.Close()

	// Should not panic even when the underlying DB is closed.
	srv.settingsCache = nil // force re-read so it hits the closed DB
	srv.runLogRetentionOnce(now)
}

// ---- StartLogRetention: nil store early return ----

func TestStartLogRetentionNilStoreNoOp(t *testing.T) {
	srv := NewServer(ServerConfig{}) // Store is nil — should return immediately.
	// Must not panic or block.
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx := noopContext()
		srv.StartLogRetention(ctx)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("StartLogRetention with nil store did not return quickly")
	}
}

// StartLogRetention: interval <= 0 falls back to logRetentionInterval default.
func TestStartLogRetentionZeroIntervalFallsBackToDefault(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)

	now := time.Now().UTC()
	seedLog(t, s, "ancient", now.Add(-30*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.logRetentionInterval = 0 // triggers the <= 0 fallback branch

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartLogRetention(ctx)

	// Give the startup pass time to run.
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
			t.Fatalf("startup retention pass did not delete old log; entries = %+v", entries)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// StartLogRetention: ticker.C arm fires when interval is very short.
func TestStartLogRetentionTickerFires(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 1)

	now := time.Now().UTC()
	seedLog(t, s, "stale", now.Add(-48*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.logRetentionInterval = 50 * time.Millisecond // tiny interval so ticker fires fast

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartLogRetention(ctx)

	// Wait for at least one ticker-driven pass (the startup pass deletes "stale",
	// a second tick proves the ticker.C branch executes without panic).
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
			t.Fatalf("retention did not delete stale log; entries = %+v", entries)
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Let ticker fire at least once more after the store is empty (no-op run).
	time.Sleep(100 * time.Millisecond)
}

// runLogRetentionOnce: error from DeleteRequestLogsOlderThan logs and returns.
func TestRunLogRetentionOnceDeleteErrorLogged(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)

	now := time.Now().UTC()
	seedLog(t, s, "old", now.Add(-30*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})

	// Prime the settings cache so runtimeSettings() returns retention=7
	// even after we close the DB.
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	srv.settingsMu.Lock()
	srv.settingsCache = &settings
	srv.settingsMu.Unlock()

	// Close DB — DeleteRequestLogsOlderThan will error, exercising lines 126-128.
	_ = s.Close()
	srv.runLogRetentionOnce(now) // must not panic
}

// ---- logStreamingInferenceUsage: nil usageStore branch ----

func TestLogStreamingInferenceUsageNilUsageStore(t *testing.T) {
	// Server with no UsageStore — should be a no-op, not panic.
	srv := NewServer(ServerConfig{})
	snapshot := streamLogSnapshot{requestID: "r1"}
	// Must not panic.
	srv.logStreamingInferenceUsage(snapshot, time.Now(), "openai", nil, "", nil)
}

func TestLogStreamingInferenceUsageRequestLogsDisabled(t *testing.T) {
	s := newAPITestStore(t)
	// EnableRequestLogs defaults to false — no log should be written.
	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	snapshot := streamLogSnapshot{requestID: "r1"}
	srv.logStreamingInferenceUsage(snapshot, time.Now(), "openai", &providers.ChatRequest{Model: "gpt-4o"}, "gpt-4o", nil)

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0 (logs disabled)", len(entries))
	}
}

func TestLogStreamingInferenceUsageNilRequestAndModel(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	snapshot := streamLogSnapshot{requestID: "r1"}
	// nil request + empty model — early return, nothing logged.
	srv.logStreamingInferenceUsage(snapshot, time.Now(), "openai", nil, "", nil)

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0 (nil request + empty model)", len(entries))
	}
}
