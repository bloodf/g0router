package mistral

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
)

const defaultBaseURL = "https://api.mistral.ai"

func New(baseURL string) (*openaicompat.Provider, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	provider, err := openaicompat.New(openaicompat.Config{
		Provider: providers.ProviderMistral,
		BaseURL:  baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("mistral provider: %w", err)
	}
	return provider, nil
}

func NewDefault() (*openaicompat.Provider, error) {
	return New("")
}
