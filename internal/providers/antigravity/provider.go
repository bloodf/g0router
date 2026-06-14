// Package antigravity implements the Antigravity provider adapter
// (w7-prov-special-b, PAR-PROV-020). Antigravity is a single Gemini-on-Cloud-Code
// wire format (POST /v1internal:streamGenerateContent) that fronts several
// backends (gemini / claude / gpt-oss) selected per model; the converter pair is
// chosen by the EXISTING translation converters (openaiToAntigravityRequest /
// geminiToOpenAIResponse, registry.go:158-164). This adapter supplies the
// genuinely-new executor behavior: the v1internal URL build with sandbox fallback
// ordering, the bearer/UA/source headers, and the PAR-MCP-060 unavailable-tool
// ride-along (cloakTools, filter.go).
package antigravity

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// antigravityBaseURLs is the fixed fallback host list (providers.js:106-108):
// the primary daily-cloudcode-pa host followed by the sandbox host.
var antigravityBaseURLs = []string{
	"https://daily-cloudcode-pa.googleapis.com",
	"https://daily-cloudcode-pa.sandbox.googleapis.com",
}

// internalRequestHeader is the x-request-source header Antigravity expects
// (appConstants.js INTERNAL_REQUEST_HEADER = { x-request-source: local }).
const (
	internalRequestHeaderName  = "x-request-source"
	internalRequestHeaderValue = "local"
)

// Provider implements the schemas.Provider interface for Antigravity.
type Provider struct {
	id            schemas.ModelProvider
	config        catalog.ProviderConfig
	registry      *translation.Registry
	client        *utils.ClientPool
	networkConfig schemas.NetworkConfig
	// urlOverride, when non-empty, replaces the computed v1internal URL entirely.
	// It is a test seam (set only by tests) so the round-trip can target an
	// httptest server while buildURL is exercised separately.
	urlOverride string
}

// New creates an Antigravity provider. providerID must resolve to a catalog
// entry whose Format is "antigravity".
func New(providerID string, reg *translation.Registry) (*Provider, error) {
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	if cfg.Format != "antigravity" {
		return nil, fmt.Errorf("provider %s format is %q, expected antigravity", providerID, cfg.Format)
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

// backendForModel reports which backend a model routes to behind the single
// Antigravity provider: claude models → "claude", gpt-oss models → "gpt-oss",
// everything else → "gemini" (executors/antigravity.js dispatches by
// isClaudeModel; the providerModels.js:84-94 block spans all three backends).
func backendForModel(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "claude"):
		return "claude"
	case strings.Contains(m, "gpt-oss"):
		return "gpt-oss"
	default:
		return "gemini"
	}
}

// fallbackCount returns the number of fallback URLs to try.
func (p *Provider) fallbackCount() int {
	return len(antigravityBaseURLs)
}

// buildURL constructs the v1internal endpoint URL for the given fallback index
// and stream mode (antigravity.js:26-31). The urlOverride test seam, when set,
// replaces the host entirely while preserving the action path.
func (p *Provider) buildURL(urlIndex int, stream bool) string {
	base := antigravityBaseURLs[0]
	if urlIndex >= 0 && urlIndex < len(antigravityBaseURLs) {
		base = antigravityBaseURLs[urlIndex]
	}
	if p.urlOverride != "" {
		base = p.urlOverride
	}
	action := "generateContent"
	if stream {
		action = "streamGenerateContent?alt=sse"
	}
	return fmt.Sprintf("%s/v1internal:%s", base, action)
}
