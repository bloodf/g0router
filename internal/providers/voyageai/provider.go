// Package voyageai implements the voyage-ai embedding provider (w7-prov-media,
// PAR-PROV-066). voyage-ai is OpenAI-compatible at the wire level: a bearer-auth
// POST to a fixed /v1/embeddings endpoint with a plain {model,input,...} body,
// decoded straight into schemas.EmbeddingResponse (the reference normalize is
// identity — embeddingProviders/openai.js:8,37, index.js:6-8).
//
// Only the Embedding method is real; every other schemas.Provider method is a
// 501 stub (stubs.go) — voyage-ai exposes no chat/image/audio surface.
package voyageai

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
)

// Provider implements the schemas.Provider interface for voyage-ai.
type Provider struct {
	id             schemas.ModelProvider
	config         catalog.ProviderConfig
	client         *utils.ClientPool
	networkConfig  schemas.NetworkConfig
	errorConverter *openai.ErrorConverter
	// urlOverride, when non-empty, replaces the embeddings endpoint entirely. It
	// is a test seam (set only by tests) so the round-trip can target an httptest
	// server. Mirrors urltemplate/provider.go:43-46.
	urlOverride string
}

// New creates the voyage-ai embedding provider. The provider must be the
// catalog-registered voyage-ai entry with Format "openai".
func New(providerID string) (*Provider, error) {
	if providerID != "voyage-ai" {
		return nil, fmt.Errorf("provider %q is not voyage-ai", providerID)
	}
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	if cfg.Format != "openai" {
		return nil, fmt.Errorf("provider %s format %q is not supported by the voyage-ai adapter", providerID, cfg.Format)
	}
	return &Provider{
		id:             schemas.ModelProvider(providerID),
		config:         cfg,
		client:         utils.NewClientPool(),
		errorConverter: openai.NewErrorConverter(),
	}, nil
}

// GetProvider returns the provider identifier.
func (p *Provider) GetProvider() schemas.ModelProvider {
	return p.id
}

// SetNetworkConfig updates the network configuration and pushes any per-instance
// proxy override into the client pool (PAR-PLAT-009). A SetProxyURL error is
// ignored: an invalid proxy URL falls back to the default (env-proxy) path.
func (p *Provider) SetNetworkConfig(config schemas.NetworkConfig) {
	p.networkConfig = config
	_ = p.client.SetProxyURL(config.ProxyURL)
}

// embeddingsURL returns the request endpoint: the test override when set,
// otherwise the catalog BaseURL (which is already the full /v1/embeddings URL).
func (p *Provider) embeddingsURL() string {
	if p.urlOverride != "" {
		return p.urlOverride
	}
	return p.config.BaseURL
}
