package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

func TestNewUsesFastHTTPClient(t *testing.T) {
	provider := New("")
	if provider.client == nil {
		t.Fatal("client is nil")
	}

	var _ *fasthttp.Client = provider.client
}

func TestProviderSatisfiesInterface(t *testing.T) {
	var _ providers.Provider = New("")
}

func TestBuildGenerateContentRequestWithAPIKey(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotAuth string
	var gotContentType string
	var gotRequest generateContentRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(generateContentResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	temp := 0.2
	maxTokens := 32
	resp, err := provider.ChatCompletion(context.Background(), apiKey(), &providers.ChatRequest{
		Model:       "gemini-2.5-flash",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotPath != "/v1beta/models/gemini-2.5-flash:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "key=gemini-key" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q", gotContentType)
	}
	if len(gotRequest.Contents) != 1 {
		t.Fatalf("contents len = %d", len(gotRequest.Contents))
	}
	if gotRequest.Contents[0].Role != "user" {
		t.Errorf("role = %q", gotRequest.Contents[0].Role)
	}
	if gotRequest.Contents[0].Parts[0].Text != "hello" {
		t.Errorf("text = %q", gotRequest.Contents[0].Parts[0].Text)
	}
	if gotRequest.GenerationConfig == nil {
		t.Fatal("generationConfig is nil")
	}
	if gotRequest.GenerationConfig.Temperature == nil || *gotRequest.GenerationConfig.Temperature != 0.2 {
		t.Errorf("temperature = %+v", gotRequest.GenerationConfig.Temperature)
	}
	if gotRequest.GenerationConfig.MaxOutputTokens == nil || *gotRequest.GenerationConfig.MaxOutputTokens != 32 {
		t.Errorf("maxOutputTokens = %+v", gotRequest.GenerationConfig.MaxOutputTokens)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("response model = %q", resp.Model)
	}
}

func TestBuildGenerateContentRequestWithOAuth(t *testing.T) {
	var gotQuery string
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(generateContentResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletion(context.Background(), oauthKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if gotQuery != "" {
		t.Errorf("query = %q", gotQuery)
	}
	if gotAuth != "Bearer oauth-token" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}

func TestParseGenerateContentResponse(t *testing.T) {
	server := jsonServer(t, http.StatusOK, generateContentResponseJSON)
	provider := New(server.URL)

	resp, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.Object != "chat.completion" {
		t.Errorf("Object = %q", resp.Object)
	}
	if resp.Model != "gemini-2.5-flash" {
		t.Errorf("Model = %q", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices len = %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Role != "assistant" {
		t.Errorf("role = %q", resp.Choices[0].Message.Role)
	}
	if resp.Choices[0].Message.Content != "hello back" {
		t.Errorf("content = %#v", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].FinishReason == nil || *resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish reason = %+v", resp.Choices[0].FinishReason)
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 14 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestParseError401(t *testing.T) {
	server := jsonServer(t, http.StatusUnauthorized, `{"error":{"message":"invalid api key"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseError429(t *testing.T) {
	server := jsonServer(t, http.StatusTooManyRequests, `{"error":{"message":"slow down"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrRateLimit) {
		t.Fatalf("expected ErrRateLimit, got %v", err)
	}
}

func TestParseError500(t *testing.T) {
	server := jsonServer(t, http.StatusInternalServerError, `{"error":{"message":"upstream failed"}}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), apiKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func apiKey() providers.Key {
	return providers.Key{Value: "gemini-key", Provider: providers.ProviderGemini, ConnID: "conn-1", AuthType: "api_key"}
}

func oauthKey() providers.Key {
	return providers.Key{Value: "oauth-token", Provider: providers.ProviderGemini, ConnID: "conn-1", AuthType: "oauth"}
}

func testChatRequest() *providers.ChatRequest {
	return &providers.ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

func jsonServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

const generateContentResponseJSON = `{
	"candidates": [{
		"content": {"role": "model", "parts": [{"text": "hello back"}]},
		"finishReason": "STOP"
	}],
	"usageMetadata": {
		"promptTokenCount": 5,
		"candidatesTokenCount": 9,
		"totalTokenCount": 14
	}
}`
