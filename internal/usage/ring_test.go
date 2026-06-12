package usage

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

func TestRingInitOnceFromStore(t *testing.T) {
	calls := 0
	lister := func(limit int) ([]*store.RequestLogEntry, error) {
		calls++
		// Lister returns newest-first (mirrors ORDER BY id DESC).
		return []*store.RequestLogEntry{
			{Timestamp: "2026-06-12T10:01:00Z", Model: "b"},
			{Timestamp: "2026-06-12T10:00:00Z", Model: "a"},
		}, nil
	}

	ring := NewRing(3)
	if err := ring.Init(func() ([]*store.RequestLogEntry, error) { return lister(3) }); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := ring.Init(func() ([]*store.RequestLogEntry, error) { return lister(3) }); err != nil {
		t.Fatalf("Init second: %v", err)
	}
	if calls != 1 {
		t.Errorf("lister calls = %d, want 1", calls)
	}

	snap := ring.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("snapshot len = %d, want 2", len(snap))
	}
	// Reversed to oldest-first.
	if snap[0].Model != "a" || snap[1].Model != "b" {
		t.Errorf("order = %v, want [a b]", []string{snap[0].Model, snap[1].Model})
	}

	ring.Push(&store.RequestLogEntry{Timestamp: "2026-06-12T10:02:00Z", Model: "c"})
	snap = ring.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("snapshot len after push = %d, want 3", len(snap))
	}
	if snap[2].Model != "c" {
		t.Errorf("last model = %q, want c", snap[2].Model)
	}

	// Cap enforcement: drop oldest.
	ring.Push(&store.RequestLogEntry{Timestamp: "2026-06-12T10:03:00Z", Model: "d"})
	snap = ring.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("snapshot len after cap push = %d, want 3", len(snap))
	}
	if snap[0].Model != "b" || snap[1].Model != "c" || snap[2].Model != "d" {
		t.Errorf("cap order = %v, want [b c d]", []string{snap[0].Model, snap[1].Model, snap[2].Model})
	}
}

func TestRingInitEnforcesCap(t *testing.T) {
	lister := func(limit int) ([]*store.RequestLogEntry, error) {
		// Newest-first order, mirroring ORDER BY id DESC.
		return []*store.RequestLogEntry{
			{Timestamp: "2026-06-12T10:04:00Z", Model: "e"},
			{Timestamp: "2026-06-12T10:03:00Z", Model: "d"},
			{Timestamp: "2026-06-12T10:02:00Z", Model: "c"},
			{Timestamp: "2026-06-12T10:01:00Z", Model: "b"},
			{Timestamp: "2026-06-12T10:00:00Z", Model: "a"},
		}, nil
	}

	ring := NewRing(3)
	if err := ring.Init(func() ([]*store.RequestLogEntry, error) { return lister(3) }); err != nil {
		t.Fatalf("Init: %v", err)
	}

	snap := ring.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("snapshot len = %d, want 3", len(snap))
	}
	// After reversing to oldest-first and truncating to cap, keep the 3 newest.
	if snap[0].Model != "c" || snap[1].Model != "d" || snap[2].Model != "e" {
		t.Errorf("models = %v, want [c d e]", []string{snap[0].Model, snap[1].Model, snap[2].Model})
	}
}

func TestRingInitPushConcurrent(t *testing.T) {
	lister := func() ([]*store.RequestLogEntry, error) {
		// Sleep long enough for the concurrent Push to run while Init is in
		// progress; the subsequent append loop inside initOnce.Do races with
		// Push unless Init holds r.mu.
		time.Sleep(100 * time.Millisecond)
		items := make([]*store.RequestLogEntry, 10000)
		for i := range items {
			items[i] = &store.RequestLogEntry{Timestamp: fmt.Sprintf("%d", i), Model: "x"}
		}
		return items, nil
	}

	ring := NewRing(10)
	go func() {
		if err := ring.Init(lister); err != nil {
			t.Errorf("Init: %v", err)
		}
	}()

	// Let Init enter the lister sleep before pushing.
	time.Sleep(50 * time.Millisecond)
	ring.Push(&store.RequestLogEntry{Timestamp: "now", Model: "p"})

	// Wait for Init to finish its append loop.
	time.Sleep(200 * time.Millisecond)
}

func TestConnNameCacheTTL(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	calls := 0
	failNext := false
	lister := func() ([]ConnInfo, error) {
		calls++
		if failNext {
			return nil, errors.New("boom")
		}
		return []ConnInfo{
			{ID: "c1", Name: "Alice", Email: "alice@example.com"},
			{ID: "c2", Name: "", Email: "bob@example.com"},
			{ID: "c3", Name: "", Email: ""},
		}, nil
	}

	cache := NewConnNameCache(lister, 30*time.Second, func() time.Time { return now })
	m := cache.Get()
	if calls != 1 {
		t.Errorf("lister calls = %d, want 1", calls)
	}
	if m["c1"] != "Alice" {
		t.Errorf("c1 = %q, want Alice", m["c1"])
	}
	if m["c2"] != "bob@example.com" {
		t.Errorf("c2 = %q, want bob@example.com", m["c2"])
	}
	if m["c3"] != "c3" {
		t.Errorf("c3 = %q, want c3", m["c3"])
	}

	// Within TTL: no re-list.
	now = now.Add(29 * time.Second)
	m = cache.Get()
	if calls != 1 {
		t.Errorf("lister calls within TTL = %d, want 1", calls)
	}
	if m["c1"] != "Alice" {
		t.Errorf("cached c1 = %q, want Alice", m["c1"])
	}

	// After TTL: re-list.
	now = now.Add(2 * time.Second)
	m = cache.Get()
	if calls != 2 {
		t.Errorf("lister calls after TTL = %d, want 2", calls)
	}

	// After TTL with a failing lister: return the prior cached map, no panic.
	failNext = true
	now = now.Add(31 * time.Second)
	m = cache.Get()
	if calls != 3 {
		t.Errorf("lister calls after TTL failure = %d, want 3", calls)
	}
	if m == nil {
		t.Fatalf("Get returned nil map on lister error, want prior map")
	}
	if m["c1"] != "Alice" {
		t.Errorf("stale c1 = %q, want Alice", m["c1"])
	}
}
