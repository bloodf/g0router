package bedrock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

func TestNewImplementsProvider(t *testing.T) {
	provider := New("")

	if provider.Name() != providers.ProviderBedrock {
		t.Fatalf("Name() = %q", provider.Name())
	}

	var _ providers.Provider = provider
}

func TestChatCompletionSignsConverseRequest(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotDate string
	var gotHash string
	var gotToken string
	var gotBody []byte
	var gotRequest converseRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.RequestURI
		gotAuth = r.Header.Get("Authorization")
		gotDate = r.Header.Get("X-Amz-Date")
		gotHash = r.Header.Get("X-Amz-Content-Sha256")
		gotToken = r.Header.Get("X-Amz-Security-Token")

		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if err := json.Unmarshal(gotBody, &gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockConverseResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	temp := 0.3
	maxTokens := 64
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:       "anthropic.claude-3-haiku-20240307-v1:0",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		Stop:        "END",
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	expectedPath := "/model/anthropic.claude-3-haiku-20240307-v1:0/converse"
	if gotPath != expectedPath {
		t.Fatalf("path = %q, want %q", gotPath, expectedPath)
	}
	if gotRequest.InferenceConfig == nil {
		t.Fatal("missing inferenceConfig")
	}
	if gotRequest.InferenceConfig.MaxTokens != 64 {
		t.Errorf("maxTokens = %d", gotRequest.InferenceConfig.MaxTokens)
	}
	if gotRequest.InferenceConfig.Temperature == nil || *gotRequest.InferenceConfig.Temperature != 0.3 {
		t.Errorf("temperature = %+v", gotRequest.InferenceConfig.Temperature)
	}
	if len(gotRequest.InferenceConfig.StopSequences) != 1 || gotRequest.InferenceConfig.StopSequences[0] != "END" {
		t.Errorf("stopSequences = %+v, want [END]", gotRequest.InferenceConfig.StopSequences)
	}
	if len(gotRequest.Messages) != 1 || gotRequest.Messages[0].Role != "user" || len(gotRequest.Messages[0].Content) != 1 || gotRequest.Messages[0].Content[0].Text != "hello" {
		t.Errorf("messages = %+v", gotRequest.Messages)
	}

	sum := sha256.Sum256(gotBody)
	if gotHash != hex.EncodeToString(sum[:]) {
		t.Errorf("X-Amz-Content-Sha256 = %q", gotHash)
	}
	if gotToken != "token" {
		t.Errorf("X-Amz-Security-Token = %q", gotToken)
	}
	if _, err := time.Parse("20060102T150405Z", gotDate); err != nil {
		t.Errorf("X-Amz-Date = %q: %v", gotDate, err)
	}
	assertAuthorizationHeader(t, gotAuth, gotDate)

	if resp.Model != "anthropic.claude-3-haiku-20240307-v1:0" {
		t.Errorf("model = %q", resp.Model)
	}
}

func TestChatCompletionNormalizesStopArray(t *testing.T) {
	var gotRequest converseRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockConverseResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "anthropic.claude-3-haiku-20240307-v1:0",
		Stop:     []any{"END", "STOP"},
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	got := gotRequest.InferenceConfig.StopSequences
	if len(got) != 2 || got[0] != "END" || got[1] != "STOP" {
		t.Fatalf("stopSequences = %+v, want [END STOP]", got)
	}
}

func TestChatCompletionRejectsUnsupportedStopShape(t *testing.T) {
	provider := New("http://127.0.0.1:1")

	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "anthropic.claude-3-haiku-20240307-v1:0",
		Stop:     123,
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected unsupported stop error")
	}
	if !strings.Contains(err.Error(), "unsupported stop") {
		t.Fatalf("error = %v, want unsupported stop", err)
	}
}

func TestChatCompletionParsesBedrockResponse(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusOK, bedrockConverseResponseJSON)
	provider := New(server.URL)

	resp, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}

	if resp.Object != "chat.completion" {
		t.Errorf("object = %q", resp.Object)
	}
	if resp.ID == "" {
		t.Error("id is empty")
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("choices len = %d", len(resp.Choices))
	}
	choice := resp.Choices[0]
	if choice.Message.Role != "assistant" {
		t.Errorf("role = %q", choice.Message.Role)
	}
	if choice.Message.Content != "hello from bedrock" {
		t.Errorf("content = %#v", choice.Message.Content)
	}
	if choice.FinishReason == nil || *choice.FinishReason != "end_turn" {
		t.Errorf("finish reason = %+v", choice.FinishReason)
	}
	if resp.Usage == nil || resp.Usage.PromptTokens != 5 || resp.Usage.CompletionTokens != 7 || resp.Usage.TotalTokens != 12 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

func TestChatCompletionParsesErrorResponse(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusForbidden, `{"message":"bad aws credentials"}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "bad aws credentials") {
		t.Fatalf("error = %q", err)
	}
}

func TestChatCompletionValidatesCredentials(t *testing.T) {
	provider := New("http://127.0.0.1")

	_, err := provider.ChatCompletion(context.Background(), providers.Key{Value: "missing-secret"}, testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse bedrock credentials") {
		t.Fatalf("error = %q", err)
	}
}

func TestListModelsSignsAndParsesFoundationModels(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotDate string
	var gotHash string
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.RequestURI
		gotAuth = r.Header.Get("Authorization")
		gotDate = r.Header.Get("X-Amz-Date")
		gotHash = r.Header.Get("X-Amz-Content-Sha256")
		gotToken = r.Header.Get("X-Amz-Security-Token")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockModelsResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	models, err := provider.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("method = %q", gotMethod)
	}
	if gotPath != "/foundation-models" {
		t.Errorf("path = %q", gotPath)
	}
	emptyHash := sha256.Sum256(nil)
	if gotHash != hex.EncodeToString(emptyHash[:]) {
		t.Errorf("X-Amz-Content-Sha256 = %q", gotHash)
	}
	if gotToken != "token" {
		t.Errorf("X-Amz-Security-Token = %q", gotToken)
	}
	if _, err := time.Parse("20060102T150405Z", gotDate); err != nil {
		t.Errorf("X-Amz-Date = %q: %v", gotDate, err)
	}
	assertAuthorizationHeader(t, gotAuth, gotDate)

	if len(models) != 2 {
		t.Fatalf("models len = %d", len(models))
	}
	if models[0].ID != "anthropic.claude-3-haiku-20240307-v1:0" || models[0].OwnedBy != "Anthropic" || models[0].Provider != providers.ProviderBedrock {
		t.Errorf("model[0] = %+v", models[0])
	}
	if models[1].ID != "amazon.titan-text-lite-v1" || models[1].Object != "model" {
		t.Errorf("model[1] = %+v", models[1])
	}
}

func TestListModelsUsesBedrockControlPlaneEndpointByDefault(t *testing.T) {
	provider := New("")

	req, err := provider.newListModelsRequest(context.Background(), credentials{accessKey: "AKID", secretKey: "SECRET"})
	if err != nil {
		t.Fatalf("newListModelsRequest: %v", err)
	}

	if req.URL.Host != "bedrock.us-east-1.amazonaws.com" {
		t.Fatalf("host = %q, want Bedrock control-plane endpoint", req.URL.Host)
	}
	if req.URL.Path != "/foundation-models" {
		t.Fatalf("path = %q", req.URL.Path)
	}
}

func TestUnsupportedMethodsReturnErrors(t *testing.T) {
	provider := New("http://127.0.0.1")

	if _, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest()); err == nil {
		t.Fatal("expected stream error")
	} else if !errors.Is(err, providers.ErrStreamingUnsupported) {
		t.Fatalf("error = %v, want ErrStreamingUnsupported", err)
	}
}

func assertAuthorizationHeader(t *testing.T, gotAuth, gotDate string) {
	t.Helper()

	shortDate := gotDate[:8]
	checks := []string{
		"AWS4-HMAC-SHA256 Credential=AKID/" + shortDate + "/us-east-1/bedrock/aws4_request",
		"SignedHeaders=content-type;host;x-amz-content-sha256;x-amz-date;x-amz-security-token",
		"Signature=",
	}
	for _, check := range checks {
		if !strings.Contains(gotAuth, check) {
			t.Fatalf("Authorization = %q, missing %q", gotAuth, check)
		}
	}
}

func bedrockJSONServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func testKey() providers.Key {
	return providers.Key{
		Value:    "AKID:SECRET:token",
		Provider: providers.ProviderBedrock,
		ConnID:   "conn-1",
		AuthType: "aws",
	}
}

func testChatRequest() *providers.ChatRequest {
	maxTokens := 32
	return &providers.ChatRequest{
		Model:     "anthropic.claude-3-haiku-20240307-v1:0",
		MaxTokens: &maxTokens,
		Messages: []providers.Message{
			{Role: "user", Content: "hello"},
		},
	}
}

const bedrockConverseResponseJSON = `{
	"output": {
		"message": {
			"role": "assistant",
			"content": [{"text": "hello from bedrock"}]
		}
	},
	"stopReason": "end_turn",
	"usage": {"inputTokens": 5, "outputTokens": 7, "totalTokens": 12}
}`

const bedrockModelsResponseJSON = `{
	"modelSummaries": [
		{
			"modelId": "anthropic.claude-3-haiku-20240307-v1:0",
			"providerName": "Anthropic",
			"modelName": "Claude 3 Haiku"
		},
		{
			"modelId": "amazon.titan-text-lite-v1",
			"providerName": "Amazon",
			"modelName": "Titan Text Lite"
		}
	]
}`
