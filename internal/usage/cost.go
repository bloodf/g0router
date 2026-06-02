package usage

import (
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
)

func CalculateCostUSD(catalog modelcatalog.Catalog, provider providers.ModelProvider, model string, usage *Usage) (float64, error) {
	if usage == nil {
		return 0, nil
	}

	price, ok := catalog.Lookup(provider, model)
	if !ok {
		return 0, fmt.Errorf("lookup pricing for %s/%s: %w", provider, model, errors.New("usage: pricing not found"))
	}

	cachedInputTokens := usage.CacheReadTokens
	inputTokens := usage.InputTokens - cachedInputTokens

	inputCost := float64(inputTokens) * price.InputPerMillionUSD / 1_000_000
	cachedInputCost := float64(cachedInputTokens) * price.CachedInputPerMillionUSD / 1_000_000
	outputCost := float64(usage.OutputTokens) * price.OutputPerMillionUSD / 1_000_000

	return inputCost + cachedInputCost + outputCost, nil
}
