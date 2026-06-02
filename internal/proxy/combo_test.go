package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestComboResolvePreservesStoredStepOrder(t *testing.T) {
	s := openProxyTestStore(t)
	if err := s.CreateCombo(&store.Combo{
		Name: "fast-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	resolver := NewComboResolver(s)
	steps, err := resolver.Resolve("fast-fallback")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("step count = %d, want 2", len(steps))
	}
	if steps[0].Provider != providers.ProviderGroq || steps[0].Model != "llama-3.3-70b-versatile" {
		t.Fatalf("first step = %#v, want groq llama-3.3-70b-versatile", steps[0])
	}
	if steps[1].Provider != providers.ProviderOpenAI || steps[1].Model != "gpt-4o-mini" {
		t.Fatalf("second step = %#v, want openai gpt-4o-mini", steps[1])
	}
}

func TestComboDispatchFallsBackSequentially(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "fast-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	groq := &fakeProvider{
		name: providers.ProviderGroq,
		err:  errors.New("rate limited"),
	}
	openAI := &fakeProvider{
		name: providers.ProviderOpenAI,
		response: &providers.ChatResponse{
			ID:    "chatcmpl-fallback",
			Model: "gpt-4o-mini",
		},
	}
	engine := NewEngine(s)
	engine.Register(groq)
	engine.Register(openAI)

	req := &providers.ChatRequest{Model: "combo/fast-fallback"}
	resp, err := NewComboResolver(s).Dispatch(context.Background(), engine, "fast-fallback", req)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if resp.ID != "chatcmpl-fallback" {
		t.Fatalf("response ID = %q, want chatcmpl-fallback", resp.ID)
	}
	if !groq.called {
		t.Fatal("first combo step was not called")
	}
	if !openAI.called {
		t.Fatal("second combo step was not called")
	}
	if groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("groq request model = %q, want llama-3.3-70b-versatile", groq.received.Model)
	}
	if openAI.received.Model != "gpt-4o-mini" {
		t.Fatalf("openai request model = %q, want gpt-4o-mini", openAI.received.Model)
	}
	if req.Model != "combo/fast-fallback" {
		t.Fatalf("original request model = %q, want combo/fast-fallback", req.Model)
	}
	if openAI.receivedKey.Value != "openai-key" {
		t.Fatalf("openai key = %q, want openai-key", openAI.receivedKey.Value)
	}
}

func TestComboDispatchReturnsLastStepError(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.CreateCombo(&store.Combo{
		Name: "fast-fallback",
		Steps: []store.ComboStep{
			{Provider: "groq", Model: "llama-3.3-70b-versatile"},
			{Provider: "openai", Model: "gpt-4o-mini"},
		},
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	lastErr := errors.New("quota exhausted")
	engine := NewEngine(s)
	engine.Register(&fakeProvider{name: providers.ProviderGroq, err: errors.New("rate limited")})
	engine.Register(&fakeProvider{name: providers.ProviderOpenAI, err: lastErr})

	_, err := NewComboResolver(s).Dispatch(
		context.Background(),
		engine,
		"fast-fallback",
		&providers.ChatRequest{Model: "combo/fast-fallback"},
	)
	if !errors.Is(err, lastErr) {
		t.Fatalf("Dispatch error = %v, want wrapped last step error", err)
	}
}

func createProxyConnection(t *testing.T, s *store.Store, provider string, key string) {
	t.Helper()

	if err := s.CreateConnection(&store.Connection{
		Provider: provider,
		Name:     provider + "-primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &key,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection %s: %v", provider, err)
	}
}
