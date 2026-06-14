// Package urltemplate implements the URL-template / runtime-URL-build openai
// providers (w7-prov-special-a): cloudflare-ai (PAR-PROV-033), azure
// (PAR-PROV-032), and xiaomi-tokenplan (PAR-PROV-047). These are plain OpenAI
// chat-completions at the WIRE level; what differs is that the endpoint URL is
// computed at request time from schemas.Key.ProviderSpecificData (account id /
// resource / deployment / region). The request/response bodies pass through
// unchanged.
//
// qoder (PAR-PROV-028) is intentionally NOT handled here: its endpoint requires
// opaque COSY signing (RSA+AES+MD5, ~17 Cosy-* headers) plus a proprietary body
// encoder that cannot be soundly reproduced from the reference — DEFERRED
// (ESC-A3).
package urltemplate

import (
	"fmt"
	"strings"

	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/utils"
	"github.com/bloodf/g0router/internal/schemas"
)

// xiaomiTokenplanRegions maps the region key to its base URL prefix
// (providers.js:447-451).
var xiaomiTokenplanRegions = map[string]string{
	"sgp": "https://token-plan-sgp.xiaomimimo.com/v1",
	"cn":  "https://token-plan-cn.xiaomimimo.com/v1",
	"ams": "https://token-plan-ams.xiaomimimo.com/v1",
}

const xiaomiTokenplanDefaultRegion = "sgp"

// Provider implements the schemas.Provider interface for the URL-template
// openai providers.
type Provider struct {
	id             schemas.ModelProvider
	config         catalog.ProviderConfig
	client         *utils.ClientPool
	networkConfig  schemas.NetworkConfig
	errorConverter *openai.ErrorConverter
	// urlOverride, when non-empty, replaces the computed URL entirely. It is a
	// test seam (set only by tests) so the round-trip can target an httptest
	// server while the URL-build logic is exercised separately by buildURL.
	urlOverride string
}

// New creates a URL-template provider for one of cloudflare-ai/azure/
// xiaomi-tokenplan. The provider must exist in the catalog with Format "openai".
func New(providerID string) (*Provider, error) {
	if !isURLTemplateProvider(providerID) {
		return nil, fmt.Errorf("provider %q is not a url-template provider", providerID)
	}
	cfg, ok := catalog.Lookup(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
	return &Provider{
		id:             schemas.ModelProvider(providerID),
		config:         cfg,
		client:         utils.NewClientPool(),
		errorConverter: openai.NewErrorConverter(),
	}, nil
}

// isURLTemplateProvider reports whether the provider id is handled by this
// adapter.
func isURLTemplateProvider(id string) bool {
	switch id {
	case "cloudflare-ai", "azure", "xiaomi-tokenplan":
		return true
	}
	return false
}

// IsURLTemplateProvider is the exported predicate used by the factory dispatch
// to route a Format:"openai" catalog provider to this adapter.
func IsURLTemplateProvider(id string) bool { return isURLTemplateProvider(id) }

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

// buildURL computes the request-time endpoint URL from the key's
// ProviderSpecificData. It returns an empty string when a required field is
// missing (the caller surfaces that as a request error).
func (p *Provider) buildURL(key schemas.Key, model string) string {
	psd := key.ProviderSpecificData
	switch p.id {
	case "cloudflare-ai":
		accountID := psd["accountId"]
		if accountID == "" {
			return ""
		}
		return strings.Replace(p.config.BaseURL, "{accountId}", accountID, 1)
	case "azure":
		endpoint := psd["azureEndpoint"]
		if endpoint == "" {
			endpoint = "https://api.openai.com"
		}
		endpoint = strings.TrimRight(endpoint, "/")
		apiVersion := psd["apiVersion"]
		if apiVersion == "" {
			apiVersion = "2024-10-01-preview"
		}
		deployment := psd["deployment"]
		if deployment == "" {
			deployment = model
		}
		if deployment == "" {
			deployment = "gpt-4"
		}
		return fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s", endpoint, deployment, apiVersion)
	case "xiaomi-tokenplan":
		region := psd["region"]
		base, ok := xiaomiTokenplanRegions[region]
		if !ok {
			base = xiaomiTokenplanRegions[xiaomiTokenplanDefaultRegion]
		}
		return base + "/chat/completions"
	default:
		return p.config.BaseURL
	}
}

// authHeaderName returns the auth header for the provider. azure uses the
// "api-key" header (azure.js:37); the others use the default bearer
// Authorization header.
func (p *Provider) authHeaderName() string {
	if p.id == "azure" {
		return "api-key"
	}
	return ""
}
