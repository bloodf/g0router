package xiaomi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestProviderRoutesStandardKeysToXiaomiAnthropicEndpoint(t *testing.T) {
	var gotPath string
	var gotAPIKey string
	var gotRequest map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-api-key")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL, "http://token-plan.invalid/anthropic")
	if provider.Name() != providers.ModelProvider("xiaomi") {
		t.Fatalf("Name = %q, want xiaomi", provider.Name())
	}

	resp, err := provider.ChatCompletion(context.Background(), providers.Key{
		Provider: providers.ModelProvider("xiaomi"),
		Value:    "sk-xiaomi",
		AuthType: "api_key",
	}, &providers.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q, want /v1/messages", gotPath)
	}
	if gotAPIKey != "sk-xiaomi" {
		t.Fatalf("x-api-key = %q, want sk-xiaomi", gotAPIKey)
	}
	if gotRequest["model"] != "claude-sonnet-4" {
		t.Fatalf("model = %v, want claude-sonnet-4", gotRequest["model"])
	}
	if resp.Provider != "" {
		t.Fatalf("provider should be annotated by proxy, got %q", resp.Provider)
	}
}

func TestProviderRoutesTokenPlanKeysToTokenPlanEndpoint(t *testing.T) {
	standard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("standard endpoint should not receive token-plan key")
	}))
	t.Cleanup(standard.Close)

	var gotPath string
	tokenPlan := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_2","type":"message","role":"assistant","model":"claude-sonnet-4","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`))
	}))
	t.Cleanup(tokenPlan.Close)

	provider := New(standard.URL, tokenPlan.URL)
	_, err := provider.ChatCompletion(context.Background(), providers.Key{
		Provider: providers.ModelProvider("xiaomi"),
		Value:    "tp-xiaomi",
		AuthType: "api_key",
	}, &providers.ChatRequest{
		Model:    "claude-sonnet-4",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q, want /v1/messages", gotPath)
	}
}
