package gitlabduo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

func TestChatCompletionExchangesDirectAccessAndRoutesOpenAIModel(t *testing.T) {
	var directAccessAuth string
	var directAccessBody map[string]any
	var gotGatewayAuth string
	var gotGatewayHeader string
	var gotPath string
	var gotRequest providers.ChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/ai/third_party_agents/direct_access":
			directAccessAuth = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&directAccessBody); err != nil {
				t.Fatalf("decode direct access request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"direct-token","headers":{"X-GitLab-Feature":"duo"}}`))
		case "/ai/v1/proxy/openai/v1/chat/completions":
			gotPath = r.URL.Path
			gotGatewayAuth = r.Header.Get("Authorization")
			gotGatewayHeader = r.Header.Get("X-GitLab-Feature")
			if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
				t.Fatalf("decode gateway request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(chatResponseJSON))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	provider := New(Config{GitLabURL: server.URL, GatewayURL: server.URL})
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "duo-chat-gpt-5-1",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if provider.Name() != providers.ProviderGitLabDuo {
		t.Fatalf("Name = %q, want gitlab-duo", provider.Name())
	}
	if directAccessAuth != "Bearer gitlab-oauth-token" {
		t.Fatalf("direct access auth = %q, want GitLab OAuth bearer", directAccessAuth)
	}
	if featureFlags, ok := directAccessBody["feature_flags"].(map[string]any); !ok || featureFlags["DuoAgentPlatformNext"] != true {
		t.Fatalf("direct access body = %+v, want DuoAgentPlatformNext", directAccessBody)
	}
	if gotPath != "/ai/v1/proxy/openai/v1/chat/completions" {
		t.Fatalf("path = %q, want OpenAI proxy chat completions", gotPath)
	}
	if gotGatewayAuth != "Bearer direct-token" {
		t.Fatalf("gateway auth = %q, want direct access bearer", gotGatewayAuth)
	}
	if gotGatewayHeader != "duo" {
		t.Fatalf("gateway header = %q, want direct access header", gotGatewayHeader)
	}
	if gotRequest.Model != "gpt-5.1-2025-11-13" {
		t.Fatalf("upstream model = %q, want mapped OpenAI model", gotRequest.Model)
	}
	if resp.ID != "chatcmpl-gitlab" {
		t.Fatalf("response id = %q", resp.ID)
	}
}

func TestChatCompletionRoutesAnthropicModelWithDirectAccessHeaders(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotHeader string
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/ai/third_party_agents/direct_access":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"direct-token","headers":{"X-GitLab-Feature":"duo"}}`))
		case "/ai/v1/proxy/anthropic/v1/messages":
			gotPath = r.URL.Path
			gotAuth = r.Header.Get("Authorization")
			gotHeader = r.Header.Get("X-GitLab-Feature")
			var req struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode anthropic request: %v", err)
			}
			gotModel = req.Model
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(anthropicResponseJSON))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	provider := New(Config{GitLabURL: server.URL, GatewayURL: server.URL})
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "duo-chat-sonnet-4-5",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotPath != "/ai/v1/proxy/anthropic/v1/messages" {
		t.Fatalf("path = %q, want Anthropic proxy messages", gotPath)
	}
	if gotAuth != "Bearer direct-token" {
		t.Fatalf("Authorization = %q, want direct access bearer", gotAuth)
	}
	if gotHeader != "duo" {
		t.Fatalf("X-GitLab-Feature = %q, want direct access header", gotHeader)
	}
	if gotModel != "claude-sonnet-4-5-20250929" {
		t.Fatalf("model = %q, want mapped Anthropic model", gotModel)
	}
	if resp.ID != "msg_gitlab" {
		t.Fatalf("response id = %q", resp.ID)
	}
}

func TestChatCompletionCachesDirectAccessToken(t *testing.T) {
	var directAccessCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v4/ai/third_party_agents/direct_access":
			directAccessCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"direct-token","headers":{}}`))
		case "/ai/v1/proxy/openai/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(chatResponseJSON))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	provider := New(Config{GitLabURL: server.URL, GatewayURL: server.URL})
	req := &providers.ChatRequest{Model: "duo-chat-gpt-5-1", Messages: []providers.Message{{Role: "user", Content: "hello"}}}
	if _, err := provider.ChatCompletion(context.Background(), testKey(), req); err != nil {
		t.Fatalf("first ChatCompletion: %v", err)
	}
	if _, err := provider.ChatCompletion(context.Background(), testKey(), req); err != nil {
		t.Fatalf("second ChatCompletion: %v", err)
	}
	if directAccessCalls != 1 {
		t.Fatalf("direct access calls = %d, want cached token reuse", directAccessCalls)
	}
}

func TestChatCompletionRejectsUnsupportedModel(t *testing.T) {
	provider := New(Config{GitLabURL: "https://gitlab.example", GatewayURL: "https://gateway.example"})
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "duo-chat-unknown",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("ChatCompletion error is nil")
	}
}

func TestListModelsReturnsDuoAliasesDeterministically(t *testing.T) {
	provider := New(Config{})
	models, err := provider.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != len(modelMappings) {
		t.Fatalf("models = %d, want %d", len(models), len(modelMappings))
	}
	if models[0].ID != "duo-chat-gpt-5-1" {
		t.Fatalf("first model = %q, want sorted Duo alias list", models[0].ID)
	}
	for _, model := range models {
		if model.Provider != providers.ProviderGitLabDuo || model.OwnedBy != "gitlab-duo" {
			t.Fatalf("model metadata = %+v, want gitlab-duo owner/provider", model)
		}
	}
}

func testKey() providers.Key {
	return providers.Key{Value: "gitlab-oauth-token", Provider: providers.ProviderGitLabDuo, AuthType: "oauth", ConnID: "conn-1"}
}

const chatResponseJSON = `{
	"id": "chatcmpl-gitlab",
	"object": "chat.completion",
	"created": 1710000000,
	"model": "gpt-5.1-2025-11-13",
	"choices": [{"index": 0, "message": {"role": "assistant", "content": "hello back"}, "finish_reason": "stop"}]
}`

const anthropicResponseJSON = `{
	"id": "msg_gitlab",
	"type": "message",
	"role": "assistant",
	"model": "claude-sonnet-4-5-20250929",
	"content": [{"type": "text", "text": "hello back"}],
	"stop_reason": "end_turn",
	"usage": {"input_tokens": 5, "output_tokens": 9}
}`
