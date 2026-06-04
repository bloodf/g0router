package xiaomi

import (
	"context"
	"strings"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/anthropic"
)

const (
	defaultStandardBaseURL  = "https://api.xiaomimimo.com/anthropic"
	defaultTokenPlanBaseURL = "https://token-plan-ams.xiaomimimo.com/anthropic"
)

type Provider struct {
	standard  *anthropic.AnthropicProvider
	tokenPlan *anthropic.AnthropicProvider
}

func New(standardBaseURL, tokenPlanBaseURL string) *Provider {
	if strings.TrimSpace(standardBaseURL) == "" {
		standardBaseURL = defaultStandardBaseURL
	}
	if strings.TrimSpace(tokenPlanBaseURL) == "" {
		tokenPlanBaseURL = defaultTokenPlanBaseURL
	}
	return &Provider{
		standard:  anthropic.NewForProvider(providers.ProviderXiaomi, standardBaseURL),
		tokenPlan: anthropic.NewForProvider(providers.ProviderXiaomi, tokenPlanBaseURL),
	}
}

func NewDefault() *Provider {
	return New("", "")
}

func (p *Provider) Name() providers.ModelProvider {
	return providers.ProviderXiaomi
}

func (p *Provider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return p.providerForKey(key).ChatCompletion(ctx, key, req)
}

func (p *Provider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return p.providerForKey(key).ChatCompletionStream(ctx, key, req)
}

func (p *Provider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return p.providerForKey(key).ListModels(ctx, key)
}

func (p *Provider) providerForKey(key providers.Key) *anthropic.AnthropicProvider {
	if strings.HasPrefix(key.Value, "tp-") {
		return p.tokenPlan
	}
	return p.standard
}
