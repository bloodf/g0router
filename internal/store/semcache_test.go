package store

import (
	"errors"
	"testing"
)

// fixedTime is a deterministic clock ISO string used to advance expiry without
// real time (hermetic, D9).
const (
	semNow     = "2026-06-15T12:00:00Z"
	semPast    = "2026-06-15T11:00:00Z"
	semFuture  = "2026-06-15T13:00:00Z"
)

func TestSemanticCacheInsertGetRoundTrip(t *testing.T) {
	st := newTestStore(t)
	entry := SemanticCacheEntry{
		CacheKey:      "key-abc",
		EmbeddingJSON: "[]",
		Model:         "gpt-4",
		ResponseJSON:  `{"id":"resp-1"}`,
		ExpiresAt:     semFuture,
	}
	if err := st.InsertSemanticCacheEntry(entry); err != nil {
		t.Fatalf("InsertSemanticCacheEntry: %v", err)
	}

	got, err := st.GetSemanticCacheByKey("key-abc", semNow)
	if err != nil {
		t.Fatalf("GetSemanticCacheByKey: %v", err)
	}
	if got.CacheKey != "key-abc" || got.Model != "gpt-4" || got.ResponseJSON != `{"id":"resp-1"}` {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	if got.ID == 0 {
		t.Fatal("expected non-zero id")
	}
}

func TestSemanticCacheGetMissing(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.GetSemanticCacheByKey("nope", semNow); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetSemanticCacheByKey missing err = %v, want ErrNotFound", err)
	}
}

func TestSemanticCacheExpiredFilteredOut(t *testing.T) {
	st := newTestStore(t)
	entry := SemanticCacheEntry{
		CacheKey:      "key-exp",
		EmbeddingJSON: "[]",
		Model:         "gpt-4",
		ResponseJSON:  `{"id":"old"}`,
		ExpiresAt:     semPast, // already expired relative to semNow
	}
	if err := st.InsertSemanticCacheEntry(entry); err != nil {
		t.Fatalf("InsertSemanticCacheEntry: %v", err)
	}
	if _, err := st.GetSemanticCacheByKey("key-exp", semNow); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired entry served: err = %v, want ErrNotFound", err)
	}
}

func TestSemanticCacheNoExpirySalwaysServed(t *testing.T) {
	st := newTestStore(t)
	entry := SemanticCacheEntry{
		CacheKey:      "key-noexp",
		EmbeddingJSON: "[]",
		Model:         "gpt-4",
		ResponseJSON:  `{"id":"perm"}`,
		// ExpiresAt empty → NULL → never expires.
	}
	if err := st.InsertSemanticCacheEntry(entry); err != nil {
		t.Fatalf("InsertSemanticCacheEntry: %v", err)
	}
	got, err := st.GetSemanticCacheByKey("key-noexp", semNow)
	if err != nil {
		t.Fatalf("GetSemanticCacheByKey (no-expiry): %v", err)
	}
	if got.ResponseJSON != `{"id":"perm"}` {
		t.Fatalf("no-expiry entry mismatch: %+v", got)
	}
}

func TestSemanticCacheIncrementHit(t *testing.T) {
	st := newTestStore(t)
	entry := SemanticCacheEntry{
		CacheKey:      "key-hit",
		EmbeddingJSON: "[]",
		Model:         "gpt-4",
		ResponseJSON:  `{"id":"hit"}`,
		ExpiresAt:     semFuture,
	}
	if err := st.InsertSemanticCacheEntry(entry); err != nil {
		t.Fatalf("InsertSemanticCacheEntry: %v", err)
	}
	got, err := st.GetSemanticCacheByKey("key-hit", semNow)
	if err != nil {
		t.Fatalf("GetSemanticCacheByKey: %v", err)
	}
	if err := st.IncrementSemanticCacheHit(got.ID); err != nil {
		t.Fatalf("IncrementSemanticCacheHit: %v", err)
	}
	if err := st.IncrementSemanticCacheHit(got.ID); err != nil {
		t.Fatalf("IncrementSemanticCacheHit (2): %v", err)
	}
	entries, err := st.ListSemanticCacheEntries()
	if err != nil {
		t.Fatalf("ListSemanticCacheEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0].HitCount != 2 {
		t.Fatalf("HitCount = %d, want 2", entries[0].HitCount)
	}
}

func TestSemanticCachePurgeExpired(t *testing.T) {
	st := newTestStore(t)
	must := func(e SemanticCacheEntry) {
		if err := st.InsertSemanticCacheEntry(e); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	must(SemanticCacheEntry{CacheKey: "fresh", EmbeddingJSON: "[]", Model: "m", ResponseJSON: "{}", ExpiresAt: semFuture})
	must(SemanticCacheEntry{CacheKey: "stale", EmbeddingJSON: "[]", Model: "m", ResponseJSON: "{}", ExpiresAt: semPast})
	must(SemanticCacheEntry{CacheKey: "perm", EmbeddingJSON: "[]", Model: "m", ResponseJSON: "{}"})

	n, err := st.PurgeExpiredSemanticCache(semNow)
	if err != nil {
		t.Fatalf("PurgeExpiredSemanticCache: %v", err)
	}
	if n != 1 {
		t.Fatalf("purged = %d, want 1 (only the stale row)", n)
	}
	entries, err := st.ListSemanticCacheEntries()
	if err != nil {
		t.Fatalf("ListSemanticCacheEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("remaining = %d, want 2 (fresh + perm)", len(entries))
	}
}

func TestSemanticCacheStats(t *testing.T) {
	st := newTestStore(t)
	if err := st.InsertSemanticCacheEntry(SemanticCacheEntry{CacheKey: "a", EmbeddingJSON: "[]", Model: "m", ResponseJSON: "{}", ExpiresAt: semFuture}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got, err := st.GetSemanticCacheByKey("a", semNow)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if err := st.IncrementSemanticCacheHit(got.ID); err != nil {
		t.Fatalf("hit: %v", err)
	}
	stats, err := st.SemanticCacheStats()
	if err != nil {
		t.Fatalf("SemanticCacheStats: %v", err)
	}
	if stats.Entries != 1 {
		t.Fatalf("stats.Entries = %d, want 1", stats.Entries)
	}
	if stats.TotalHits != 1 {
		t.Fatalf("stats.TotalHits = %d, want 1", stats.TotalHits)
	}
}

func TestSemanticCacheClear(t *testing.T) {
	st := newTestStore(t)
	if err := st.InsertSemanticCacheEntry(SemanticCacheEntry{CacheKey: "a", EmbeddingJSON: "[]", Model: "m", ResponseJSON: "{}", ExpiresAt: semFuture}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := st.ClearSemanticCache(); err != nil {
		t.Fatalf("ClearSemanticCache: %v", err)
	}
	entries, err := st.ListSemanticCacheEntries()
	if err != nil {
		t.Fatalf("ListSemanticCacheEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("after clear len = %d, want 0", len(entries))
	}
}
