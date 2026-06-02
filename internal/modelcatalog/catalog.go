package modelcatalog

import "github.com/bloodf/g0router/internal/providers"

type Catalog struct {
	prices map[providers.ModelProvider]map[string]Pricing
}

func NewCatalog() Catalog {
	return Catalog{
		prices: map[providers.ModelProvider]map[string]Pricing{
			providers.ProviderOpenAI: {
				"gpt-4o": {
					InputPerMillionUSD:       2.50,
					CachedInputPerMillionUSD: 1.25,
					OutputPerMillionUSD:      10.00,
				},
				"gpt-4o-mini": {
					InputPerMillionUSD:       0.15,
					CachedInputPerMillionUSD: 0.075,
					OutputPerMillionUSD:      0.60,
				},
			},
			providers.ProviderAnthropic: {
				"claude-sonnet-4": {
					InputPerMillionUSD:       3.00,
					CachedInputPerMillionUSD: 0.30,
					OutputPerMillionUSD:      15.00,
				},
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

func (c Catalog) Models(provider providers.ModelProvider) map[string]Pricing {
	models := c.prices[provider]
	result := make(map[string]Pricing, len(models))
	for model, price := range models {
		result[model] = price
	}

	return result
}
