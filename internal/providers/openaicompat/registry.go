package openaicompat

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

func DefaultConfigs() map[providers.ModelProvider]Config {
	return map[providers.ModelProvider]Config{
		providers.ProviderGroq:        {Provider: providers.ProviderGroq, BaseURL: "https://api.groq.com/openai"},
		providers.ProviderCerebras:    {Provider: providers.ProviderCerebras, BaseURL: "https://api.cerebras.ai"},
		providers.ProviderPerplexity:  {Provider: providers.ProviderPerplexity, BaseURL: "https://api.perplexity.ai"},
		providers.ProviderFireworks:   {Provider: providers.ProviderFireworks, BaseURL: "https://api.fireworks.ai/inference"},
		providers.ProviderTogether:    {Provider: providers.ProviderTogether, BaseURL: "https://api.together.xyz"},
		providers.ProviderNVIDIA:      {Provider: providers.ProviderNVIDIA, BaseURL: "https://integrate.api.nvidia.com"},
		providers.ProviderDeepSeek:    {Provider: providers.ProviderDeepSeek, BaseURL: "https://api.deepseek.com"},
		providers.ProviderOpenRouter:  {Provider: providers.ProviderOpenRouter, BaseURL: "https://openrouter.ai/api"},
		providers.ProviderHuggingFace: {Provider: providers.ProviderHuggingFace, BaseURL: "https://api-inference.huggingface.co"},
		providers.ProviderNebius:      {Provider: providers.ProviderNebius, BaseURL: "https://api.studio.nebius.ai"},
		providers.ProviderMiniMax:     {Provider: providers.ProviderMiniMax, BaseURL: "https://api.minimax.io/v1"},
		providers.ProviderQwen:        {Provider: providers.ProviderQwen, BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
		providers.ProviderXAI:         {Provider: providers.ProviderXAI, BaseURL: "https://api.x.ai/v1"},
	}
}

func NewDefault(provider providers.ModelProvider) (*Provider, error) {
	config, ok := DefaultConfigs()[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrUnknownProvider)
	}
	return New(config)
}
