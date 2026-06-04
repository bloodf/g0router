package cloudflare

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

// Provider routes Cloudflare AI Gateway REST API requests through its
// account-scoped OpenAI-compatible endpoint.
type Provider struct {
	baseURL string
}

func New(baseURL string) *Provider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{baseURL: strings.TrimRight(baseURL, "/")}
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
		Provider: providers.ProviderCloudflare,
		BaseURL:  baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("cloudflare provider: %w", err)
	}
	return provider, nil
}
