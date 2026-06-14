package inference

import (
	"fmt"
	"sync"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// KeyResolver resolves a provider key and any provider-specific data.
type KeyResolver interface {
	ResolveKey(providerID string) (schemas.Key, map[string]string, error)
}

// Router resolves a model name to a provider and API key using the static
// model catalog and an optional key resolver (wired from the credential store).
type Router struct {
	registry    *translation.Registry
	providers   map[string]schemas.Provider
	mu           sync.RWMutex
	keyResolver  KeyResolver
	aliasStore   AliasStore
	nodeResolver NodeResolver
}

// NewRouter creates a router with all providers wired in.
func NewRouter(reg *translation.Registry) *Router {
	return &Router{
		registry:  reg,
		providers: make(map[string]schemas.Provider),
	}
}

// SetKeyResolver sets an optional key resolver. When non-nil, Resolve consults
// it instead of returning the default empty key. nil leaves behavior unchanged.
func (r *Router) SetKeyResolver(resolver KeyResolver) {
	r.mu.Lock()
	r.keyResolver = resolver
	r.mu.Unlock()
}

// SetAliasStore sets an optional alias store. When non-nil, Resolve expands
// user-defined model aliases before catalog lookup. nil leaves behavior unchanged.
func (r *Router) SetAliasStore(st AliasStore) {
	r.mu.Lock()
	r.aliasStore = st
	r.mu.Unlock()
}

// SetNodeResolver sets an optional provider-node resolver. When non-nil, Resolve
// consults it FIRST: a model whose prefix matches a registered node prefix routes
// to that node's provider + base URL, overriding static alias/catalog resolution
// (PAR-ROUTE-009/040). nil leaves behavior byte-identical to the static chain.
func (r *Router) SetNodeResolver(nr NodeResolver) {
	r.mu.Lock()
	r.nodeResolver = nr
	r.mu.Unlock()
}

// Resolve returns the provider and key for a given model request.
// It looks up the model in the static catalog, builds the provider, and
// resolves the key via the configured key resolver when one is present.
func (r *Router) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	// Step 0 (additive): provider-node prefix override. A model "prefix/bare"
	// whose prefix matches a registered node routes to that node's provider +
	// base URL, BEFORE static alias/catalog resolution (PAR-ROUTE-009/040). When
	// no node resolver is set or no prefix matches, behavior is unchanged.
	r.mu.RLock()
	nodeResolver := r.nodeResolver
	r.mu.RUnlock()
	if nodeResolver != nil {
		if prefix, _ := ParseModelPrefix(model); prefix != "" {
			if providerID, baseURL, apiType, ok := nodeResolver.ResolveByPrefix(prefix); ok {
				return r.resolveNode(providerID, baseURL, apiType)
			}
		}
	}

	r.mu.RLock()
	aliasStore := r.aliasStore
	r.mu.RUnlock()
	if aliasStore != nil {
		model = ResolveModelAlias(aliasStore, model)
	}

	providerID, ok := providerForModel(model)
	if !ok {
		return nil, schemas.Key{}, fmt.Errorf("no provider available for model %q", model)
	}

	p, err := r.providerForID(providerID)
	if err != nil {
		return nil, schemas.Key{}, fmt.Errorf("resolve %q: %w", model, err)
	}

	r.mu.RLock()
	resolver := r.keyResolver
	r.mu.RUnlock()

	if resolver != nil {
		key, psd, err := resolver.ResolveKey(providerID)
		if err != nil {
			return nil, schemas.Key{}, fmt.Errorf("resolve key for %q: %w", model, err)
		}
		key.Provider = providerID
		key.ProviderSpecificData = psd
		return p, key, nil
	}

	return p, schemas.Key{
		ID:       providerID + "-default",
		Provider: providerID,
		Value:    "",
	}, nil
}

// resolveNode returns the provider and key for a node-routed model. The node
// provider is built once (pointed at the node base URL) and cached by the node's
// provider id; the key is resolved via the configured key resolver when present
// (so a provisioned node connection's credential is used), else a default key.
func (r *Router) resolveNode(providerID, baseURL, apiType string) (schemas.Provider, schemas.Key, error) {
	r.mu.RLock()
	p, ok := r.providers[providerID]
	r.mu.RUnlock()
	if !ok {
		built := buildNodeProvider(providerID, baseURL, apiType)
		r.mu.Lock()
		if existing, exists := r.providers[providerID]; exists {
			p = existing
		} else {
			r.providers[providerID] = built
			p = built
		}
		r.mu.Unlock()
	}

	r.mu.RLock()
	resolver := r.keyResolver
	r.mu.RUnlock()
	if resolver != nil {
		key, psd, err := resolver.ResolveKey(providerID)
		if err != nil {
			return nil, schemas.Key{}, fmt.Errorf("resolve key for node %q: %w", providerID, err)
		}
		key.Provider = providerID
		key.ProviderSpecificData = psd
		return p, key, nil
	}
	return p, schemas.Key{
		ID:       providerID + "-default",
		Provider: providerID,
		Value:    "",
	}, nil
}

// providerForID returns a cached provider instance for the given provider ID,
// creating it lazily on first access.
func (r *Router) providerForID(providerID string) (schemas.Provider, error) {
	r.mu.RLock()
	if p, ok := r.providers[providerID]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	p, err := buildProvider(providerID, r.registry)
	if err != nil {
		return nil, fmt.Errorf("build provider %q: %w", providerID, err)
	}

	r.mu.Lock()
	if existing, ok := r.providers[providerID]; ok {
		r.mu.Unlock()
		return existing, nil
	}
	r.providers[providerID] = p
	r.mu.Unlock()
	return p, nil
}

// ResolveForModel is an alias for Resolve that accepts a full ChatRequest.
func (r *Router) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	return r.Resolve(req.Model)
}

// ErrNoProvider is returned when no provider can handle a request.
var ErrNoProvider = fmt.Errorf("no provider available for model")
