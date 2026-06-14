package generic

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
)

// Provider implements the schemas.Provider interface for generic OpenAI-compatible backends.
type Provider struct {
	id             schemas.ModelProvider
	config         catalog.ProviderConfig
	client         *utils.ClientPool
	networkConfig  schemas.NetworkConfig
	errorConverter *openai.ErrorConverter
}

// New creates a generic OpenAI-compatible provider for the given provider ID.
func New(providerID string) (*Provider, error) {
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	if cfg.Format != "openai" {
		return nil, fmt.Errorf("provider %s format %q is not supported by generic adapter", providerID, cfg.Format)
	}
	return &Provider{
		id:             schemas.ModelProvider(providerID),
		config:         cfg,
		client:         utils.NewClientPool(),
		errorConverter: openai.NewErrorConverter(),
	}, nil
}

// NewNode creates a generic OpenAI-compatible provider for a dynamic provider
// node (w7-platnodes). Unlike New, it is not catalog-bound: the node's base URL
// (the API root, e.g. https://host/v1) is supplied at runtime and the chat
// endpoint is appended. The id is the node's providers-row id. It is purely
// additive and changes no existing constructor or field.
func NewNode(id, baseURL string) *Provider {
	return &Provider{
		id: schemas.ModelProvider(id),
		config: catalog.ProviderConfig{
			Name:    id,
			BaseURL: strings.TrimRight(baseURL, "/") + "/chat/completions",
			Format:  "openai",
		},
		client:         utils.NewClientPool(),
		errorConverter: openai.NewErrorConverter(),
	}
}

// GetProvider returns the provider identifier.
func (p *Provider) GetProvider() schemas.ModelProvider {
	return p.id
}

// SetNetworkConfig updates the network configuration and pushes the per-instance
// proxy override into the client pool (PAR-PLAT-009). A SetProxyURL error is
// ignored: an invalid proxy URL falls back to the default (env-proxy) path.
func (p *Provider) SetNetworkConfig(config schemas.NetworkConfig) {
	p.networkConfig = config
	_ = p.client.SetProxyURL(config.ProxyURL)
}
