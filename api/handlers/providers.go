package handlers

import (
	"context"
	"errors"
	"log"

	providerinfo "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type ManagementModelSource interface {
	ListModels(ctx context.Context) ([]providers.Model, error)
}

type ProviderAdapterSource interface {
	GetProvider(name providers.ModelProvider) (providers.Provider, bool)
}

type providerDetailStore interface {
	ListConnections() ([]*store.Connection, error)
	GetConnectionProxyPoolID(connectionID string) (*string, error)
}

type suggestedModelsStore interface {
	GetActiveConnections(provider string) ([]*store.Connection, error)
}

type providerResponse struct {
	ID                string                      `json:"id"`
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

type providerDetailResponse struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	MatrixInfo      providerResponse `json:"matrix_info"`
	ConnectionCount int              `json:"connection_count"`
	HealthStatus    string           `json:"health_status"`
	Models          []providers.Model `json:"models"`
}

type suggestedModelResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
	providerID = providerinfo.CanonicalProviderID(providerID)
	entry, ok := providerinfo.ProviderMatrix().Provider(providerID)
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}
	if !entry.PublicInference || !entry.DirectDispatch {
		writeError(ctx, fasthttp.StatusNotFound, "provider inference unavailable")
		return
	}

	if source == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "model source unavailable")
		return
	}

	models, err := source.ListModels(requestContext(ctx))
	if err != nil {
		log.Printf("list models (provider %s): %v", providerID, err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list models")
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

func ProviderDetail(ctx *fasthttp.RequestCtx, s providerDetailStore, source ManagementModelSource, providerID string) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	providerID = providerinfo.CanonicalProviderID(providerID)
	entry, ok := providerinfo.ProviderMatrix().Provider(providerID)
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}

	connections, err := s.ListConnections()
	if err != nil {
		log.Printf("list connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list connections")
		return
	}

	var connectionCount int
	var hasActive bool
	for _, conn := range connections {
		if conn.Provider == providerID {
			connectionCount++
			if conn.IsActive {
				hasActive = true
			}
		}
	}

	healthStatus := "unhealthy"
	if hasActive {
		healthStatus = "healthy"
	} else if connectionCount == 0 {
		healthStatus = "unknown"
	}

	var models []providers.Model
	if source != nil {
		allModels, err := source.ListModels(requestContext(ctx))
		if err != nil {
			log.Printf("list models: %v", err)
		} else {
			for _, m := range allModels {
				if string(m.Provider) == providerID {
					models = append(models, m)
				}
			}
		}
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"data": providerDetailResponse{
			ID:   entry.G0RouterID,
			Name: entry.G0RouterID,
			MatrixInfo: providerResponse{
				ID:                entry.G0RouterID,
				AuthTypes:         copyStringSlice(entry.AuthTypes),
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
			},
			ConnectionCount: connectionCount,
			HealthStatus:    healthStatus,
			Models:          models,
		},
	})
}

func ProviderSuggestedModels(ctx *fasthttp.RequestCtx, s suggestedModelsStore, adapterSource ProviderAdapterSource, providerID string) {
	if string(ctx.Method()) != fasthttp.MethodGet {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	providerID = providerinfo.CanonicalProviderID(providerID)
	_, ok := providerinfo.ProviderMatrix().Provider(providerID)
	if !ok {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}

	connections, err := s.GetActiveConnections(providerID)
	if err != nil {
		log.Printf("get active connections: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to get connections")
		return
	}
	if len(connections) == 0 {
		writeError(ctx, fasthttp.StatusBadRequest, "no active connections for provider")
		return
	}

	if adapterSource == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "adapter source unavailable")
		return
	}

	adapter, ok := adapterSource.GetProvider(providers.ModelProvider(providerID))
	if !ok {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "provider adapter unavailable")
		return
	}

	conn := connections[0]
	key := providers.Key{
		Provider: providers.ModelProvider(providerID),
		ConnID:   conn.ID,
		AuthType: string(conn.AuthType),
	}
	if conn.APIKey != nil {
		key.Value = *conn.APIKey
	} else if conn.AccessToken != nil {
		key.Value = *conn.AccessToken
	}
	if conn.AccountID != nil {
		key.AccountID = *conn.AccountID
	}

	modelList, err := adapter.ListModels(requestContext(ctx), key)
	if err != nil {
		if errors.Is(err, providers.ErrListModelsUnsupported) {
			writeJSON(ctx, fasthttp.StatusOK, listResponse[suggestedModelResponse]{Data: []suggestedModelResponse{}})
			return
		}
		log.Printf("list models (provider %s): %v", providerID, err)
		writeError(ctx, fasthttp.StatusBadGateway, "upstream model list failed")
		return
	}

	responses := make([]suggestedModelResponse, 0, len(modelList))
	for _, m := range modelList {
		responses = append(responses, suggestedModelResponse{
			ID:   m.ID,
			Name: m.ID,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[suggestedModelResponse]{Data: responses})
}

func knownProviders() []providerResponse {
	matrix := providerinfo.ProviderMatrix().Entries()
	responses := make([]providerResponse, 0, len(matrix))
	for _, entry := range matrix {
		responses = append(responses, providerResponse{
			ID:                entry.G0RouterID,
			AuthTypes:         copyStringSlice(entry.AuthTypes),
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

func copyStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}
