package openaicompat

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

func DefaultConfigs() map[providers.ModelProvider]Config {
	return map[providers.ModelProvider]Config{
		providers.ProviderGroq:          {Provider: providers.ProviderGroq, BaseURL: "https://api.groq.com/openai"},
		providers.ProviderCerebras:      {Provider: providers.ProviderCerebras, BaseURL: "https://api.cerebras.ai"},
		providers.ProviderPerplexity:    {Provider: providers.ProviderPerplexity, BaseURL: "https://api.perplexity.ai"},
		providers.ProviderFireworks:     {Provider: providers.ProviderFireworks, BaseURL: "https://api.fireworks.ai/inference"},
		providers.ProviderTogether:      {Provider: providers.ProviderTogether, BaseURL: "https://api.together.xyz"},
		providers.ProviderNVIDIA:        {Provider: providers.ProviderNVIDIA, BaseURL: "https://integrate.api.nvidia.com"},
		providers.ProviderDeepSeek:      {Provider: providers.ProviderDeepSeek, BaseURL: "https://api.deepseek.com"},
		providers.ProviderOpenRouter:    {Provider: providers.ProviderOpenRouter, BaseURL: "https://openrouter.ai/api"},
		providers.ProviderHuggingFace:   {Provider: providers.ProviderHuggingFace, BaseURL: "https://router.huggingface.co"},
		providers.ProviderKimi:          {Provider: providers.ProviderKimi, BaseURL: "https://api.moonshot.ai/v1"},
		providers.ProviderNebius:        {Provider: providers.ProviderNebius, BaseURL: "https://api.tokenfactory.nebius.com"},
		providers.ProviderMiniMax:       {Provider: providers.ProviderMiniMax, BaseURL: "https://api.minimax.io/v1"},
		providers.ProviderQwen:          {Provider: providers.ProviderQwen, BaseURL: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"},
		providers.ProviderXAI:           {Provider: providers.ProviderXAI, BaseURL: "https://api.x.ai/v1"},
		providers.ProviderVercelGateway: {Provider: providers.ProviderVercelGateway, BaseURL: "https://ai-gateway.vercel.sh/v1"},
		providers.ProviderGitHubCopilot: {Provider: providers.ProviderGitHubCopilot, BaseURL: "https://api.githubcopilot.com", Headers: map[string]string{"User-Agent": "opencode/1.3.15"}},
		providers.ProviderAlibaba:       {Provider: providers.ProviderAlibaba, BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
		providers.ProviderQianfan:       {Provider: providers.ProviderQianfan, BaseURL: "https://api.baiduqianfan.ai/v1"},
		providers.ProviderZhipu:         {Provider: providers.ProviderZhipu, BaseURL: "https://api.z.ai/api/paas/v4", ChatCompletionsPath: "/chat/completions"},
		providers.ProviderLiteLLM:       {Provider: providers.ProviderLiteLLM, BaseURL: "http://localhost:4000"},
		providers.ProviderVLLM:          {Provider: providers.ProviderVLLM, BaseURL: "http://localhost:8000/v1"},
		providers.ProviderLMStudio:      {Provider: providers.ProviderLMStudio, BaseURL: "http://localhost:1234/v1"},
		providers.ProviderOpenCode:      {Provider: providers.ProviderOpenCode, BaseURL: "https://opencode.ai/zen/v1"},
		providers.ProviderKilo:          {Provider: providers.ProviderKilo, BaseURL: "https://api.kilo.ai/api/gateway"},
	}
}

func NewDefault(provider providers.ModelProvider) (*Provider, error) {
	config, ok := DefaultConfigs()[provider]
	if !ok {
		return nil, fmt.Errorf("%s: %w", provider, ErrUnknownProvider)
	}
	return New(config)
}
