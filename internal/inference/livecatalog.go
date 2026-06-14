package inference

import (
	"fmt"

	"github.com/bloodf/g0router/internal/store"
)

// ConnLister lists connections for the live-catalog resolver. It is the minimal
// read-only seam over the store; production wires *store.Store, tests a fake.
type ConnLister interface {
	ListConnections() ([]*store.Connection, error)
}

// LiveCatalogModel is one dynamic model returned by a live-catalog fetcher for a
// single connection (e.g. Kiro/Qoder per-account models). Provider is filled in
// by the resolver from the connection's provider ID.
type LiveCatalogModel struct {
	ID string
}

// LiveCatalogFetcher fetches the live, per-account dynamic model list for a
// single connection (PAR-ROUTE-056). It is the injectable network seam: in
// production it issues an HTTP request to the provider; in tests a deterministic
// fake returns fixed models with no network.
type LiveCatalogFetcher interface {
	Fetch(conn *store.Connection) ([]LiveCatalogModel, error)
}

// ResolvedLiveModel is a dynamic model resolved across all eligible connections,
// carrying the owning provider for /v1/models owned_by attribution.
type ResolvedLiveModel struct {
	ID       string
	Provider string
}

// LiveCatalogResolver resolves dynamic per-account models for providers that
// expose them (Kiro/Qoder). It lists connections, skips upstream connections
// (PAR-ROUTE-060), and fetches each remaining connection's dynamic models via an
// injectable fetcher. It performs no network I/O itself.
type LiveCatalogResolver struct {
	cl      ConnLister
	fetcher LiveCatalogFetcher
}

// NewLiveCatalogResolver creates a resolver over the given connection lister and
// fetcher. A nil fetcher disables live resolution (returns no models, no error),
// keeping behavior additive and backward-compatible.
func NewLiveCatalogResolver(cl ConnLister, fetcher LiveCatalogFetcher) *LiveCatalogResolver {
	return &LiveCatalogResolver{cl: cl, fetcher: fetcher}
}

// ResolveLiveModels returns the dynamic models across all non-upstream
// connections. A connection whose name carries a UUID suffix is an upstream
// connection and is skipped (PAR-ROUTE-060). With a nil fetcher it returns no
// models. A fetcher error is propagated so the caller can degrade to static-only
// silently (route.js:296-298).
func (r *LiveCatalogResolver) ResolveLiveModels() ([]ResolvedLiveModel, error) {
	if r == nil || r.fetcher == nil {
		return nil, nil
	}
	conns, err := r.cl.ListConnections()
	if err != nil {
		return nil, fmt.Errorf("list connections for live catalog: %w", err)
	}
	var out []ResolvedLiveModel
	for _, c := range conns {
		// PAR-ROUTE-060: skip upstream connections (UUID-suffixed names).
		if IsUpstreamConnection(c.Name) {
			continue
		}
		models, err := r.fetcher.Fetch(c)
		if err != nil {
			return nil, fmt.Errorf("fetch live models for %s: %w", c.Name, err)
		}
		for _, m := range models {
			if m.ID == "" {
				continue
			}
			out = append(out, ResolvedLiveModel{ID: m.ID, Provider: c.ProviderID})
		}
	}
	return out, nil
}
