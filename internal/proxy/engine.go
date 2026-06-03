package proxy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/modelcatalog"
	providercore "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrNoConnections    = errors.New("no active connections")
)

const defaultRefreshWindow = 5 * time.Minute

type oauthRefresher interface {
	Refresh(ctx context.Context, refreshToken string) (oauth.TokenResult, error)
}

type Engine struct {
	store          *store.Store
	pool           providerPool
	refreshers     map[oauth.ProviderID]oauthRefresher
	refreshManager *providercore.RefreshManager
	refreshWindow  time.Duration
	now            func() time.Time
}

func NewEngine(s *store.Store) *Engine {
	return &Engine{
		store:          s,
		pool:           newProviderPool(),
		refreshers:     make(map[oauth.ProviderID]oauthRefresher),
		refreshManager: providercore.NewRefreshManager(),
		refreshWindow:  defaultRefreshWindow,
		now:            time.Now,
	}
}

func (e *Engine) Register(provider providers.Provider) {
	e.pool.register(provider)
}

func (e *Engine) RegisterOAuthRefresher(provider oauth.ProviderID, refresher oauthRefresher) {
	e.refreshers[provider] = refresher
}

func (e *Engine) RegisteredProviders() []providers.ModelProvider {
	return e.pool.names()
}

func (e *Engine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	provider, key, err := e.providerFor(ctx, req.Model)
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
	provider, key, err := e.providerFor(ctx, req.Model)
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

	key, err := e.keyFor(ctx, providerName)
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

func (e *Engine) providerFor(ctx context.Context, model string) (providers.Provider, providers.Key, error) {
	providerName, ok := resolveProvider(model)
	if !ok {
		return nil, providers.Key{}, ErrProviderNotFound
	}

	provider, ok := e.pool.get(providerName)
	if !ok {
		return nil, providers.Key{}, ErrProviderNotFound
	}

	key, err := e.keyFor(ctx, providerName)
	if err != nil {
		return nil, providers.Key{}, err
	}
	return provider, key, nil
}

func (e *Engine) keyFor(ctx context.Context, provider providers.ModelProvider) (providers.Key, error) {
	conns, err := e.activeConnectionsForProvider(provider)
	if err != nil {
		return providers.Key{}, fmt.Errorf("get active connections: %w", err)
	}
	if len(conns) == 0 {
		return providers.Key{}, ErrNoConnections
	}

	conn := conns[0]
	conn, err = e.refreshConnectionIfNeeded(ctx, provider, conn)
	if err != nil {
		return providers.Key{}, err
	}

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

func (e *Engine) activeConnectionsForProvider(provider providers.ModelProvider) ([]*store.Connection, error) {
	var connections []*store.Connection
	for _, providerID := range providercore.ProviderAliases(provider.String()) {
		providerConnections, err := e.store.GetActiveConnections(providerID)
		if err != nil {
			return nil, err
		}
		connections = append(connections, providerConnections...)
	}
	return connections, nil
}

func (e *Engine) refreshConnectionIfNeeded(ctx context.Context, provider providers.ModelProvider, conn *store.Connection) (*store.Connection, error) {
	if !e.connectionNeedsRefresh(conn) {
		return conn, nil
	}
	oauthProvider := e.oauthProviderForConnection(provider, conn)
	refresher, ok := e.refreshers[oauthProvider]
	if !ok {
		return conn, nil
	}

	token, err := e.refreshManager.Refresh(ctx, conn, func(ctx context.Context, conn *store.Connection) (oauth.TokenResult, error) {
		return refresher.Refresh(ctx, *conn.RefreshToken)
	})
	if err != nil {
		return nil, fmt.Errorf("refresh oauth credentials: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("refresh oauth credentials: access token is required")
	}

	accessToken := token.AccessToken
	refreshToken := conn.RefreshToken
	if token.RefreshToken != "" {
		newRefreshToken := token.RefreshToken
		refreshToken = &newRefreshToken
	}
	expiresAt := conn.ExpiresAt
	if !token.ExpiresAt.IsZero() {
		newExpiresAt := token.ExpiresAt.Unix()
		expiresAt = &newExpiresAt
	}

	if err := e.store.UpdateConnectionCredentials(conn.ID, &accessToken, refreshToken, expiresAt); err != nil {
		return nil, fmt.Errorf("update refreshed credentials: %w", err)
	}

	updated := *conn
	updated.AccessToken = &accessToken
	updated.RefreshToken = refreshToken
	updated.ExpiresAt = expiresAt
	return &updated, nil
}

func (e *Engine) oauthProviderForConnection(runtimeProvider providers.ModelProvider, conn *store.Connection) oauth.ProviderID {
	if conn.ProviderSpecificData != nil {
		if value, ok := conn.ProviderSpecificData["oauth_provider"].(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return oauth.ProviderID(value)
			}
		}
	}
	if runtimeProvider == providers.ProviderOpenAI {
		return oauth.ProviderID("codex")
	}
	return oauth.ProviderID(runtimeProvider.String())
}

func (e *Engine) connectionNeedsRefresh(conn *store.Connection) bool {
	if conn.AuthType != store.AuthTypeOAuth {
		return false
	}
	if conn.RefreshToken == nil || *conn.RefreshToken == "" {
		return false
	}
	if conn.ExpiresAt == nil {
		return false
	}
	return time.Unix(*conn.ExpiresAt, 0).Before(e.now().Add(e.refreshWindow))
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
