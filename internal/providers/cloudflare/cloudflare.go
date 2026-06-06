package cloudflare

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
	"github.com/bloodf/g0router/internal/store"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// Provider routes Cloudflare AI Gateway REST API requests through its
// account-scoped OpenAI-compatible endpoint.
type Provider struct {
	baseURL   string
	proxyPool *store.ProxyPool
}

func New(baseURL string, proxyPool ...*store.ProxyPool) *Provider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	var pool *store.ProxyPool
	if len(proxyPool) > 0 {
		pool = proxyPool[0]
	}
	return &Provider{baseURL: strings.TrimRight(baseURL, "/"), proxyPool: pool}
}

func (p *Provider) WithProxyPool(pool *store.ProxyPool) providers.Provider {
	return New(p.baseURL, pool)
}

func (p *Provider) Name() providers.ModelProvider {
	return providers.ProviderCloudflare
}

func (p *Provider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	provider, err := p.compatProvider(key)
	if err != nil {
		return nil, err
	}
	return provider.ChatCompletion(ctx, key, req)
}

func (p *Provider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	provider, err := p.compatProvider(key)
	if err != nil {
		return nil, err
	}
	return provider.ChatCompletionStream(ctx, key, req)
}

func (p *Provider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	provider, err := p.compatProvider(key)
	if err != nil {
		return nil, err
	}
	return provider.ListModels(ctx, key)
}

func (p *Provider) compatProvider(key providers.Key) (*openaicompat.Provider, error) {
	accountID := strings.TrimSpace(key.AccountID)
	if accountID == "" {
		return nil, fmt.Errorf("cloudflare account id is required")
	}
	baseURL := p.baseURL + "/accounts/" + url.PathEscape(accountID) + "/ai/v1"
	provider, err := openaicompat.New(openaicompat.Config{
		Provider:  providers.ProviderCloudflare,
		BaseURL:   baseURL,
		ProxyPool: p.proxyPool,
	})
	if err != nil {
		return nil, fmt.Errorf("cloudflare provider: %w", err)
	}
	return provider, nil
}
