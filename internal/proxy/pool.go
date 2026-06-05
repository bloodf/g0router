package proxy

import (
	"sort"
	"sync"

	"github.com/bloodf/g0router/internal/providers"
)

type providerPool struct {
	mu        sync.RWMutex
	providers map[providers.ModelProvider]providers.Provider
}

func newProviderPool() providerPool {
	return providerPool{providers: make(map[providers.ModelProvider]providers.Provider)}
}

func (p *providerPool) register(provider providers.Provider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.providers[provider.Name()] = provider
}

func (p *providerPool) get(name providers.ModelProvider) (providers.Provider, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	provider, ok := p.providers[name]
	return provider, ok
}

func (p *providerPool) names() []providers.ModelProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	names := make([]providers.ModelProvider, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return names[i] < names[j]
	})
	return names
}
