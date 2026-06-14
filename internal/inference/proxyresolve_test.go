package inference

import (
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// fakeProxyResolver implements ProxyResolver for the selection proxy hook tests.
type fakeProxyResolver struct {
	byPoolID map[string]string
}

func (f *fakeProxyResolver) ResolveProxyForConnection(conn *store.Connection) (string, bool) {
	if conn == nil || conn.ProxyPoolID == "" {
		return "", false
	}
	url, ok := f.byPoolID[conn.ProxyPoolID]
	return url, ok
}

func TestSelectionResolveProxy(t *testing.T) {
	cs := &fakeConnStore{}
	ss := &fakeSettingStore{settings: map[string]string{}}
	cd := &fakeCooldownForSelection{}
	eng := NewSelectionEngine(cs, ss, cd, time.Now)

	resolver := &fakeProxyResolver{byPoolID: map[string]string{
		"pool-1": "http://proxy.example.com:8080",
	}}
	eng.SetProxyResolver(resolver)

	// Connection bound to an active proxy pool yields the proxy URL.
	bound := &store.Connection{ID: "c1", ProviderID: "openai", ProxyPoolID: "pool-1"}
	url, ok := eng.ResolveProxy(bound)
	if !ok || url != "http://proxy.example.com:8080" {
		t.Fatalf("ResolveProxy(bound) = (%q,%v); want the pool URL", url, ok)
	}

	// Connection with no proxy pool yields no proxy.
	unbound := &store.Connection{ID: "c2", ProviderID: "openai"}
	if url, ok := eng.ResolveProxy(unbound); ok || url != "" {
		t.Fatalf("ResolveProxy(unbound) = (%q,%v); want empty", url, ok)
	}
}

func TestSelectionResolveProxyNoResolver(t *testing.T) {
	cs := &fakeConnStore{}
	ss := &fakeSettingStore{settings: map[string]string{}}
	cd := &fakeCooldownForSelection{}
	eng := NewSelectionEngine(cs, ss, cd, time.Now)

	// With no resolver wired, ResolveProxy is a safe no-op (backward-compatible).
	if url, ok := eng.ResolveProxy(&store.Connection{ProxyPoolID: "pool-1"}); ok || url != "" {
		t.Fatalf("ResolveProxy without resolver = (%q,%v); want empty no-op", url, ok)
	}
}
