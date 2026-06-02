package handlers

import (
	"context"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

type ManagementModelSource interface {
	ListModels(ctx context.Context) ([]providers.Model, error)
}

type providerResponse struct {
	ID string `json:"id"`
}

func Providers(ctx *fasthttp.RequestCtx, source ManagementModelSource, providerID string) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	if providerID == "" {
		writeJSON(ctx, fasthttp.StatusOK, listResponse[providerResponse]{Data: knownProviders()})
		return
	}

	if source == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "model source unavailable")
		return
	}

	models, err := source.ListModels(context.Background())
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list models: %v", err))
		return
	}

	filtered := make([]providers.Model, 0)
	for _, model := range models {
		if string(model.Provider) == providerID {
			filtered = append(filtered, model)
		}
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[providers.Model]{Data: filtered})
}

func knownProviders() []providerResponse {
	providerIDs := []string{
		"anthropic",
		"azure",
		"bedrock",
		"cerebras",
		"cohere",
		"cursor",
		"deepseek",
		"fireworks",
		"gemini",
		"github-copilot",
		"groq",
		"huggingface",
		"mistral",
		"nebius",
		"nvidia",
		"ollama",
		"openai",
		"openrouter",
		"perplexity",
		"replicate",
		"together",
		"vertex",
		"xai",
	}
	providers := make([]providerResponse, 0, len(providerIDs))
	for _, id := range providerIDs {
		providers = append(providers, providerResponse{ID: id})
	}
	return providers
}
