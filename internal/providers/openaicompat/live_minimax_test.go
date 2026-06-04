package openaicompat

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

func TestLiveMiniMaxListModelsSmoke(t *testing.T) {
	if os.Getenv("G0ROUTER_LIVE_TESTS") != "1" {
		t.Skip("set G0ROUTER_LIVE_TESTS=1 to run live provider smoke tests")
	}
	apiKey := os.Getenv("G0ROUTER_E2E_MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("set G0ROUTER_E2E_MINIMAX_API_KEY to run MiniMax live smoke test")
	}

	baseURL := os.Getenv("G0ROUTER_E2E_MINIMAX_BASE_URL")
	if baseURL == "" {
		baseURL = DefaultConfigs()[providers.ProviderMiniMax].BaseURL
	}
	provider, err := New(Config{Provider: providers.ProviderMiniMax, BaseURL: baseURL})
	if err != nil {
		t.Fatalf("create minimax provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	models, err := provider.ListModels(ctx, providers.Key{
		Provider: providers.ProviderMiniMax,
		Value:    apiKey,
		AuthType: "api_key",
	})
	if err != nil {
		if strings.Contains(err.Error(), apiKey) {
			t.Fatal("MiniMax live smoke error leaked API key")
		}
		t.Fatalf("MiniMax list models smoke failed: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("MiniMax list models returned no models")
	}
}
