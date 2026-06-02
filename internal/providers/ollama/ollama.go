package ollama

import (
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/openaicompat"
)

const defaultBaseURL = "http://localhost:11434"

func New(baseURL string) (*openaicompat.Provider, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	provider, err := openaicompat.New(openaicompat.Config{
		Provider: providers.ProviderOllama,
		BaseURL:  baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("ollama provider: %w", err)
	}
	return provider, nil
}

func NewDefault() (*openaicompat.Provider, error) {
	return New("")
}
