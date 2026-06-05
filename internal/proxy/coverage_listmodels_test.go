package proxy

import (
	"context"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// TestListModelsProviderModelsErrorContinues exercises the providerModels error
// path in ListModels (lines 316-318): when providerModels returns a non-nil
// error the provider is skipped and ListModels continues. We trigger this by
// closing the store so connectionForModel errors with a non-ErrNoConnections DB
// error, causing keyFor to return that error, which providerModels propagates.
func TestListModelsProviderModelsErrorContinues(t *testing.T) {
	s := openProxyTestStore(t)
	apiKey := "sk-test"
	// Create an active connection so the provider has at least one connection
	// registered — otherwise ErrNoConnections triggers catalog fallback (not error).
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "test",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	provider := &fakeProvider{
		name:   providers.ProviderOpenAI,
		models: []providers.Model{{ID: "gpt-4o", Provider: providers.ProviderOpenAI}},
	}
	engine := NewEngine(s)
	engine.Register(provider)

	// Close the store AFTER setup so the DB query inside connectionForModel fails
	// with a closed-DB error (not ErrNoConnections).
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// ListModels should not error; it skips failing providers and continues.
	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels should not error even when a provider fails: %v", err)
	}
	// No models returned because openai errored, but the function still returns.
	_ = models
}
