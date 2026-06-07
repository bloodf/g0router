package handlers

import (
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type adminModelsStore interface {
	IsModelDisabled(provider, model string) (bool, error)
	GetPricingOverride(provider, model string) (store.PricingOverride, error)
}

type adminModelResponse struct {
	ID            string  `json:"id"`
	Provider      string  `json:"provider"`
	Name          string  `json:"name"`
	InputCost     float64 `json:"input_cost"`
	OutputCost    float64 `json:"output_cost"`
	ContextWindow int     `json:"context_window"`
	IsDisabled    bool    `json:"is_disabled"`
	IsCustom      bool    `json:"is_custom"`
}

func AdminModels(ctx *fasthttp.RequestCtx, engine InferenceEngine, s adminModelsStore) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	models, err := engine.ListModels(requestContext(ctx))
	if err != nil {
		log.Printf("list models: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list models")
		return
	}

	resp := make([]adminModelResponse, 0, len(models))
	for _, m := range models {
		provider := string(m.Provider)
		name := m.ID
		id := name
		if provider != "" {
			id = provider + "/" + name
		}

		item := adminModelResponse{
			ID:       id,
			Provider: provider,
			Name:     name,
			IsCustom: m.IsCustom,
		}

		if s != nil {
			disabled, err := s.IsModelDisabled(provider, name)
			if err != nil {
				log.Printf("check model disabled: %v", err)
			} else {
				item.IsDisabled = disabled
			}

			override, err := s.GetPricingOverride(provider, name)
			if err == nil {
				item.InputCost = override.InputCostPerToken
				item.OutputCost = override.OutputCostPerToken
			}
		}

		resp = append(resp, item)
	}

	writeJSON(ctx, fasthttp.StatusOK, resp)
}
