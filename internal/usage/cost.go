package usage

// CalculateCost computes the dollar cost from normalized tokens and pricing.
// Rates are dollars per 1M tokens. Zero/absent pricing yields zero cost.
func CalculateCost(tokens TokenSet, p Pricing) float64 {
	if p == (Pricing{}) {
		return 0
	}

	var cost float64

	nonCachedInput := tokens.Prompt - tokens.Cached
	if nonCachedInput < 0 {
		nonCachedInput = 0
	}
	cost += float64(nonCachedInput) * (p.Input / 1e6)

	if tokens.Cached > 0 {
		rate := p.Cached
		if rate == 0 {
			rate = p.Input
		}
		cost += float64(tokens.Cached) * (rate / 1e6)
	}

	cost += float64(tokens.Completion) * (p.Output / 1e6)

	if tokens.Reasoning > 0 {
		rate := p.Reasoning
		if rate == 0 {
			rate = p.Output
		}
		cost += float64(tokens.Reasoning) * (rate / 1e6)
	}

	if tokens.CacheCreation > 0 {
		rate := p.CacheCreation
		if rate == 0 {
			rate = p.Input
		}
		cost += float64(tokens.CacheCreation) * (rate / 1e6)
	}

	return cost
}

// CostFor resolves pricing for provider/model and calculates cost.
// Returns 0 when no pricing resolves.
func (r *Resolver) CostFor(provider, model string, tokens TokenSet) float64 {
	p, ok := r.PricingForModel(provider, model)
	if !ok {
		return 0
	}
	return CalculateCost(tokens, p)
}
