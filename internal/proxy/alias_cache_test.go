package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestDispatchCachesResolvedAliasWithinTTL(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias initial: %v", err)
	}

	now := time.Unix(1_700_000_000, 0)
	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-openai"}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(groq)
	engine.Register(openAI)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "fast"}); err != nil {
		t.Fatalf("Dispatch initial: %v", err)
	}
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "openai",
		Model:    "gpt-4o-mini",
	}); err != nil {
		t.Fatalf("SetModelAlias updated: %v", err)
	}
	now = now.Add(time.Minute)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "fast"}); err != nil {
		t.Fatalf("Dispatch cached: %v", err)
	}
	if groq.calls != 2 {
		t.Fatalf("groq calls = %d, want cached alias to route twice to groq", groq.calls)
	}
	if openAI.calls != 0 {
		t.Fatalf("openai calls = %d, want alias update hidden until TTL expires", openAI.calls)
	}
	if groq.requests[1].Model != "llama-3.3-70b-versatile" {
		t.Fatalf("cached request model = %q, want original alias target", groq.requests[1].Model)
	}
}

func TestDispatchRefreshesAliasAfterTTL(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias initial: %v", err)
	}

	now := time.Unix(1_700_000_000, 0)
	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-openai"}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(groq)
	engine.Register(openAI)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "fast"}); err != nil {
		t.Fatalf("Dispatch initial: %v", err)
	}
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "fast",
		Provider: "openai",
		Model:    "gpt-4o-mini",
	}); err != nil {
		t.Fatalf("SetModelAlias updated: %v", err)
	}
	now = now.Add(6 * time.Minute)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "fast"}); err != nil {
		t.Fatalf("Dispatch refreshed: %v", err)
	}
	if groq.calls != 1 {
		t.Fatalf("groq calls = %d, want only initial cached route", groq.calls)
	}
	if openAI.calls != 1 {
		t.Fatalf("openai calls = %d, want refreshed alias route", openAI.calls)
	}
	if openAI.received.Model != "gpt-4o-mini" {
		t.Fatalf("refreshed request model = %q, want updated alias target", openAI.received.Model)
	}
}

func TestDispatchDoesNotCacheMissingAlias(t *testing.T) {
	s := openProxyTestStore(t)
	createProxyConnection(t, s, "groq", "groq-key")
	createProxyConnection(t, s, "openai", "openai-key")

	now := time.Unix(1_700_000_000, 0)
	groq := &fakeProvider{name: providers.ProviderGroq, response: &providers.ChatResponse{ID: "chatcmpl-groq"}}
	openAI := &fakeProvider{name: providers.ProviderOpenAI, response: &providers.ChatResponse{ID: "chatcmpl-openai"}}
	engine := NewEngine(s)
	engine.now = func() time.Time { return now }
	engine.Register(groq)
	engine.Register(openAI)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-cache-later"}); err != nil {
		t.Fatalf("Dispatch prefix fallback: %v", err)
	}
	if err := s.SetModelAlias(store.ModelAlias{
		Alias:    "gpt-cache-later",
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	}); err != nil {
		t.Fatalf("SetModelAlias: %v", err)
	}
	now = now.Add(time.Minute)

	if _, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-cache-later"}); err != nil {
		t.Fatalf("Dispatch new alias: %v", err)
	}
	if openAI.calls != 1 {
		t.Fatalf("openai calls = %d, want only the initial prefix fallback", openAI.calls)
	}
	if groq.calls != 1 {
		t.Fatalf("groq calls = %d, want newly created alias route", groq.calls)
	}
	if groq.received.Model != "llama-3.3-70b-versatile" {
		t.Fatalf("groq request model = %q, want alias target", groq.received.Model)
	}
}
