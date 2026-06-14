package inference

import (
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// fakeConnLister implements ConnLister for live-catalog resolver tests.
type fakeConnLister struct {
	conns []*store.Connection
	err   error
}

func (f *fakeConnLister) ListConnections() ([]*store.Connection, error) {
	return f.conns, f.err
}

// fakeLiveFetcher implements LiveCatalogFetcher: a deterministic, network-free
// fake that records which connections it was asked to fetch for.
type fakeLiveFetcher struct {
	byProvider map[string][]LiveCatalogModel
	called     []string // connection names passed to Fetch
	err        error
}

func (f *fakeLiveFetcher) Fetch(conn *store.Connection) ([]LiveCatalogModel, error) {
	f.called = append(f.called, conn.Name)
	if f.err != nil {
		return nil, f.err
	}
	return f.byProvider[conn.ProviderID], nil
}

// TestLiveCatalogResolverMergesDynamicModels verifies PAR-ROUTE-056: the resolver
// fetches per-connection dynamic models via the injectable fetcher and returns
// them (no network).
func TestLiveCatalogResolverMergesDynamicModels(t *testing.T) {
	cl := &fakeConnLister{conns: []*store.Connection{
		{ID: "c1", ProviderID: "kiro", Name: "kiro-prod"},
	}}
	fetcher := &fakeLiveFetcher{byProvider: map[string][]LiveCatalogModel{
		"kiro": {{ID: "kiro-dyn-1"}, {ID: "kiro-dyn-2"}},
	}}
	r := NewLiveCatalogResolver(cl, fetcher)

	models, err := r.ResolveLiveModels()
	if err != nil {
		t.Fatalf("ResolveLiveModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2: %+v", len(models), models)
	}
	if models[0].ID != "kiro-dyn-1" || models[0].Provider != "kiro" {
		t.Errorf("models[0] = %+v, want {kiro-dyn-1, kiro}", models[0])
	}
}

// TestLiveCatalogResolverSkipsUpstream verifies PAR-ROUTE-060: a connection whose
// name carries a UUID suffix (upstream connection) is NOT fetched.
func TestLiveCatalogResolverSkipsUpstream(t *testing.T) {
	cl := &fakeConnLister{conns: []*store.Connection{
		{ID: "c1", ProviderID: "kiro", Name: "kiro-prod"},
		{ID: "c2", ProviderID: "kiro", Name: "kiro-550e8400-e29b-41d4-a716-446655440000"},
	}}
	fetcher := &fakeLiveFetcher{byProvider: map[string][]LiveCatalogModel{
		"kiro": {{ID: "kiro-dyn-1"}},
	}}
	r := NewLiveCatalogResolver(cl, fetcher)

	if _, err := r.ResolveLiveModels(); err != nil {
		t.Fatalf("ResolveLiveModels: %v", err)
	}
	if len(fetcher.called) != 1 || fetcher.called[0] != "kiro-prod" {
		t.Errorf("fetcher called for %v, want only [kiro-prod] (upstream skipped)", fetcher.called)
	}
}

// TestLiveCatalogResolverFetcherErrorPropagates verifies the resolver surfaces a
// fetcher error so the handler can degrade to static-only silently.
func TestLiveCatalogResolverFetcherErrorPropagates(t *testing.T) {
	cl := &fakeConnLister{conns: []*store.Connection{
		{ID: "c1", ProviderID: "kiro", Name: "kiro-prod"},
	}}
	fetcher := &fakeLiveFetcher{err: errors.New("boom")}
	r := NewLiveCatalogResolver(cl, fetcher)

	if _, err := r.ResolveLiveModels(); err == nil {
		t.Fatal("expected error from failing fetcher, got nil")
	}
}

// TestLiveCatalogResolverNoFetcher verifies a nil fetcher yields no models and no
// error (additive, backward-compatible).
func TestLiveCatalogResolverNoFetcher(t *testing.T) {
	cl := &fakeConnLister{conns: []*store.Connection{
		{ID: "c1", ProviderID: "kiro", Name: "kiro-prod"},
	}}
	r := NewLiveCatalogResolver(cl, nil)
	models, err := r.ResolveLiveModels()
	if err != nil {
		t.Fatalf("ResolveLiveModels: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("got %d models, want 0 with nil fetcher", len(models))
	}
}
