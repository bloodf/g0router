package inference

import (
	"github.com/bloodf/g0router/internal/providers/generic"
	"github.com/bloodf/g0router/internal/schemas"
)

// NodeResolver resolves a routing prefix to a provider node's provider id, base
// URL, and api type. It is the inference-side seam for the provider-node
// prefix-routing engine (w7-platnodes, PAR-ROUTE-009/040). The implementation
// lives in internal/platform and is wired in server.go to avoid an
// inference→platform import cycle.
type NodeResolver interface {
	ResolveByPrefix(prefix string) (providerID, baseURL, apiType string, ok bool)
}

// buildNodeProvider constructs a provider instance pointed at a node's base URL.
// Both openai- and anthropic-compatible nodes are served through the generic
// OpenAI-compatible adapter pointed at the node base URL; the apiType is reserved
// for future adapter selection. It changes no existing provider constructor.
func buildNodeProvider(providerID, baseURL, apiType string) schemas.Provider {
	_ = apiType
	return generic.NewNode(providerID, baseURL)
}
