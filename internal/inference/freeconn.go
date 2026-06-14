package inference

import (
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/store"
)

// freeConnSentinelID is the sentinel ID for an injected no-auth virtual
// connection (PAR-ROUTE-039). It mirrors 9router's id:"noauth" sentinel, which
// markAccountUnavailable skips (auth.js:204) — a synthetic connection has no
// stored state to cool down.
const freeConnSentinelID = "noauth"

// isFreeProvider reports whether the provider requires no auth (a "free"
// provider, e.g. opencode), sourced from the catalog's NoAuth flag
// (ESC-FREE-SET). Mirrors providers.js:14 / auth.js:36-53.
func isFreeProvider(providerID string) bool {
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return false
	}
	return cfg.NoAuth
}

// syntheticFreeConnection builds a no-auth virtual connection for a free
// provider (PAR-ROUTE-039). The connection carries the provider ID, a sentinel
// ID+name, and an empty secret (the free provider needs no credential). It is
// returned by SelectConnection when a free provider has no real eligible
// connection so a request still routes (auth.js:36-53).
func syntheticFreeConnection(providerID string) *store.Connection {
	return &store.Connection{
		ID:         freeConnSentinelID,
		ProviderID: providerID,
		Name:       freeConnSentinelID,
		Kind:       "api_key",
		Secret:     "",
	}
}
