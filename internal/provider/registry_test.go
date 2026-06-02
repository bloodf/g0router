package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

type fakeProvider struct {
	name providers.ModelProvider
}

func (f *fakeProvider) Name() providers.ModelProvider {
	return f.name
}

func (f *fakeProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, nil
}

func (f *fakeProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f *fakeProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

func TestRegistryReturnsRegisteredProvider(t *testing.T) {
	registry := NewRegistry()
	openAI := &fakeProvider{name: providers.ProviderOpenAI}

	registry.Register(openAI)

	got, ok := registry.Provider(providers.ProviderOpenAI)
	if !ok {
		t.Fatal("provider was not registered")
	}
	if got != openAI {
		t.Fatal("registered provider mismatch")
	}
}

func TestRegistryResolveModel(t *testing.T) {
	registry := NewRegistry()
	openAI := &fakeProvider{name: providers.ProviderOpenAI}
	anthropic := &fakeProvider{name: providers.ProviderAnthropic}
	registry.Register(openAI)
	registry.Register(anthropic)
	registry.RegisterModels([]providers.Model{
		{ID: "gpt-4o", Provider: providers.ProviderOpenAI},
		{ID: "claude-sonnet-4", Provider: providers.ProviderAnthropic},
	})

	gotProvider, gotModel, err := registry.Resolve("claude-sonnet-4")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if gotProvider != anthropic {
		t.Fatal("resolved provider mismatch")
	}
	if gotModel.ID != "claude-sonnet-4" {
		t.Fatalf("model ID = %q, want claude-sonnet-4", gotModel.ID)
	}
	if gotModel.Provider != providers.ProviderAnthropic {
		t.Fatalf("model provider = %q, want anthropic", gotModel.Provider)
	}
}

func TestRegistryResolveUnknownModel(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&fakeProvider{name: providers.ProviderOpenAI})

	_, _, err := registry.Resolve("missing-model")
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestRegistryResolveUnregisteredProvider(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterModels([]providers.Model{
		{ID: "gpt-4o", Provider: providers.ProviderOpenAI},
	})

	_, _, err := registry.Resolve("gpt-4o")
	if !errors.Is(err, ErrProviderNotFound) {
		t.Fatalf("expected ErrProviderNotFound, got %v", err)
	}
}
