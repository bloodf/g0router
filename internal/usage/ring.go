package usage

import (
	"fmt"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// Ring is a fixed-capacity buffer of recent request log entries.
// It lazy-initializes once from a store lister and appends new entries at the
// tail, dropping the oldest entries when the cap is exceeded.
type Ring struct {
	cap       int
	mu        sync.Mutex
	initOnce  sync.Once
	initErr   error
	items     []*store.RequestLogEntry
}

// NewRing creates a Ring with the given capacity.
func NewRing(capacity int) *Ring {
	return &Ring{cap: capacity}
}

// Init lazy-loads the ring from lister exactly once. Subsequent calls are no-ops.
func (r *Ring) Init(lister func() ([]*store.RequestLogEntry, error)) error {
	r.initOnce.Do(func() {
		items, err := lister()
		if err != nil {
			r.initErr = fmt.Errorf("init ring: %w", err)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		// The lister returns newest-first; reverse to oldest-first so pushes
		// append at the tail in chronological order.
		for i := len(items) - 1; i >= 0; i-- {
			r.items = append(r.items, items[i])
		}
		// Cap enforcement: keep only the newest items.
		if len(r.items) > r.cap {
			r.items = r.items[len(r.items)-r.cap:]
		}
	})
	return r.initErr
}

// Push appends an entry to the tail and truncates to the configured cap.
func (r *Ring) Push(e *store.RequestLogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append(r.items, e)
	if len(r.items) > r.cap {
		r.items = r.items[len(r.items)-r.cap:]
	}
}

// Snapshot returns a copy of the current ring contents.
func (r *Ring) Snapshot() []*store.RequestLogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*store.RequestLogEntry, len(r.items))
	copy(out, r.items)
	return out
}

// ConnInfo is the raw shape needed to build a connection-name map.
type ConnInfo struct {
	ID    string
	Name  string
	Email string
}

// ConnNameCache caches a connection-id → display-name map with a TTL.
// The display name follows the fallback chain: name → email → id.
type ConnNameCache struct {
	lister func() ([]ConnInfo, error)
	ttl    time.Duration
	clock  func() time.Time

	mu    sync.Mutex
	cache map[string]string
	ts    time.Time
}

// NewConnNameCache creates a connection-name cache.
func NewConnNameCache(lister func() ([]ConnInfo, error), ttl time.Duration, clock func() time.Time) *ConnNameCache {
	return &ConnNameCache{
		lister: lister,
		ttl:    ttl,
		clock:  clock,
	}
}

// Get returns the cached id→name map, refreshing from the lister after ttl.
// If the lister fails, the previous cached map is returned (empty if none).
func (c *ConnNameCache) Get() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.clock()
	if c.cache != nil && now.Sub(c.ts) < c.ttl {
		return copyConnMap(c.cache)
	}

	infos, err := c.lister()
	if err != nil {
		return copyConnMap(c.cache)
	}

	m := make(map[string]string, len(infos))
	for _, info := range infos {
		name := info.Name
		if name == "" {
			name = info.Email
		}
		if name == "" {
			name = info.ID
		}
		m[info.ID] = name
	}

	c.cache = m
	c.ts = now
	return copyConnMap(m)
}

func copyConnMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
