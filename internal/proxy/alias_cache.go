package proxy

import (
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

const defaultAliasCacheTTL = 5 * time.Minute

type aliasCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	entries map[string]aliasCacheEntry
}

type aliasCacheEntry struct {
	alias     store.ModelAlias
	expiresAt time.Time
}

func newAliasCache(ttl time.Duration) *aliasCache {
	return &aliasCache{
		ttl:     ttl,
		entries: make(map[string]aliasCacheEntry),
	}
}

func (c *aliasCache) get(alias string, now time.Time) (store.ModelAlias, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[alias]
	if !ok {
		return store.ModelAlias{}, false
	}
	if !now.Before(entry.expiresAt) {
		delete(c.entries, alias)
		return store.ModelAlias{}, false
	}
	return entry.alias, true
}

func (c *aliasCache) set(alias string, modelAlias store.ModelAlias, now time.Time) {
	c.mu.Lock()
	c.entries[alias] = aliasCacheEntry{
		alias:     modelAlias,
		expiresAt: now.Add(c.ttl),
	}
	c.mu.Unlock()
}
