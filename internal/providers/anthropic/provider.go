package anthropic

import (
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
)

// Provider implements the schemas.Provider interface for Anthropic.
type Provider struct {
	provider       schemas.ModelProvider
	baseURL        string
	client         *utils.ClientPool
	networkConfig  schemas.NetworkConfig
	errorConverter *ErrorConverter
}

// NewProvider creates a new Anthropic provider.
func NewProvider() *Provider {
	return &Provider{
		provider:       schemas.ProviderAnthropic,
		baseURL:        "https://api.anthropic.com",
		client:         utils.NewClientPool(),
		errorConverter: NewErrorConverter(),
	}
}

// GetProvider returns the provider identifier.
func (p *Provider) GetProvider() schemas.ModelProvider {
	return p.provider
}

// SetNetworkConfig updates the network configuration.
func (p *Provider) SetNetworkConfig(config schemas.NetworkConfig) {
	p.networkConfig = config
}
