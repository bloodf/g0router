package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestListModelsFiltersDisabledModels(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:   providers.ProviderOpenAI,
		models: []providers.Model{{ID: "gpt-4o", Provider: providers.ProviderOpenAI}, {ID: "gpt-4o-mini", Provider: providers.ProviderOpenAI}},
	}

	engine := NewEngine(s)
	engine.Register(openAI)

	// Disable one model.
	if _, err := s.CreateDisabledModel("openai", "gpt-4o"); err != nil {
		t.Fatalf("CreateDisabledModel: %v", err)
	}

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	var sawDisabled, sawEnabled bool
	for _, m := range models {
		if m.ID == "gpt-4o" {
			sawDisabled = true
		}
		if m.ID == "gpt-4o-mini" {
			sawEnabled = true
		}
	}
	if sawDisabled {
		t.Fatal("expected disabled model gpt-4o to be filtered from listing")
	}
	if !sawEnabled {
		t.Fatal("expected enabled model gpt-4o-mini to remain in listing")
	}
}

func TestListModelsIncludesCustomModels(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:   providers.ProviderOpenAI,
		models: []providers.Model{{ID: "gpt-4o", Provider: providers.ProviderOpenAI}},
	}

	engine := NewEngine(s)
	engine.Register(openAI)

	if _, err := s.CreateCustomModel("openai", "gpt-custom", "My Custom"); err != nil {
		t.Fatalf("CreateCustomModel: %v", err)
	}

	models, err := engine.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	var sawCustom bool
	for _, m := range models {
		if m.ID == "gpt-custom" {
			if !m.IsCustom {
				t.Fatal("expected custom model to have IsCustom=true")
			}
			if m.OwnedBy != "openai" {
				t.Fatalf("custom model owned_by = %q, want openai", m.OwnedBy)
			}
			sawCustom = true
		}
	}
	if !sawCustom {
		t.Fatal("expected custom model gpt-custom in listing")
	}
}

func TestDispatchRejectsDisabledModel(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{
		name:     providers.ProviderOpenAI,
		response: &providers.ChatResponse{ID: "chatcmpl-1"},
	}

	engine := NewEngine(s)
	engine.Register(openAI)

	if _, err := s.CreateDisabledModel("openai", "gpt-4o"); err != nil {
		t.Fatalf("CreateDisabledModel: %v", err)
	}

	_, err := engine.Dispatch(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrModelDisabled) {
		t.Fatalf("Dispatch error = %v, want ErrModelDisabled", err)
	}
	if openAI.called {
		t.Fatal("provider should not be called for disabled model")
	}
}

func TestDispatchStreamRejectsDisabledModel(t *testing.T) {
	s := openProxyTestStore(t)
	key := "sk-openai"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai", Name: "p", AuthType: store.AuthTypeAPIKey, APIKey: &key, IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	openAI := &fakeProvider{name: providers.ProviderOpenAI}

	engine := NewEngine(s)
	engine.Register(openAI)

	if _, err := s.CreateDisabledModel("openai", "gpt-4o"); err != nil {
		t.Fatalf("CreateDisabledModel: %v", err)
	}

	_, err := engine.DispatchStream(context.Background(), &providers.ChatRequest{Model: "gpt-4o"})
	if !errors.Is(err, ErrModelDisabled) {
		t.Fatalf("DispatchStream error = %v, want ErrModelDisabled", err)
	}
	if openAI.streamed {
		t.Fatal("provider should not be called for disabled model")
	}
}

func TestClassifyDispatchErrorModelDisabled(t *testing.T) {
	classified := ClassifyDispatchError(ErrModelDisabled)
	if classified.StatusCode != 400 {
		t.Fatalf("status code = %d, want 400", classified.StatusCode)
	}
	if classified.Code != "model_disabled" {
		t.Fatalf("code = %q, want model_disabled", classified.Code)
	}
}
