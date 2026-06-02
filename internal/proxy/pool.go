package proxy

import "github.com/bloodf/g0router/internal/providers"

type providerPool struct {
	providers map[providers.ModelProvider]providers.Provider
}

func newProviderPool() providerPool {
	return providerPool{providers: make(map[providers.ModelProvider]providers.Provider)}
}

func (p *providerPool) register(provider providers.Provider) {
	p.providers[provider.Name()] = provider
}

func (p *providerPool) get(name providers.ModelProvider) (providers.Provider, bool) {
	provider, ok := p.providers[name]
	return provider, ok
}
