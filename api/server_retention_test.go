package api

import (
	"context"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func setRetentionDays(t *testing.T, s *store.Store, days int) {
	t.Helper()

	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.LogRetentionDays = days
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
}

func seedLog(t *testing.T, s *store.Store, requestID string, ts time.Time) {
	t.Helper()

	if err := s.LogRequest(&store.RequestLogEntry{
		RequestID: requestID,
		Timestamp: ts,
		Provider:  "openai",
		Model:     "gpt-4o",
		AuthType:  "noauth",
	}); err != nil {
		t.Fatalf("LogRequest: %v", err)
	}
}

func TestRunLogRetentionOnceDeletesOldKeepsNew(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	seedLog(t, s, "old", now.Add(-10*24*time.Hour))
	seedLog(t, s, "fresh", now.Add(-1*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.runLogRetentionOnce(now)

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestID != "fresh" {
		t.Fatalf("entries = %+v, want only fresh", entries)
	}
}

func TestRunLogRetentionOnceZeroKeepsEverything(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 0)

	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	seedLog(t, s, "ancient", now.Add(-1000*24*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.runLogRetentionOnce(now)

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1 (retention 0 keeps forever)", len(entries))
	}
}

func TestStartLogRetentionRunsAtStartupAndStopsOnCancel(t *testing.T) {
	s := newAPITestStore(t)
	setRetentionDays(t, s, 7)

	now := time.Now().UTC()
	seedLog(t, s, "old", now.Add(-30*24*time.Hour))
	seedLog(t, s, "fresh", now.Add(-1*time.Hour))

	srv := NewServer(ServerConfig{Store: s, UsageStore: s})
	srv.logRetentionInterval = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartLogRetention(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for {
		entries, err := s.GetUsage(store.UsageFilter{})
		if err != nil {
			t.Fatalf("GetUsage: %v", err)
		}
		if len(entries) == 1 && entries[0].RequestID == "fresh" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("startup retention pass did not delete old log; entries = %+v", entries)
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
}
