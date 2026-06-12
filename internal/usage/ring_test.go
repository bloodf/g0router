package usage

import (
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

func TestConnNameCacheTTL(t *testing.T) {
	now := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	calls := 0
	lister := func() ([]ConnInfo, error) {
		calls++
		return []ConnInfo{
			{ID: "c1", Name: "Alice", Email: "alice@example.com"},
			{ID: "c2", Name: "", Email: "bob@example.com"},
			{ID: "c3", Name: "", Email: ""},
		}, nil
	}

	cache := NewConnNameCache(lister, 30*time.Second, func() time.Time { return now })
	m, err := cache.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
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
	m, err = cache.Get()
	if err != nil {
		t.Fatalf("Get within TTL: %v", err)
	}
	if calls != 1 {
		t.Errorf("lister calls within TTL = %d, want 1", calls)
	}
	if m["c1"] != "Alice" {
		t.Errorf("cached c1 = %q, want Alice", m["c1"])
	}

	// After TTL: re-list.
	now = now.Add(2 * time.Second)
	m, err = cache.Get()
	if err != nil {
		t.Fatalf("Get after TTL: %v", err)
	}
	if calls != 2 {
		t.Errorf("lister calls after TTL = %d, want 2", calls)
	}
}
