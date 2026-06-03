package handlers

import (
	"context"
	"fmt"

	providerinfo "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

type ManagementModelSource interface {
	ListModels(ctx context.Context) ([]providers.Model, error)
}

type providerResponse struct {
	ID                string                      `json:"id"`
	OMPID             string                      `json:"omp_id"`
	Router9ID         string                      `json:"router9_id"`
	BifrostID         string                      `json:"bifrost_id"`
	AuthTypes         []string                    `json:"auth_types"`
	OAuthProvider     string                      `json:"oauth_provider,omitempty"`
	Refresh           bool                        `json:"refresh"`
	RegisteredAdapter bool                        `json:"registered_adapter"`
	PublicInference   bool                        `json:"public_inference"`
	DirectDispatch    bool                        `json:"direct_dispatch"`
	Inference         bool                        `json:"inference"`
	Streaming         bool                        `json:"streaming"`
	ModelCatalog      bool                        `json:"model_catalog"`
	ListModels        bool                        `json:"list_models"`
	Quota             bool                        `json:"quota"`
	PublicStatus      providerinfo.ProviderStatus `json:"public_status"`
	Notes             string                      `json:"notes,omitempty"`
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

	models, err := source.ListModels(requestContext(ctx))
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
	matrix := providerinfo.ProviderMatrix().Entries()
	responses := make([]providerResponse, 0, len(matrix))
	for _, entry := range matrix {
		responses = append(responses, providerResponse{
			ID:                entry.G0RouterID,
			OMPID:             entry.OMPID,
			Router9ID:         entry.Router9ID,
			BifrostID:         entry.BifrostID,
			AuthTypes:         append([]string(nil), entry.AuthTypes...),
			OAuthProvider:     entry.OAuthProvider,
			Refresh:           entry.Refresh,
			RegisteredAdapter: entry.RegisteredAdapter,
			PublicInference:   entry.PublicInference,
			DirectDispatch:    entry.DirectDispatch,
			Inference:         entry.Inference,
			Streaming:         entry.Streaming,
			ModelCatalog:      entry.ModelCatalog,
			ListModels:        entry.ListModels,
			Quota:             entry.Quota,
			PublicStatus:      entry.PublicStatus,
			Notes:             entry.Notes,
		})
	}
	return responses
}
