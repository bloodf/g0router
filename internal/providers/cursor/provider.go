package cursor

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// cursorChatPath is the connect-RPC method path appended to the base URL
// (providers.js:210 chatPath).
const cursorChatPath = "/aiserver.v1.ChatService/StreamUnifiedChatWithTools"

// Provider implements the schemas.Provider interface for Cursor.
type Provider struct {
	id            schemas.ModelProvider
	config        catalog.ProviderConfig
	registry      *translation.Registry
	client        *utils.ClientPool
	networkConfig schemas.NetworkConfig
	// urlOverride, when non-empty, replaces the computed chat URL entirely. It is
	// a test seam (set only by tests) so the round-trip can target an httptest
	// server.
	urlOverride string
}

// New creates a Cursor provider. providerID must resolve to a catalog entry
// whose Format is "cursor".
func New(providerID string, reg *translation.Registry) (*Provider, error) {
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	if cfg.Format != "cursor" {
		return nil, fmt.Errorf("provider %s format is %q, expected cursor", providerID, cfg.Format)
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

// chatURL resolves the request URL. The urlOverride test seam takes precedence
// over the catalog base URL + chat path.
func (p *Provider) chatURL() string {
	if p.urlOverride != "" {
		return p.urlOverride
	}
	return p.config.BaseURL + cursorChatPath
}
