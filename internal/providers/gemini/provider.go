package gemini

import (
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
)

// Provider implements the schemas.Provider interface for Google Gemini.
type Provider struct {
	provider       schemas.ModelProvider
	baseURL        string
	client         *utils.ClientPool
	networkConfig  schemas.NetworkConfig
	errorConverter *ErrorConverter
}

// NewProvider creates a new Gemini provider.
func NewProvider() *Provider {
	return &Provider{
		provider:       schemas.ProviderGemini,
		baseURL:        "https://generativelanguage.googleapis.com/v1beta",
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
