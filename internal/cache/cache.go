// Package cache provides a thread-safe TTL+LRU cache for proxy responses.
//
// Keys are derived from a canonicalized request body so that semantically
// identical requests (e.g. JSON objects with reordered fields) map to the same
// entry. TTL is evaluated against an injected clock for deterministic tests.
package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// entry is the value stored in the LRU list. value is treated as immutable once
// inserted; callers receive copies via Get.
type entry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

// Cache is a bounded, thread-safe TTL+LRU response cache.
type Cache struct {
	mu         sync.Mutex
	maxEntries int
	ttl        time.Duration
	now        func() time.Time
	ll         *list.List               // front = most recently used
	items      map[string]*list.Element // key -> *list.Element holding *entry
}

// NewCache builds a Cache holding at most maxEntries entries, each living for
// ttl. now supplies the clock used for expiry; pass time.Now in production and a
// controllable function in tests. A non-positive maxEntries yields a cache that
// stores nothing.
func NewCache(maxEntries int, ttl time.Duration, now func() time.Time) *Cache {
	if now == nil {
		now = time.Now
	}
	return &Cache{
		maxEntries: maxEntries,
		ttl:        ttl,
		now:        now,
		ll:         list.New(),
		items:      make(map[string]*list.Element),
	}
}

// Key returns a stable sha256 hex digest of the model plus the canonicalized
// request body. Bodies that are valid JSON are normalized so field ordering does
// not affect the key; invalid JSON is hashed verbatim.
func (c *Cache) Key(model string, body []byte) string {
	h := sha256.New()
	h.Write([]byte(model))
	h.Write([]byte{0}) // separator so model+body cannot collide across the boundary
	h.Write(canonicalJSON(body))
	return hex.EncodeToString(h.Sum(nil))
}

// canonicalJSON returns a deterministic encoding of body. When body is valid
// JSON it is unmarshaled and re-marshaled (encoding/json sorts map keys), giving
// order-independent output. Otherwise the raw bytes are returned unchanged.
func canonicalJSON(body []byte) []byte {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return body
	}
	canonical, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return canonical
}

// Get returns a copy of the cached value for key when present and unexpired.
// Expired entries are purged on access. A hit marks the entry most-recently-used.
func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	ent := el.Value.(*entry)
	if !c.now().Before(ent.expiresAt) {
		c.removeElement(el)
		return nil, false
	}
	c.ll.MoveToFront(el)

	out := make([]byte, len(ent.value))
	copy(out, ent.value)
	return out, true
}

// Set stores a copy of value under key with the configured TTL, evicting the
// least-recently-used entry when the cache is over capacity.
func (c *Cache) Set(key string, value []byte) {
	if c.maxEntries <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := make([]byte, len(value))
	copy(stored, value)
	expiresAt := c.now().Add(c.ttl)

	if el, ok := c.items[key]; ok {
		ent := el.Value.(*entry)
		ent.value = stored
		ent.expiresAt = expiresAt
		c.ll.MoveToFront(el)
		return
	}

	el := c.ll.PushFront(&entry{key: key, value: stored, expiresAt: expiresAt})
	c.items[key] = el

	for c.ll.Len() > c.maxEntries {
		if back := c.ll.Back(); back != nil {
			c.removeElement(back)
		}
	}
}

// removeElement drops el from both the list and the index. Caller holds c.mu.
func (c *Cache) removeElement(el *list.Element) {
	c.ll.Remove(el)
	delete(c.items, el.Value.(*entry).key)
}
