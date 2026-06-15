package semcache

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeRepo is an in-memory CacheRepository for hermetic tests (D9): no SQLite,
// no network. It records calls so the cosine/embedder absence and the
// lazy-purge behavior can be asserted.
type fakeRepo struct {
	byKey      map[string]*CachedEntry
	insertErr  error
	getErr     error
	hits       map[int64]int
	purgeCalls int
	nextID     int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byKey: map[string]*CachedEntry{}, hits: map[int64]int{}}
}

func (r *fakeRepo) GetByKey(_ context.Context, cacheKey, nowISO string) (*CachedEntry, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	e, ok := r.byKey[cacheKey]
	if !ok {
		return nil, ErrCacheMiss
	}
	// Honor expiry against the injected clock.
	if e.ExpiresAt != "" && e.ExpiresAt <= nowISO {
		return nil, ErrCacheMiss
	}
	return e, nil
}

func (r *fakeRepo) Insert(_ context.Context, e CachedEntry) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	r.nextID++
	stored := e
	stored.ID = r.nextID
	r.byKey[e.CacheKey] = &stored
	return nil
}

func (r *fakeRepo) IncrementHit(_ context.Context, id int64) error {
	r.hits[id]++
	return nil
}

func (r *fakeRepo) PurgeExpired(_ context.Context, _ string) (int64, error) {
	r.purgeCalls++
	return 0, nil
}

// fakeSettings is a hermetic settings reader.
type fakeSettings struct {
	values map[string]string
}

func (s fakeSettings) GetSetting(key string) (string, error) {
	v, ok := s.values[key]
	if !ok {
		return "", ErrSettingMissing
	}
	return v, nil
}

// fixedClock returns a constant time so TTL/expiry are deterministic (D7/D9).
func fixedClock(iso string) func() time.Time {
	t, _ := time.Parse(time.RFC3339, iso)
	return func() time.Time { return t }
}

const (
	clockNow = "2026-06-15T12:00:00Z"
)

func newTestCache(repo CacheRepository, settings SettingsReader) *Cache {
	return NewCache(repo, settings, fixedClock(clockNow))
}

// TestLookupMiss verifies a cold lookup reports no hit and returns no bytes.
func TestLookupMiss(t *testing.T) {
	repo := newFakeRepo()
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})

	resp, hit, err := c.Lookup(context.Background(), "gpt-4", "hello")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if hit {
		t.Fatal("Lookup reported hit on empty cache")
	}
	if resp != nil {
		t.Fatalf("Lookup returned bytes on miss: %q", resp)
	}
}

// TestStoreThenLookupHit verifies the write-through then read-through path: a
// stored response is returned verbatim on a subsequent identical lookup, and
// hit_count is incremented (the short-circuit datum the chat hook depends on).
func TestStoreThenLookupHit(t *testing.T) {
	repo := newFakeRepo()
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})

	want := []byte(`{"id":"resp-1","choices":[]}`)
	if err := c.Store(context.Background(), "gpt-4", "hello", want); err != nil {
		t.Fatalf("Store: %v", err)
	}

	resp, hit, err := c.Lookup(context.Background(), "gpt-4", "hello")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if !hit {
		t.Fatal("Lookup did not hit after Store")
	}
	if string(resp) != string(want) {
		t.Fatalf("Lookup bytes = %q, want %q", resp, want)
	}
	// hit_count incremented for the matched row id (id 1 from fakeRepo).
	if repo.hits[1] != 1 {
		t.Fatalf("hit_count increment = %d, want 1", repo.hits[1])
	}
}

// TestStoreWritesEmptyEmbedding verifies the hash-only cache stores
// embedding_json="[]" — forward-compatibility data, never executed (D2).
func TestStoreWritesEmptyEmbedding(t *testing.T) {
	repo := newFakeRepo()
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})
	if err := c.Store(context.Background(), "gpt-4", "hello", []byte("{}")); err != nil {
		t.Fatalf("Store: %v", err)
	}
	key := CacheKey("gpt-4", "hello")
	got := repo.byKey[key]
	if got == nil {
		t.Fatal("entry not stored under exact key")
	}
	if got.EmbeddingJSON != "[]" {
		t.Fatalf("EmbeddingJSON = %q, want []", got.EmbeddingJSON)
	}
}

// TestStoreSetsExpiryFromTTL verifies cache_ttl_seconds is read and expires_at
// is computed from the injected clock (D5/D7): now + ttl.
func TestStoreSetsExpiryFromTTL(t *testing.T) {
	repo := newFakeRepo()
	c := newTestCache(repo, fakeSettings{values: map[string]string{"cache_ttl_seconds": "3600"}})
	if err := c.Store(context.Background(), "gpt-4", "hello", []byte("{}")); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got := repo.byKey[CacheKey("gpt-4", "hello")]
	if got == nil {
		t.Fatal("entry not stored")
	}
	want := "2026-06-15T13:00:00Z" // clockNow + 3600s
	if got.ExpiresAt != want {
		t.Fatalf("ExpiresAt = %q, want %q", got.ExpiresAt, want)
	}
}

// TestStoreNoTTLNoExpiry verifies an unset/zero TTL stores no expiry (the row
// never expires) — a missing setting must not error (D5).
func TestStoreNoTTLNoExpiry(t *testing.T) {
	repo := newFakeRepo()
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})
	if err := c.Store(context.Background(), "gpt-4", "hello", []byte("{}")); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got := repo.byKey[CacheKey("gpt-4", "hello")]
	if got == nil {
		t.Fatal("entry not stored")
	}
	if got.ExpiresAt != "" {
		t.Fatalf("ExpiresAt = %q, want empty (no expiry)", got.ExpiresAt)
	}
}

// TestLookupExpiredNotServed verifies an entry past the injected clock is not
// served (the repo expiry filter excludes it) — reported as a miss.
func TestLookupExpiredNotServed(t *testing.T) {
	repo := newFakeRepo()
	// Seed an already-expired row directly.
	repo.byKey[CacheKey("gpt-4", "hello")] = &CachedEntry{
		ID: 1, CacheKey: CacheKey("gpt-4", "hello"), Model: "gpt-4",
		ResponseJSON: "{}", ExpiresAt: "2026-06-15T11:00:00Z", // before clockNow
	}
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})

	_, hit, err := c.Lookup(context.Background(), "gpt-4", "hello")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if hit {
		t.Fatal("expired entry was served")
	}
}

// TestLookupPurgesLazily verifies the read path opportunistically purges expired
// rows (lazy purge, D7) — there is no background reaper.
func TestLookupPurgesLazily(t *testing.T) {
	repo := newFakeRepo()
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})
	if _, _, err := c.Lookup(context.Background(), "gpt-4", "hello"); err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if repo.purgeCalls != 1 {
		t.Fatalf("PurgeExpired calls = %d, want 1 (lazy purge on read)", repo.purgeCalls)
	}
}

// TestLookupRepoErrorPropagates verifies a real repo error is surfaced (not
// silently swallowed as a miss).
func TestLookupRepoErrorPropagates(t *testing.T) {
	repo := newFakeRepo()
	repo.getErr = errors.New("db boom")
	c := newTestCache(repo, fakeSettings{values: map[string]string{}})
	if _, _, err := c.Lookup(context.Background(), "gpt-4", "hello"); err == nil {
		t.Fatal("Lookup swallowed a repo error")
	}
}
