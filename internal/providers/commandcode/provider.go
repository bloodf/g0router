// Package commandcode implements the CommandCode provider adapter
// (w7-prov-special-a, PAR-PROV-040). CommandCode speaks a custom JSON request
// shape and an AI-SDK-v5 NDJSON/SSE event stream; both directions are handled
// by the EXISTING translation converters (openaiToCommandCodeRequest /
// commandcodeToOpenAIResponse, registry.go:169-170). This adapter is a thin
// HTTP layer that runs the request/response through the registry — it adds no
// new wire-format logic.
package commandcode

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// Provider implements the schemas.Provider interface for CommandCode.
type Provider struct {
	id            schemas.ModelProvider
	config        catalog.ProviderConfig
	registry      *translation.Registry
	client        *utils.ClientPool
	networkConfig schemas.NetworkConfig
}

// New creates a CommandCode provider. providerID must resolve to a catalog
// entry whose Format is "commandcode".
func New(providerID string, reg *translation.Registry) (*Provider, error) {
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	if cfg.Format != "commandcode" {
		return nil, fmt.Errorf("provider %s format is %q, expected commandcode", providerID, cfg.Format)
	}
	return &Provider{
		id:       schemas.ModelProvider(providerID),
		config:   cfg,
		registry: reg,
		client:   utils.NewClientPool(),
	}, nil
}

// GetProvider returns the provider identifier.
func (p *Provider) GetProvider() schemas.ModelProvider {
	return p.id
}

// SetNetworkConfig updates the network configuration and pushes any per-instance
// proxy override into the client pool.
func (p *Provider) SetNetworkConfig(config schemas.NetworkConfig) {
	p.networkConfig = config
	_ = p.client.SetProxyURL(config.ProxyURL)
}
