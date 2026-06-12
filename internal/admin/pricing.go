package admin

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/usage"
	"github.com/valyala/fasthttp"
)

var validPricingFields = map[string]bool{
	"input":          true,
	"output":         true,
	"cached":         true,
	"reasoning":      true,
	"cache_creation": true,
}

// GetPricing handles GET /api/pricing.
func (h *Handlers) GetPricing(ctx *fasthttp.RequestCtx) {
	pricing, err := h.resolver.Merged()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "pricing")
		return
	}
	writeData(ctx, fasthttp.StatusOK, pricingToSnakeCase(pricing))
}

// PatchPricing handles PATCH /api/pricing.
func (h *Handlers) PatchPricing(ctx *fasthttp.RequestCtx) {
	var body map[string]map[string]map[string]float64
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := validatePricingBody(body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	if err := h.resolver.Update(body); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update pricing")
		return
	}

	pricing, err := h.resolver.UserPricing()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "pricing")
		return
	}
	writeData(ctx, fasthttp.StatusOK, pricing)
}

// DeletePricing handles DELETE /api/pricing.
func (h *Handlers) DeletePricing(ctx *fasthttp.RequestCtx) {
	provider := string(ctx.QueryArgs().Peek("provider"))
	model := string(ctx.QueryArgs().Peek("model"))

	var err error
	if provider == "" {
		err = h.resolver.ResetAll()
	} else {
		err = h.resolver.Reset(provider, model)
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "reset pricing")
		return
	}

	pricing, err := h.resolver.UserPricing()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "pricing")
		return
	}
	writeData(ctx, fasthttp.StatusOK, pricing)
}

func pricingToSnakeCase(pricing map[string]map[string]usage.Pricing) map[string]map[string]map[string]float64 {
	out := make(map[string]map[string]map[string]float64, len(pricing))
	for provider, models := range pricing {
		mout := make(map[string]map[string]float64, len(models))
		for model, p := range models {
			mout[model] = map[string]float64{
				"input":          p.Input,
				"output":         p.Output,
				"cached":         p.Cached,
				"reasoning":      p.Reasoning,
				"cache_creation": p.CacheCreation,
			}
		}
		out[provider] = mout
	}
	return out
}

func validatePricingBody(body map[string]map[string]map[string]float64) error {
	if body == nil {
		return fmt.Errorf("invalid pricing data format")
	}
	for provider, models := range body {
		if models == nil {
			return fmt.Errorf("invalid pricing for provider: %s", provider)
		}
		for model, rates := range body[provider] {
			if rates == nil {
				return fmt.Errorf("invalid pricing for model: %s/%s", provider, model)
			}
			for field, value := range rates {
				if !validPricingFields[field] {
					return fmt.Errorf("invalid pricing field: %s for %s/%s", field, provider, model)
				}
				if value < 0 {
					return fmt.Errorf("invalid pricing value for %s in %s/%s: must be non-negative number", field, provider, model)
				}
			}
		}
	}
	return nil
}
