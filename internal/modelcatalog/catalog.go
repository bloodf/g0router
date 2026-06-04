package modelcatalog

import (
	"sort"

	"github.com/bloodf/g0router/internal/providers"
)

type Catalog struct {
	prices map[providers.ModelProvider]map[string]Pricing
}

func NewCatalog() Catalog {
	paid := func(input, cachedInput, output float64) Pricing {
		return Pricing{
			InputPerMillionUSD:       input,
			CachedInputPerMillionUSD: cachedInput,
			OutputPerMillionUSD:      output,
		}
	}
	noHostedCacheDiscount := func(input, output float64) Pricing {
		return paid(input, input, output)
	}

	return Catalog{
		prices: map[providers.ModelProvider]map[string]Pricing{
			providers.ProviderOpenAI: {
				"gpt-4o":      paid(2.50, 1.25, 10.00),
				"gpt-4o-mini": paid(0.15, 0.075, 0.60),
			},
			providers.ProviderAnthropic: {
				"claude-sonnet-4":           paid(3.00, 0.30, 15.00),
				"claude-sonnet-4-20250514":  paid(3.00, 0.30, 15.00),
				"claude-opus-4":             paid(15.00, 1.50, 75.00),
				"claude-opus-4-20250514":    paid(15.00, 1.50, 75.00),
				"claude-3-5-haiku-20241022": paid(0.80, 0.08, 4.00),
			},
			providers.ProviderGemini: {
				"gemini-2.5-flash":      paid(0.30, 0.03, 2.50),
				"gemini-2.5-flash-lite": paid(0.10, 0.01, 0.40),
			},
			providers.ProviderGroq: {
				"llama-3.3-70b-versatile": noHostedCacheDiscount(0.59, 0.79),
				"llama-3.1-8b-instant":    noHostedCacheDiscount(0.05, 0.08),
			},
			providers.ProviderMistral: {
				"mistral-large-latest":   noHostedCacheDiscount(2.00, 6.00),
				"mistral-small-latest":   noHostedCacheDiscount(0.10, 0.30),
				"magistral-small-latest": noHostedCacheDiscount(0.50, 1.50),
				"ministral-8b-latest":    noHostedCacheDiscount(0.15, 0.15),
			},
			providers.ProviderOpenRouter: {
				"openai/gpt-4o":      noHostedCacheDiscount(2.50, 10.00),
				"openai/gpt-4o-mini": noHostedCacheDiscount(0.15, 0.60),
			},
			providers.ProviderDeepSeek: {
				"deepseek-chat":     paid(0.27, 0.07, 1.10),
				"deepseek-reasoner": paid(0.55, 0.14, 2.19),
			},
			providers.ProviderPerplexity: {
				"sonar":               noHostedCacheDiscount(1.00, 1.00),
				"sonar-pro":           noHostedCacheDiscount(3.00, 15.00),
				"sonar-reasoning-pro": noHostedCacheDiscount(2.00, 8.00),
			},
			providers.ProviderMiniMax: {
				"MiniMax-M3": paid(0.30, 0.06, 1.20),
			},
			providers.ProviderQwen: {
				"qwen3.7-max":  noHostedCacheDiscount(2.50, 7.50),
				"qwen3.6-plus": noHostedCacheDiscount(0.50, 3.00),
			},
			providers.ProviderXAI: {
				"grok-4.3": paid(1.25, 0.20, 2.50),
			},
			providers.ProviderCerebras: {
				"llama3.1-8b": noHostedCacheDiscount(0.10, 0.10),
			},
			providers.ProviderCohere: {
				"command-r-08-2024": noHostedCacheDiscount(0.15, 0.60),
			},
			providers.ProviderFireworks: {
				"accounts/fireworks/models/deepseek-v4-flash":       paid(0.14, 0.028, 0.28),
				"accounts/fireworks/models/llama-v3p1-70b-instruct": paid(0.30, 0.15, 1.20),
				"accounts/fireworks/models/llama-v3p3-70b-instruct": noHostedCacheDiscount(0.90, 0.90),
			},
			providers.ProviderTogether: {
				"meta-llama/Llama-3.3-70B-Instruct-Turbo":  noHostedCacheDiscount(1.04, 1.04),
				"meta-llama/Meta-Llama-3-8B-Instruct-Lite": noHostedCacheDiscount(0.10, 0.10),
			},
			providers.ProviderOllama: {
				"llama3.1:8b":  {},
				"llama3.3:70b": {},
			},
			providers.ProviderVertex: {
				"gemini-2.5-flash":      paid(0.30, 0.03, 2.50),
				"gemini-2.5-flash-lite": paid(0.10, 0.01, 0.40),
			},
		},
	}
}

func (c Catalog) Lookup(provider providers.ModelProvider, model string) (Pricing, bool) {
	models, ok := c.prices[provider]
	if !ok {
		return Pricing{}, false
	}

	price, ok := models[model]
	return price, ok
}

func (c Catalog) ProviderForModel(model string) (providers.ModelProvider, bool) {
	for _, provider := range c.providerNames() {
		if _, ok := c.prices[provider][model]; ok {
			return provider, true
		}
	}
	return "", false
}

func (c Catalog) Models(provider providers.ModelProvider) map[string]Pricing {
	models := c.prices[provider]
	result := make(map[string]Pricing, len(models))
	for model, price := range models {
		result[model] = price
	}

	return result
}

func (c Catalog) providerNames() []providers.ModelProvider {
	names := make([]providers.ModelProvider, 0, len(c.prices))
	for provider := range c.prices {
		names = append(names, provider)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}
