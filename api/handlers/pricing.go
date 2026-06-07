package handlers

import (
	"encoding/json"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type pricingRequest struct {
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	InputCostPerToken  float64 `json:"input_cost_per_token"`
	OutputCostPerToken float64 `json:"output_cost_per_token"`
	// UI sends these aliases
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
}

type pricingOverrideResponse struct {
	ID         string  `json:"id"`
	Provider   string  `json:"provider"`
	Model      string  `json:"model"`
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
}

type pricingStore interface {
	ListPricingOverrides() ([]store.PricingOverride, error)
	SetPricingOverride(store.PricingOverride) error
	DeletePricingOverride(string, string) error
}

func newPricingOverrideResponse(o store.PricingOverride) pricingOverrideResponse {
	return pricingOverrideResponse{
		ID:         o.Provider + "/" + o.Model,
		Provider:   o.Provider,
		Model:      o.Model,
		InputCost:  o.InputCostPerToken * 1_000_000,
		OutputCost: o.OutputCostPerToken * 1_000_000,
	}
}

func Pricing(ctx *fasthttp.RequestCtx, s pricingStore, provider, model string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		overrides, err := s.ListPricingOverrides()
		if err != nil {
			log.Printf("list pricing overrides: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to list pricing overrides")
			return
		}
		views := make([]pricingOverrideResponse, len(overrides))
		for i, o := range overrides {
			views[i] = newPricingOverrideResponse(o)
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[pricingOverrideResponse]{Data: views})
	case fasthttp.MethodPost:
		override, ok := decodePricingRequest(ctx, "", "")
		if !ok {
			return
		}
		if err := s.SetPricingOverride(override); err != nil {
			log.Printf("set pricing override (create): %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to set pricing override")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, newPricingOverrideResponse(override))
	case fasthttp.MethodPut:
		if provider == "" || model == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "provider and model required")
			return
		}
		override, ok := decodePricingRequest(ctx, provider, model)
		if !ok {
			return
		}
		if err := s.SetPricingOverride(override); err != nil {
			log.Printf("set pricing override (update): %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to set pricing override")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newPricingOverrideResponse(override))
	case fasthttp.MethodDelete:
		if provider == "" || model == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "provider and model required")
			return
		}
		if err := s.DeletePricingOverride(provider, model); err != nil {
			writeStoreError(ctx, "delete pricing override", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func decodePricingRequest(ctx *fasthttp.RequestCtx, provider, model string) (store.PricingOverride, bool) {
	var req pricingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return store.PricingOverride{}, false
	}
	if req.InputCostPerToken == 0 && req.InputCost != 0 {
		req.InputCostPerToken = req.InputCost / 1_000_000
	}
	if req.OutputCostPerToken == 0 && req.OutputCost != 0 {
		req.OutputCostPerToken = req.OutputCost / 1_000_000
	}
	if provider != "" {
		req.Provider = provider
	}
	if model != "" {
		req.Model = model
	}
	if req.Provider == "" || req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "provider and model are required")
		return store.PricingOverride{}, false
	}
	return store.PricingOverride{
		Provider:           req.Provider,
		Model:              req.Model,
		InputCostPerToken:  req.InputCostPerToken,
		OutputCostPerToken: req.OutputCostPerToken,
	}, true
}
