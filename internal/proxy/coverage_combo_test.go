package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// TestDispatchComboEmptyStepsReturnsError exercises the len(steps)==0 branch in
// Dispatch (line 91) and DispatchStream (line 120).
func TestDispatchComboEmptyStepsReturnsError(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateCombo(&store.Combo{
		Name:     "empty-combo",
		Steps:    []store.ComboStep{},
		IsActive: true,
		Strategy: "fallback",
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	engine := NewEngine(s)
	resolver := engine.comboResolver

	_, err := resolver.Dispatch(context.Background(), engine, "empty-combo", &providers.ChatRequest{Model: "empty-combo"})
	if !errors.Is(err, ErrNoComboSteps) {
		t.Fatalf("Dispatch empty steps: got %v, want ErrNoComboSteps", err)
	}

	_, err = resolver.DispatchStream(context.Background(), engine, "empty-combo", &providers.ChatRequest{Model: "empty-combo"})
	if !errors.Is(err, ErrNoComboSteps) {
		t.Fatalf("DispatchStream empty steps: got %v, want ErrNoComboSteps", err)
	}
}

// TestResolveComboStepRouteNonProviderNotFoundError exercises L440 in
// resolveComboStepRoute: when resolveModelRoute returns a non-ErrProviderNotFound
// error (DB error from ResolveModelAlias), L440 returns the error instead of
// falling through to routableModelRoute.
func TestResolveComboStepRouteNonProviderNotFoundError(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "sk-combo-route")
	engine := NewEngine(s)

	// Close the DB so ResolveModelAlias fails with a DB error (not ErrNotFound).
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Call resolveComboStepRoute directly with a model that is NOT in the catalog.
	// For such models, resolveModelRoute falls through to ResolveModelAlias.
	// With closed DB, ResolveModelAlias errors → L392 fires in resolveModelRoute
	// → error propagates to resolveComboStepRoute → L440 fires.
	step := ComboStep{Provider: providers.ProviderOpenAI, Model: "custom-unknown-model-xyz-123"}
	_, err := engine.resolveComboStepRoute(step)
	if err == nil {
		t.Fatal("resolveComboStepRoute with closed DB should error")
	}
}

// TestDispatchComboResolveComboStepRouteError exercises L440 in resolveComboStepRoute:
// when the store is closed, resolveModelRoute returns a non-ErrProviderNotFound error
// (DB error from ResolveModelAlias), triggering the error-passthrough branch.
func TestDispatchComboResolveComboStepRouteError(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "sk-openai-combo")
	provider := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "resp"},
	}
	if err := s.CreateCombo(&store.Combo{
		Name:     "step-route-err",
		Steps:    []store.ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
		Strategy: "fallback",
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	engine := NewEngine(s)
	engine.Register(provider)

	// Close the store so ResolveModelAlias errors → resolveComboStepRoute L440 fires.
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	resolver := engine.comboResolver
	_, err := resolver.Dispatch(context.Background(), engine, "step-route-err", &providers.ChatRequest{Model: "step-route-err"})
	if err == nil {
		t.Fatal("Dispatch with closed DB should error")
	}
}

// TestDispatchStreamComboFastestFetchesStats exercises the fastest-strategy stats
// path in DispatchStream (line 124): when strategy is "fastest", fetchTelemetryStats
// is called before dispatching.
func TestDispatchStreamComboFastestFetchesStats(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "openai", "sk-openai")
	ch := make(chan providers.StreamChunk)
	close(ch)
	provider := &fakeProvider{
		name:   providers.ProviderOpenAI,
		stream: ch,
	}
	if err := s.CreateCombo(&store.Combo{
		Name:     "fastest-combo",
		Steps:    []store.ComboStep{{Provider: "openai", Model: "gpt-4o"}},
		IsActive: true,
		Strategy: store.ComboStrategyFastest,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}
	engine := NewEngine(s)
	engine.Register(provider)
	resolver := engine.comboResolver

	stream, err := resolver.DispatchStream(context.Background(), engine, "fastest-combo", &providers.ChatRequest{Model: "fastest-combo"})
	if err != nil {
		t.Fatalf("DispatchStream fastest: %v", err)
	}
	for range stream {
	}
}
