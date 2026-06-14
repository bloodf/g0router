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
	// chatURL, when non-empty, is used VERBATIM as the messages endpoint
	// instead of baseURL + "/v1/messages". It is set only by NewForProvider
	// for the w7-prov-special-a claude-format providers, whose catalog base
	// URL is already the full .../v1/messages endpoint (+ ?beta=true suffix).
	// Empty for the default NewProvider() path (existing behavior unchanged).
	chatURL string
	// betaHeader, when non-empty, is sent as the Anthropic-Beta header. Set
	// only by NewForProvider (CLAUDE_API_HEADERS); empty for NewProvider().
	betaHeader string
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

// NewForProvider creates an Anthropic-Messages-wire provider for a
// claude-format catalog provider (w7-prov-special-a: glm/kimi/minimax/
// minimax-cn). It is purely additive and does NOT change NewProvider().
//
// baseURL is the catalog base URL, which for these providers is already the
// full messages endpoint (e.g. https://api.z.ai/api/anthropic/v1/messages).
// The constructor appends 9router's "?beta=true" suffix and sets the
// CLAUDE_API_HEADERS Anthropic-Beta flag; auth remains x-api-key
// (setAuthHeader). The OpenAI<->Anthropic request/response converters are
// reused unchanged.
func NewForProvider(id, baseURL string) *Provider {
	return &Provider{
		provider:       schemas.ModelProvider(id),
		baseURL:        baseURL,
		chatURL:        baseURL + "?beta=true",
		betaHeader:     "claude-code-20250219,interleaved-thinking-2025-05-14",
		client:         utils.NewClientPool(),
		errorConverter: NewErrorConverter(),
	}
}

// messagesURL returns the endpoint for the Messages API. For the default
// provider it is baseURL + "/v1/messages"; claude-format providers
// (NewForProvider) override it with the full catalog URL + "?beta=true".
func (p *Provider) messagesURL() string {
	if p.chatURL != "" {
		return p.chatURL
	}
	return p.baseURL + "/v1/messages"
}

// GetProvider returns the provider identifier.
func (p *Provider) GetProvider() schemas.ModelProvider {
	return p.provider
}

// SetNetworkConfig updates the network configuration.
func (p *Provider) SetNetworkConfig(config schemas.NetworkConfig) {
	p.networkConfig = config
}
