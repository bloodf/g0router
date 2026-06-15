package semcache

import (
	"context"
	"errors"
	"strconv"
	"time"
)

// ErrCacheMiss is returned by a CacheRepository when no live entry exists for a
// key. The Cache translates it into a (nil, false, nil) miss.
var ErrCacheMiss = errors.New("semcache: cache miss")

// ErrSettingMissing is returned by a SettingsReader when a key is unset. The
// Cache treats a missing cache_ttl_seconds as "no expiry".
var ErrSettingMissing = errors.New("semcache: setting not found")

// emptyEmbedding is written for every hash-only entry: the embedding_json column
// is retained for forward-compatibility with the deferred semantic half (D2);
// the hash-only cache never reads it.
const emptyEmbedding = "[]"

// CachedEntry is the persistence-facing view of a cache row. The domain owns
// this shape so the package does not import store (mirrors the api seams).
type CachedEntry struct {
	ID            int64
	CacheKey      string
	EmbeddingJSON string
	Model         string
	ResponseJSON  string
	ExpiresAt     string // ISO-8601; "" == no expiry
	HitCount      int64
}

// CacheRepository is the persistence seam for the exact-key cache. The store
// adapter implements it. There is NO candidate loader — the semantic half is
// deferred (D2).
type CacheRepository interface {
	GetByKey(ctx context.Context, cacheKey, nowISO string) (*CachedEntry, error)
	Insert(ctx context.Context, e CachedEntry) error
	IncrementHit(ctx context.Context, id int64) error
	PurgeExpired(ctx context.Context, nowISO string) (int64, error)
}

// SettingsReader reads global settings (cache_ttl_seconds). The store satisfies
// it via GetSetting.
type SettingsReader interface {
	GetSetting(key string) (string, error)
}

// cacheTTLSecondsKey is the settings key for the cache entry TTL (D5).
const cacheTTLSecondsKey = "cache_ttl_seconds"

// Cache is the exact-key-hash read-through/write-through cache. It has NO
// embedder field and NO semantic branch: Lookup does exactly one thing — an
// exact-key, non-expired SQLite read (D1). The clock is injected so TTL/expiry
// are hermetic (D7).
type Cache struct {
	repo     CacheRepository
	settings SettingsReader
	clock    func() time.Time
}

// NewCache constructs a Cache. clock defaults to time.Now when nil.
func NewCache(repo CacheRepository, settings SettingsReader, clock func() time.Time) *Cache {
	if clock == nil {
		clock = time.Now
	}
	return &Cache{repo: repo, settings: settings, clock: clock}
}

// Lookup returns the cached response bytes for (model, prompt) when a live
// (non-expired) exact-key entry exists. On a hit it increments hit_count and
// returns (bytes, true, nil); on a miss it returns (nil, false, nil). Expired
// rows are purged opportunistically on the read path (lazy purge, D7) — there is
// no background reaper.
func (c *Cache) Lookup(ctx context.Context, model, prompt string) ([]byte, bool, error) {
	nowISO := c.nowISO()

	// Lazy purge of expired rows on read (D7); best-effort — a purge failure
	// must not fail the lookup.
	_, _ = c.repo.PurgeExpired(ctx, nowISO)

	key := CacheKey(model, prompt)
	entry, err := c.repo.GetByKey(ctx, key, nowISO)
	if errors.Is(err, ErrCacheMiss) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if entry == nil {
		return nil, false, nil
	}

	// Best-effort hit accounting; a counter failure must not fail the hit.
	_ = c.repo.IncrementHit(ctx, entry.ID)
	return []byte(entry.ResponseJSON), true, nil
}

// Store write-through persists response for (model, prompt). The embedding_json
// column is written as "[]" (forward-compatibility data, D2). expires_at is
// computed from cache_ttl_seconds against the injected clock; an unset or
// non-positive TTL stores no expiry.
func (c *Cache) Store(ctx context.Context, model, prompt string, response []byte) error {
	entry := CachedEntry{
		CacheKey:      CacheKey(model, prompt),
		EmbeddingJSON: emptyEmbedding,
		Model:         model,
		ResponseJSON:  string(response),
		ExpiresAt:     c.expiresAt(),
	}
	return c.repo.Insert(ctx, entry)
}

// nowISO returns the injected clock as an RFC3339 UTC string.
func (c *Cache) nowISO() string {
	return c.clock().UTC().Format(time.RFC3339)
}

// expiresAt computes the expiry ISO string from cache_ttl_seconds, or "" when
// the TTL is unset or non-positive (no expiry).
func (c *Cache) expiresAt() string {
	ttl := c.ttlSeconds()
	if ttl <= 0 {
		return ""
	}
	return c.clock().UTC().Add(time.Duration(ttl) * time.Second).Format(time.RFC3339)
}

// ttlSeconds reads cache_ttl_seconds; a missing or malformed value yields 0.
func (c *Cache) ttlSeconds() int {
	raw, err := c.settings.GetSetting(cacheTTLSecondsKey)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}
