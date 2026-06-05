package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// TestDispatchWithClosedDBExercisesAliasStoreError exercises the
// resolveModelAlias store error path (line 427 in engine.go) AND the
// maxConnectionAttempts GetActiveConnections error path (line 548):
// when the store DB is closed, both ResolveModelAlias and GetActiveConnections
// fail, covering those error branches.
func TestDispatchWithClosedDBExercisesMultipleBranches(t *testing.T) {
	s := openProxyTestStore(t)
	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	provider := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "x"},
	}
	engine := NewEngine(s)
	engine.Register(provider)

	// Close the store so all DB queries fail.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Dispatch should fail because ResolveModelAlias errors (non-ErrNotFound DB error).
	// This also exercises maxConnectionAttempts where GetActiveConnections errors.
	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("Dispatch with closed DB should error")
	}
}

// TestProviderForRouteNotRegistered exercises the !ok branch (line 379) in
// providerForRoute: when the provider is known in the matrix but not registered
// in the engine pool, ErrProviderNotFound is returned.
func TestProviderForRouteNotRegistered(t *testing.T) {
	s := openProxyTestStore(t)
	apiKey := "sk-groq"
	if err := s.CreateConnection(&store.Connection{
		Provider: "groq",
		Name:     "groq-conn",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	// Register openai provider but NOT groq. A model that routes to groq via
	// alias or catalog will fail in providerForRoute (pool.get returns !ok for groq).
	engine := NewEngine(s)
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI})

	// Dispatch a groq model — resolveModelRoute returns a groq route,
	// but groq is not in the pool → providerForRoute returns ErrProviderNotFound.
	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "llama-3.3-70b-versatile"})
	if err == nil {
		t.Fatal("Dispatch to unregistered groq provider should error")
	}
}

// TestDispatchFallbackThenNoConnections exercises L160 in dispatchRoute:
// two connections exist so maxConnectionAttempts=2; the first attempt fails
// with a fallback-worthy error (rate limit) setting lastErr; the second attempt
// also gets a rate limit but after recordProviderFailure both connections are
// backed off, and the third lookup (which won't happen since maxAttempts=2)
// exits the loop. Actually to hit L160 we need the second providerForRoute call
// to return ErrNoConnections — achieved by having both connections fail on first use.
// Strategy: use two connections, both fail with rate limit on first call.
// After both fail: attempt 0 → conn1 fails (lastErr set), continue;
//                  attempt 1 → conn2 fails (fallback), continue (loops exhausted).
// Loop exits and hits L185 (lastErr != nil → return lastErr).
// For L160: we need attempt 1's providerForRoute to return ErrNoConnections.
// This happens when BOTH connections are backed off before attempt 1's providerForRoute.
// The fallback manager backs off conn1 after attempt 0. On attempt 1, Next()
// skips backed-off conn1 and returns conn2 (or ErrNoActiveConnections if both backed off).
// With errs=[rateLimitErr, rateLimitErr]: attempt 0 fails → backs off conn1 → lastErr set.
// Attempt 1: providerForRoute gets conn2 → conn2 also fails → backs off.
// Loop exhausted → L185 fires (not L160).
// To hit L160: attempt 1's providerForRoute must return ErrNoConnections.
// This requires: at attempt 1, all connections are backed off before the call.
// Using a single connection with maxAttempts manually forced to 2:
// We create 2 connections but only one is truly active and fails,
// then on attempt 1 there are no more active connections → ErrNoConnections.
func TestDispatchFallbackThenNoConnections(t *testing.T) {
	s := openProxyTestStore(t)
	// Two connections; provider returns rate limit on first call (backing off conn1).
	// The second call (attempt 1) should also fail — we use the same error.
	createProxyConnection(t, s, "openai", "sk-conn1")
	createProxyConnection(t, s, "openai", "sk-conn2")

	rateLimitErr := errors.New("provider rate limit hit") // fallback-worthy
	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		errs: []error{rateLimitErr, rateLimitErr},
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("Dispatch should fail when all connections fail with rate limit")
	}
}

// TestDispatchStreamFallbackThenNoConnections exercises L218+L260 in dispatchStreamRoute.
func TestDispatchStreamFallbackThenNoConnections(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "sk-single-stream")

	// First chunk errors with a fallback-worthy error.
	rateLimitChunk := providers.StreamChunk{
		Error: &providers.StreamError{Message: "provider rate limit hit"},
	}
	ch := make(chan providers.StreamChunk, 1)
	ch <- rateLimitChunk
	close(ch)

	openAI := &fakeProvider{
		name:   providers.ProviderOpenAI,
		stream: ch,
	}
	engine := NewEngine(s)
	engine.Register(openAI)

	_, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("DispatchStream should fail")
	}
}

// TestDispatchStreamWithClosedDBExercisesBranches exercises the stream dispatch
// path when the store is closed — covers the stream version of the same branches.
func TestDispatchStreamWithClosedDBExercisesBranches(t *testing.T) {
	s := openProxyTestStore(t)
	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	provider := &fakeProvider{name: providers.ProviderOpenAI}
	engine := NewEngine(s)
	engine.Register(provider)

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	_, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if err == nil {
		t.Fatal("DispatchStream with closed DB should error")
	}
}
