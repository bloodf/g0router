package proxy

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/anthropic"
	"github.com/bloodf/g0router/internal/providers/azure"
	"github.com/bloodf/g0router/internal/providers/bedrock"
	"github.com/bloodf/g0router/internal/providers/cloudflare"
	"github.com/bloodf/g0router/internal/providers/gemini"
	"github.com/bloodf/g0router/internal/providers/gitlabduo"
	"github.com/bloodf/g0router/internal/providers/ollamacloud"
	"github.com/bloodf/g0router/internal/providers/openai"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
	"github.com/bloodf/g0router/internal/providers/replicate"
	"github.com/bloodf/g0router/internal/providers/vertex"
	"github.com/bloodf/g0router/internal/providers/xiaomi"
	"github.com/bloodf/g0router/internal/store"
)

func TestProviderWithProxyPoolMethods(t *testing.T) {
	pool := &store.ProxyPool{Protocol: "http", Host: "proxy", Port: 8080}

	cases := []struct {
		name string
		pc   proxyConfigurable
	}{
		{"anthropic", anthropic.New("")},
		{"azure", azure.New("", "")},
		{"bedrock", bedrock.New("")},
		{"cloudflare", cloudflare.New("")},
		{"gemini", gemini.New("")},
		{"gitlabduo", gitlabduo.NewDefault()},
		{"ollamacloud", mustOllamaCloudProvider()},
		{"openai", openai.New("")},
		{"openaicompat", mustOpenAICompatProvider()},
		{"replicate", replicate.NewDefault()},
		{"vertex", vertex.New("", vertex.Config{})},
		{"xiaomi", xiaomi.NewDefault()},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := c.pc.WithProxyPool(pool)
			if result == nil {
				t.Fatal("expected non-nil provider")
			}
			if result == c.pc.(providers.Provider) {
				t.Fatal("expected a new provider instance, not the same")
			}
		})
	}
}

func mustOllamaCloudProvider() proxyConfigurable {
	p, err := ollamacloud.NewDefault()
	if err != nil {
		panic(err)
	}
	return p
}

func mustOpenAICompatProvider() proxyConfigurable {
	p, err := openaicompat.New(openaicompat.Config{
		Provider: providers.ModelProvider("openai"),
		BaseURL:  "http://localhost:8080",
	})
	if err != nil {
		panic(err)
	}
	return p
}
