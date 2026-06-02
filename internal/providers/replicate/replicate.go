package replicate

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
)

const defaultBaseURL = "https://api.replicate.com"

func New(baseURL string) (*openaicompat.Provider, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	provider, err := openaicompat.New(openaicompat.Config{
		Provider: providers.ProviderReplicate,
		BaseURL:  baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("replicate provider: %w", err)
	}
	return provider, nil
}

func NewDefault() (*openaicompat.Provider, error) {
	return New("")
}
