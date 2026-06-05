package handlers

import (
	"context"
	"log"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

type InferenceEngine interface {
	Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error)
	DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error)
	ListModels(ctx context.Context) ([]providers.Model, error)
}

type modelsResponse struct {
	Object string            `json:"object"`
	Data   []providers.Model `json:"data"`
}

func Models(ctx *fasthttp.RequestCtx, engine InferenceEngine) {
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

	writeJSON(ctx, fasthttp.StatusOK, modelsResponse{
		Object: "list",
		Data:   models,
	})
}
