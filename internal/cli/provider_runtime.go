package cli

import (
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/azure"
	"github.com/bloodf/g0router/internal/providers/bedrock"
	"github.com/bloodf/g0router/internal/providers/cloudflare"
	"github.com/bloodf/g0router/internal/providers/cohere"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/mistral"
	"github.com/bloodf/g0router/internal/providers/ollama"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
	"github.com/bloodf/g0router/internal/providers/replicate"
	"github.com/bloodf/g0router/internal/providers/vertex"
	"github.com/bloodf/g0router/internal/providers/xiaomi"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/bloodf/g0router/internal/store"
)

func newDefaultInferenceEngine(s *store.Store) *proxy.Engine {
	engine := proxy.NewEngine(s)
	registerOAuthRefreshers(engine)
	engine.Register(openai.New(""))
	engine.Register(anthropic.New(""))
	engine.Register(gemini.New(""))
	engine.Register(azure.New("", ""))
	engine.Register(bedrock.New(""))
	engine.Register(cloudflare.New(""))
	engine.Register(vertex.New("", vertex.Config{
		ProjectID: envString("VERTEX_PROJECT_ID", ""),
		Location:  envString("VERTEX_LOCATION", "us-central1"),
	}))
	registerOpenAICompatible(engine)
	registerProvider(engine, func() (providers.Provider, error) {
		return mistral.NewDefault()
	})
	registerProvider(engine, func() (providers.Provider, error) {
		return cohere.NewDefault()
	})
	registerProvider(engine, func() (providers.Provider, error) {
		return ollama.NewDefault()
	})
	registerProvider(engine, func() (providers.Provider, error) {
		return replicate.NewDefault()
	})
	engine.Register(xiaomi.NewDefault())
	return engine
}

func registerOAuthRefreshers(engine *proxy.Engine) {
	for _, factory := range oauthFlowFactories() {
		flow := factory()
		refresher, ok := flow.(oauth.RefreshableFlow)
		if !ok {
			continue
		}
		engine.RegisterOAuthRefresher(flow.ProviderID(), refresher)
	}
}

func registerOpenAICompatible(engine *proxy.Engine) {
	for _, provider := range []providers.ModelProvider{
		providers.ProviderGroq,
		providers.ProviderCerebras,
		providers.ProviderPerplexity,
		providers.ProviderFireworks,
		providers.ProviderTogether,
		providers.ProviderNVIDIA,
		providers.ProviderDeepSeek,
		providers.ProviderOpenRouter,
		providers.ProviderHuggingFace,
		providers.ProviderNebius,
		providers.ProviderMiniMax,
		providers.ProviderQwen,
		providers.ProviderXAI,
		providers.ProviderVercelGateway,
		providers.ProviderGitHubCopilot,
		providers.ProviderAlibaba,
		providers.ProviderKimi,
		providers.ProviderQianfan,
		providers.ProviderZhipu,
		providers.ProviderLiteLLM,
		providers.ProviderVLLM,
		providers.ProviderLMStudio,
		providers.ProviderOpenCode,
		providers.ProviderKilo,
	} {
		provider := provider
		registerProvider(engine, func() (providers.Provider, error) {
			return openaicompat.NewDefault(provider)
		})
	}
}

func registerProvider(engine *proxy.Engine, factory func() (providers.Provider, error)) {
	provider, err := factory()
	if err != nil {
		return
	}
	engine.Register(provider)
}
