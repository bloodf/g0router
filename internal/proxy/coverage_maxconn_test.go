package proxy

import (
	"context"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// TestMaxConnectionAttemptsClosedDBContinues exercises the continue branch (line 548)
// in maxConnectionAttempts: when GetActiveConnections errors (closed DB),
// the error is swallowed and the loop continues, returning 0 → clamped to 1.
func TestMaxConnectionAttemptsClosedDBContinues(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn",
		AuthType: store.AuthTypeAPIKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	engine := NewEngine(s)

	// Close the DB so GetActiveConnections errors in maxConnectionAttempts.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// With closed DB, all GetActiveConnections calls fail → total=0 → returns 1.
	got := engine.maxConnectionAttempts(providers.ProviderOpenAI)
	if got != 1 {
		t.Fatalf("maxConnectionAttempts with closed DB = %d, want 1 (min)", got)
	}
}

// TestDispatchWithPreCachedAliasAndClosedDB exercises dispatch when resolveModelAlias
// returns from cache (no DB), so dispatchRoute IS entered and maxConnectionAttempts
// queries the closed DB, covering the continue-on-error branch (line 548).
func TestDispatchWithPreCachedAliasAndClosedDB(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn",
		AuthType: store.AuthTypeAPIKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	provider := &fakeProvider{name: providers.ProviderOpenAI}
	engine := NewEngine(s)
	engine.Register(provider)

	// Pre-populate alias cache so resolveModelAlias doesn't hit the DB.
	engine.aliasCache.set("my-cached-alias", store.ModelAlias{
		Alias:    "my-cached-alias",
		Provider: "openai",
		Model:    "gpt-4o",
	}, engine.now())

	// Close DB AFTER caching — resolveModelAlias returns from cache,
	// resolveModelRoute succeeds via alias, dispatchRoute is entered.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "my-cached-alias"})
	if err == nil {
		t.Fatal("Dispatch with closed DB should error")
	}
}
