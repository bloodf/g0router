package usage

import (
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/modelcatalog"
	"github.com/bloodf/g0router/internal/providers"
)

type PricingOverride struct {
	InputCostPerToken  float64
	OutputCostPerToken float64
}

type PricingOverrideResolver interface {
	PricingOverride(provider, model string) (PricingOverride, bool, error)
}

func CalculateCostUSD(catalog modelcatalog.Catalog, provider providers.ModelProvider, model string, usage *Usage) (float64, error) {
	return CalculateCostUSDWithOverrides(catalog, nil, provider, model, usage)
}

func CalculateCostUSDWithOverrides(catalog modelcatalog.Catalog, overrides PricingOverrideResolver, provider providers.ModelProvider, model string, usage *Usage) (float64, error) {
	if usage == nil {
		return 0, nil
	}

	if overrides != nil {
		override, ok, err := overrides.PricingOverride(provider.String(), model)
		if err != nil {
			return 0, fmt.Errorf("lookup pricing override for %s/%s: %w", provider, model, err)
		}
		if ok {
			return calculateOverrideCostUSD(override, usage), nil
		}
	}

	price, ok := catalog.Lookup(provider, model)
	if !ok {
		return 0, fmt.Errorf("lookup pricing for %s/%s: %w", provider, model, errors.New("usage: pricing not found"))
	}

	// Convention: InputTokens is inclusive of cache reads; we subtract to get the
	// non-cached portion. Some providers report InputTokens exclusive of cache reads,
	// which would underflow. Clamp both values to defend against that.
	cachedInputTokens := usage.CacheReadTokens
	if cachedInputTokens < 0 {
		cachedInputTokens = 0
	}
	inputTokens := usage.InputTokens - cachedInputTokens
	if inputTokens < 0 {
		inputTokens = 0
	}

	inputCost := float64(inputTokens) * price.InputPerMillionUSD / 1_000_000
	cachedInputCost := float64(cachedInputTokens) * price.CachedInputPerMillionUSD / 1_000_000
	outputCost := float64(usage.OutputTokens) * price.OutputPerMillionUSD / 1_000_000

	return inputCost + cachedInputCost + outputCost, nil
}

func calculateOverrideCostUSD(price PricingOverride, usage *Usage) float64 {
	inputCost := float64(usage.InputTokens) * price.InputCostPerToken
	outputCost := float64(usage.OutputTokens) * price.OutputCostPerToken
	return inputCost + outputCost
}
