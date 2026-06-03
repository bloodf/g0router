package proxy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrNoConnections    = errors.New("no active connections")
)

type Engine struct {
	store *store.Store
	pool  providerPool
}

func NewEngine(s *store.Store) *Engine {
	return &Engine{
		store: s,
		pool:  newProviderPool(),
	}
}

func (e *Engine) Register(provider providers.Provider) {
	e.pool.register(provider)
}

func (e *Engine) RegisteredProviders() []providers.ModelProvider {
	return e.pool.names()
}

func (e *Engine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	provider, key, err := e.providerFor(req.Model)
	if err != nil {
		return nil, err
	}

	resp, err := provider.ChatCompletion(ctx, key, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}
	return resp, nil
}

func (e *Engine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	provider, key, err := e.providerFor(req.Model)
	if err != nil {
		return nil, err
	}

	stream, err := provider.ChatCompletionStream(ctx, key, req)
	if err != nil {
		return nil, fmt.Errorf("chat completion stream: %w", err)
	}
	return stream, nil
}

func (e *Engine) ListModels(ctx context.Context) ([]providers.Model, error) {
	var models []providers.Model
	for _, providerName := range e.pool.names() {
		providerModels, err := e.providerModels(ctx, providerName)
		if err != nil {
			return nil, err
		}
		models = append(models, providerModels...)
	}
	return models, nil
}

func (e *Engine) providerModels(ctx context.Context, providerName providers.ModelProvider) ([]providers.Model, error) {
	provider, ok := e.pool.get(providerName)
	if !ok {
		return nil, ErrProviderNotFound
	}

	key, err := e.keyFor(providerName)
	if errors.Is(err, ErrNoConnections) {
		return catalogModels(providerName), nil
	}
	if err != nil {
		return nil, err
	}

	models, err := provider.ListModels(ctx, key)
	if err == nil && len(models) > 0 {
		return models, nil
	}
	return catalogModels(providerName), nil
}

func catalogModels(providerName providers.ModelProvider) []providers.Model {
	prices := modelcatalog.NewCatalog().Models(providerName)
	modelIDs := make([]string, 0, len(prices))
	for modelID := range prices {
		modelIDs = append(modelIDs, modelID)
	}
	sort.Strings(modelIDs)

	models := make([]providers.Model, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		models = append(models, providers.Model{
			ID:       modelID,
			Object:   "model",
			OwnedBy:  providerName.String(),
			Provider: providerName,
		})
	}
	return models
}

func (e *Engine) providerFor(model string) (providers.Provider, providers.Key, error) {
	providerName, ok := resolveProvider(model)
	if !ok {
		return nil, providers.Key{}, ErrProviderNotFound
	}

	provider, ok := e.pool.get(providerName)
	if !ok {
		return nil, providers.Key{}, ErrProviderNotFound
	}

	key, err := e.keyFor(providerName)
	if err != nil {
		return nil, providers.Key{}, err
	}
	return provider, key, nil
}

func (e *Engine) keyFor(provider providers.ModelProvider) (providers.Key, error) {
	conns, err := e.store.GetActiveConnections(provider.String())
	if err != nil {
		return providers.Key{}, fmt.Errorf("get active connections: %w", err)
	}
	if len(conns) == 0 {
		return providers.Key{}, ErrNoConnections
	}

	conn := conns[0]
	key := providers.Key{
		Provider: provider,
		ConnID:   conn.ID,
		AuthType: string(conn.AuthType),
	}
	if conn.APIKey != nil {
		key.Value = *conn.APIKey
	} else if conn.AccessToken != nil {
		key.Value = *conn.AccessToken
	}

	return key, nil
}

func resolveProvider(model string) (providers.ModelProvider, bool) {
	switch {
	case strings.HasPrefix(model, "gpt-"):
		return providers.ProviderOpenAI, true
	case strings.HasPrefix(model, "claude-"):
		return providers.ProviderAnthropic, true
	default:
		return "", false
	}
}
