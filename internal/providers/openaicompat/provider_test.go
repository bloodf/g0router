package openaicompat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

func TestConfiguredProvidersUseOpenAICompatibleEndpoints(t *testing.T) {
	cases := []struct {
		name     string
		provider providers.ModelProvider
	}{
		{"groq", providers.ProviderGroq},
		{"cerebras", providers.ProviderCerebras},
		{"perplexity", providers.ProviderPerplexity},
		{"fireworks", providers.ProviderFireworks},
		{"together", providers.ProviderTogether},
		{"nvidia", providers.ProviderNVIDIA},
		{"deepseek", providers.ProviderDeepSeek},
		{"openrouter", providers.ProviderOpenRouter},
		{"huggingface", providers.ProviderHuggingFace},
		{"nebius", providers.ProviderNebius},
		{"minimax", providers.ProviderMiniMax},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var chatPath string
			var modelsPath string
			var gotAuth string
			var gotContentType string
			var gotRequest providers.ChatRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/chat/completions":
					chatPath = r.URL.Path
					gotAuth = r.Header.Get("Authorization")
					gotContentType = r.Header.Get("Content-Type")
					if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
						t.Fatalf("decode request: %v", err)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(chatResponseJSON))
				case "/v1/models":
					modelsPath = r.URL.Path
					gotAuth = r.Header.Get("Authorization")
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(modelsResponseJSON))
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(server.Close)

			provider, err := New(Config{Provider: tc.provider, BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if provider.Name() != tc.provider {
				t.Fatalf("Name = %q", provider.Name())
			}

			temp := 0.2
			resp, err := provider.ChatCompletion(context.Background(), testKey(tc.provider), &providers.ChatRequest{
				Model:       "test-model",
				Temperature: &temp,
				Messages: []providers.Message{
					{Role: "user", Content: "hello"},
				},
			})
			if err != nil {
				t.Fatalf("ChatCompletion: %v", err)
			}
			if chatPath != "/v1/chat/completions" {
				t.Errorf("chat path = %q", chatPath)
			}
			if gotAuth != "Bearer sk-test" {
				t.Errorf("Authorization = %q", gotAuth)
			}
			if gotContentType != "application/json" {
				t.Errorf("Content-Type = %q", gotContentType)
			}
			if gotRequest.Model != "test-model" {
				t.Errorf("model = %q", gotRequest.Model)
			}
			if gotRequest.Temperature == nil || *gotRequest.Temperature != 0.2 {
				t.Errorf("temperature = %+v", gotRequest.Temperature)
			}
			if resp.ID != "chatcmpl-123" {
				t.Errorf("response id = %q", resp.ID)
			}

			models, err := provider.ListModels(context.Background(), testKey(tc.provider))
			if err != nil {
				t.Fatalf("ListModels: %v", err)
			}
			if modelsPath != "/v1/models" {
				t.Errorf("models path = %q", modelsPath)
			}
			if len(models) != 2 {
				t.Fatalf("models len = %d", len(models))
			}
			if models[0].Provider != tc.provider {
				t.Errorf("model provider = %q", models[0].Provider)
			}
		})
	}
}

func TestNewDefaultProvider(t *testing.T) {
	provider, err := NewDefault(providers.ProviderGroq)
	if err != nil {
		t.Fatalf("NewDefault: %v", err)
	}
	if provider.Name() != providers.ProviderGroq {
		t.Fatalf("Name = %q", provider.Name())
	}
	if provider.client == nil {
		t.Fatal("client is nil")
	}

	var _ *fasthttp.Client = provider.client
}

func TestDefaultConfigsAreRegistered(t *testing.T) {
	configs := DefaultConfigs()
	want := map[providers.ModelProvider]string{
		providers.ProviderGroq:        "https://api.groq.com/openai",
		providers.ProviderCerebras:    "https://api.cerebras.ai",
		providers.ProviderPerplexity:  "https://api.perplexity.ai",
		providers.ProviderFireworks:   "https://api.fireworks.ai/inference",
		providers.ProviderTogether:    "https://api.together.xyz",
		providers.ProviderNVIDIA:      "https://integrate.api.nvidia.com",
		providers.ProviderDeepSeek:    "https://api.deepseek.com",
		providers.ProviderOpenRouter:  "https://openrouter.ai/api",
		providers.ProviderHuggingFace: "https://api-inference.huggingface.co",
		providers.ProviderNebius:      "https://api.studio.nebius.ai",
		providers.ProviderMiniMax:     "https://api.minimax.io/v1",
	}

	if len(configs) != len(want) {
		t.Fatalf("configs len = %d", len(configs))
	}
	for provider, baseURL := range want {
		config, ok := configs[provider]
		if !ok {
			t.Fatalf("missing config for %s", provider)
		}
		if config.BaseURL != baseURL {
			t.Errorf("%s BaseURL = %q", provider, config.BaseURL)
		}
	}

	configs[providers.ProviderGroq] = Config{Provider: providers.ProviderGroq, BaseURL: "http://changed"}
	fresh := DefaultConfigs()
	if fresh[providers.ProviderGroq].BaseURL != want[providers.ProviderGroq] {
		t.Fatalf("DefaultConfigs returned mutable registry")
	}
}

func TestNewRejectsInvalidConfig(t *testing.T) {
	_, err := New(Config{Provider: providers.ProviderGroq})
	if err == nil {
		t.Fatal("expected missing base URL error")
	}
	if !strings.Contains(err.Error(), "base URL") {
		t.Fatalf("error = %q", err)
	}

	_, err = NewDefault(providers.ProviderOpenAI)
	if err == nil {
		t.Fatal("expected unknown provider error")
	}
	if !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("expected ErrUnknownProvider, got %v", err)
	}
}

func TestParseSSEStream(t *testing.T) {
	server := streamServer(t, strings.Join([]string{
		"data: " + streamChunkRoleJSON,
		"",
		"data: " + streamChunkContentJSON,
		"",
		"data: [DONE]",
		"",
	}, "\n"))
	provider, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	chunks, err := provider.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}

	got := collectChunks(chunks)
	if len(got) != 2 {
		t.Fatalf("chunks len = %d", len(got))
	}
	if got[0].Choices[0].Delta.Role == nil || *got[0].Choices[0].Delta.Role != "assistant" {
		t.Errorf("first role = %+v", got[0].Choices[0].Delta.Role)
	}
	if got[1].Choices[0].Delta.Content == nil || *got[1].Choices[0].Delta.Content != "hello" {
		t.Errorf("second content = %+v", got[1].Choices[0].Delta.Content)
	}
}

func TestChatCompletionStreamReturnsBeforeUpstreamCompletes(t *testing.T) {
	release := make(chan struct{})
	server := liveStreamServer(t, release, streamChunkContentJSON, streamChunkRoleJSON)
	provider, err := New(Config{Provider: providers.ProviderGroq, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	type streamResult struct {
		chunks <-chan providers.StreamChunk
		err    error
	}
	result := make(chan streamResult, 1)
	go func() {
		chunks, err := provider.ChatCompletionStream(context.Background(), testKey(providers.ProviderGroq), testChatRequest())
		result <- streamResult{chunks: chunks, err: err}
	}()

	var chunks <-chan providers.StreamChunk
	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("ChatCompletionStream: %v", got.err)
		}
		chunks = got.chunks
	case <-time.After(200 * time.Millisecond):
		close(release)
		t.Fatal("ChatCompletionStream blocked until the upstream response completed")
	}

	select {
	case chunk := <-chunks:
		if chunk.Choices[0].Delta.Content == nil || *chunk.Choices[0].Delta.Content != "hello" {
			t.Fatalf("first chunk = %+v", chunk)
		}
	case <-time.After(200 * time.Millisecond):
		close(release)
		t.Fatal("stream did not deliver the first chunk before upstream completion")
	}

	close(release)
	rest := collectChunks(chunks)
	if len(rest) != 1 || rest[0].Choices[0].Delta.Role == nil || *rest[0].Choices[0].Delta.Role != "assistant" {
		t.Fatalf("remaining chunks = %+v", rest)
	}
}

func testKey(provider providers.ModelProvider) providers.Key {
	return providers.Key{Value: "sk-test", Provider: provider, ConnID: "conn-1", AuthType: "api_key"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "test-model",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

func streamServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var got providers.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode stream request: %v", err)
		}
		if got.Stream == nil || !*got.Stream {
			t.Errorf("stream = %+v", got.Stream)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func liveStreamServer(t *testing.T, release <-chan struct{}, firstChunk string, secondChunk string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var got providers.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode stream request: %v", err)
		}
		if got.Stream == nil || !*got.Stream {
			t.Errorf("stream = %+v", got.Stream)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: " + firstChunk + "\n\n"))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-release
		_, _ = w.Write([]byte("data: " + secondChunk + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(server.Close)
	return server
}

func collectChunks(chunks <-chan providers.StreamChunk) []providers.StreamChunk {
	var got []providers.StreamChunk
	for chunk := range chunks {
		got = append(got, chunk)
	}
	return got
}

const chatResponseJSON = `{
	"id": "chatcmpl-123",
	"object": "chat.completion",
	"created": 1710000000,
	"model": "test-model",
	"choices": [{
		"index": 0,
		"message": {"role": "assistant", "content": "hello back"},
		"finish_reason": "stop"
	}],
	"usage": {"prompt_tokens": 5, "completion_tokens": 9, "total_tokens": 14}
}`

const modelsResponseJSON = `{
	"object": "list",
	"data": [
		{"id": "model-a", "object": "model", "created": 1710000001, "owned_by": "provider"},
		{"id": "model-b", "object": "model", "created": 1710000002, "owned_by": "provider"}
	]
}`

const streamChunkRoleJSON = `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710000000,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`

const streamChunkContentJSON = `{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1710000000,"model":"test-model","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`
