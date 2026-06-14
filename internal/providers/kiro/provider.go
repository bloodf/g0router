package kiro

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// Provider implements the schemas.Provider interface for Kiro
// (AWS CodeWhisperer). The request body is built by the EXISTING converter
// (buildKiroPayload, registry.go:171) and the binary eventstream response is
// decoded by this package's DecodeEventStream, then translated to OpenAI chunks
// by the EXISTING converter (kiroToOpenAIResponse, registry.go:172).
type Provider struct {
	id            schemas.ModelProvider
	config        catalog.ProviderConfig
	registry      *translation.Registry
	client        *utils.ClientPool
	networkConfig schemas.NetworkConfig
	// urlOverride, when non-empty, replaces the catalog base URL entirely. It is
	// a test seam (set only by tests) so the round-trip can target an httptest
	// server.
	urlOverride string
}

// New creates a Kiro provider. providerID must resolve to a catalog entry whose
// Format is "kiro".
func New(providerID string, reg *translation.Registry) (*Provider, error) {
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	if cfg.Format != "kiro" {
		return nil, fmt.Errorf("provider %s format is %q, expected kiro", providerID, cfg.Format)
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
