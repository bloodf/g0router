package provider

import (
	"errors"

	"github.com/bloodf/g0router/internal/providers"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrModelNotFound    = errors.New("model not found")
)

type Registry struct {
	providers map[providers.ModelProvider]providers.Provider
	models    map[string]providers.Model
}

func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[providers.ModelProvider]providers.Provider),
		models:    make(map[string]providers.Model),
	}
}

func (r *Registry) Register(provider providers.Provider) {
	r.providers[provider.Name()] = provider
}

func (r *Registry) RegisterModels(models []providers.Model) {
	for _, model := range models {
		r.models[model.ID] = model
	}
}

func (r *Registry) Provider(name providers.ModelProvider) (providers.Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

func (r *Registry) Resolve(modelID string) (providers.Provider, providers.Model, error) {
	model, ok := r.models[modelID]
	if !ok {
		return nil, providers.Model{}, ErrModelNotFound
	}

	provider, ok := r.Provider(model.Provider)
	if !ok {
		return nil, providers.Model{}, ErrProviderNotFound
	}

	return provider, model, nil
}
