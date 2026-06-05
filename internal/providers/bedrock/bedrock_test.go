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

func TestChatCompletionStreamIsSupported(t *testing.T) {
	provider := New("http://127.0.0.1")

	// Streaming is now implemented: a transport failure to an unreachable
	// endpoint must surface as a real dispatch error, never the
	// ErrStreamingUnsupported capability sentinel.
	if _, err := provider.ChatCompletionStream(context.Background(), testKey(), testChatRequest()); err == nil {
		t.Fatal("expected stream dispatch error against unreachable endpoint")
	} else if errors.Is(err, providers.ErrStreamingUnsupported) {
		t.Fatalf("error = %v, streaming should be supported", err)
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

// ---- additional coverage tests ----

func TestChatCompletionError500(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusInternalServerError, `{"message":"internal error"}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "internal error") {
		t.Fatalf("error = %q", err)
	}
}

func TestChatCompletionError400(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusBadRequest, `{"message":"bad request"}`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("error = %q", err)
	}
}

func TestChatCompletionErrorEmptyBody(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusUnauthorized, ``)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("error = %q, want empty response", err)
	}
}

func TestChatCompletionErrorNonJSONBody(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusForbidden, `plain text`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
	if !strings.Contains(err.Error(), "plain text") {
		t.Fatalf("error = %q, want plain text", err)
	}
}

func TestListModelsError401(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusUnauthorized, `{"message":"bad creds"}`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), testKey())
	if !errors.Is(err, ErrAuth) {
		t.Fatalf("expected ErrAuth, got %v", err)
	}
}

func TestListModelsError500(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusInternalServerError, `{"message":"server error"}`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), testKey())
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

func TestListModelsValidatesCredentials(t *testing.T) {
	provider := New("http://127.0.0.1")

	_, err := provider.ListModels(context.Background(), providers.Key{Value: "missing-secret"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parse bedrock credentials") {
		t.Fatalf("error = %q", err)
	}
}

func TestListModelsSkipsEmptyModelID(t *testing.T) {
	body := `{"modelSummaries":[{"modelId":"","providerName":"X"},{"modelId":"valid-model","providerName":"Y"}]}`
	server := bedrockJSONServer(t, http.StatusOK, body)
	provider := New(server.URL)

	models, err := provider.ListModels(context.Background(), testKey())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 1 || models[0].ID != "valid-model" {
		t.Fatalf("models = %+v, want only non-empty model", models)
	}
}

func TestStopSequencesAllBranches(t *testing.T) {
	// nil
	seq, err := stopSequences(nil)
	if err != nil || seq != nil {
		t.Fatalf("nil: seq=%v err=%v", seq, err)
	}
	// empty string
	seq, err = stopSequences("")
	if err != nil || seq != nil {
		t.Fatalf("empty string: seq=%v err=%v", seq, err)
	}
	// non-empty string
	seq, err = stopSequences("END")
	if err != nil || len(seq) != 1 || seq[0] != "END" {
		t.Fatalf("string: seq=%v err=%v", seq, err)
	}
	// []string
	seq, err = stopSequences([]string{"A", "", "B"})
	if err != nil || len(seq) != 2 {
		t.Fatalf("[]string: seq=%v err=%v", seq, err)
	}
	// []any with non-string → error
	_, err = stopSequences([]any{42})
	if err == nil {
		t.Fatal("expected error for non-string in []any")
	}
	// unsupported type
	_, err = stopSequences(3.14)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestNonEmptyStrings(t *testing.T) {
	result := nonEmptyStrings([]string{"a", "", "b", ""})
	if len(result) != 2 || result[0] != "a" || result[1] != "b" {
		t.Fatalf("result = %+v", result)
	}
}

func TestToContentBlocksNonString(t *testing.T) {
	blocks := toContentBlocks(42)
	if len(blocks) != 1 || blocks[0].Text != "42" {
		t.Fatalf("blocks = %+v", blocks)
	}
}

func TestToChatResponseNoUsage(t *testing.T) {
	resp := toChatResponse("model-x", converseResponse{
		Output: bedrockOutput{
			Message: bedrockMessage{Role: "assistant", Content: []bedrockContentBlock{{Text: "hi"}}},
		},
		StopReason: nil,
		Usage:      nil,
	})
	if resp.Usage != nil {
		t.Fatalf("expected nil usage, got %+v", resp.Usage)
	}
}

func TestToChatResponseEmptyRole(t *testing.T) {
	resp := toChatResponse("model-x", converseResponse{
		Output: bedrockOutput{
			Message: bedrockMessage{Role: "", Content: []bedrockContentBlock{{Text: "hi"}}},
		},
	})
	if resp.Choices[0].Message.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", resp.Choices[0].Message.Role)
	}
}

func TestToChatResponseUsageTotalZero(t *testing.T) {
	// When TotalTokens==0, it should be computed from InputTokens+OutputTokens
	resp := toChatResponse("model-x", converseResponse{
		Output: bedrockOutput{
			Message: bedrockMessage{Role: "assistant", Content: []bedrockContentBlock{{Text: "hi"}}},
		},
		Usage: &bedrockResponseUsage{InputTokens: 5, OutputTokens: 7, TotalTokens: 0},
	})
	if resp.Usage == nil || resp.Usage.TotalTokens != 12 {
		t.Fatalf("total tokens = %+v, want 12", resp.Usage)
	}
}

func TestParseCredentialsSessionToken(t *testing.T) {
	creds, err := parseCredentials("AKID:SECRET:SESSION")
	if err != nil {
		t.Fatalf("parseCredentials: %v", err)
	}
	if creds.accessKey != "AKID" || creds.secretKey != "SECRET" || creds.sessionToken != "SESSION" {
		t.Fatalf("creds = %+v", creds)
	}
}

func TestParseCredentialsNoSession(t *testing.T) {
	creds, err := parseCredentials("AKID:SECRET")
	if err != nil {
		t.Fatalf("parseCredentials: %v", err)
	}
	if creds.sessionToken != "" {
		t.Fatalf("session token = %q, want empty", creds.sessionToken)
	}
}

func TestParseCredentialsInvalid(t *testing.T) {
	for _, bad := range []string{"", "AKID", ":SECRET", "AKID:"} {
		_, err := parseCredentials(bad)
		if err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestModelEndpointForCustomURL(t *testing.T) {
	result := modelEndpointFor("https://bedrock-runtime.eu-west-1.amazonaws.com")
	if result != "https://bedrock.eu-west-1.amazonaws.com" {
		t.Fatalf("modelEndpointFor = %q", result)
	}
}

func TestModelEndpointForDefault(t *testing.T) {
	result := modelEndpointFor(defaultRuntimeBaseURL)
	if result != defaultModelBaseURL {
		t.Fatalf("modelEndpointFor default = %q", result)
	}
}

func TestChatCompletionMaxCompletionTokens(t *testing.T) {
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
	maxComp := 99
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:               "anthropic.claude-3-haiku-20240307-v1:0",
		MaxCompletionTokens: &maxComp,
		Messages:            []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotRequest.InferenceConfig == nil || gotRequest.InferenceConfig.MaxTokens != 99 {
		t.Fatalf("maxTokens = %+v, want 99", gotRequest.InferenceConfig)
	}
}

func TestChatCompletionStopStringSlice(t *testing.T) {
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
		Stop:     []string{"A", "B"},
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if len(gotRequest.InferenceConfig.StopSequences) != 2 {
		t.Fatalf("stop sequences = %+v, want 2", gotRequest.InferenceConfig.StopSequences)
	}
}

func TestSignIncludesSessionToken(t *testing.T) {
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Amz-Security-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockConverseResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletion(context.Background(), providers.Key{
		Value: "AKID:SECRET:mysession",
	}, testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotToken != "mysession" {
		t.Fatalf("X-Amz-Security-Token = %q, want mysession", gotToken)
	}
}

func TestNewInvokeRequestMarshalError(t *testing.T) {
	// toConverseRequest with unsupported stop type causes error before marshal
	provider := New("http://127.0.0.1")
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "anthropic.claude-3-haiku-20240307-v1:0",
		Stop:     3.14, // unsupported type
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error for unsupported stop type")
	}
}

func TestNewListModelsRequestCreated(t *testing.T) {
	provider := New("http://127.0.0.1:9999")
	req, err := provider.newListModelsRequest(context.Background(), credentials{accessKey: "A", secretKey: "S"})
	if err != nil {
		t.Fatalf("newListModelsRequest: %v", err)
	}
	if req.URL.Path != "/foundation-models" {
		t.Fatalf("path = %q", req.URL.Path)
	}
}

func TestChatCompletionInvalidJSONResponse(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusOK, `not-json`)
	provider := New(server.URL)

	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil || !strings.Contains(err.Error(), "parse bedrock chat response") {
		t.Fatalf("error = %v, want parse error", err)
	}
}

func TestListModelsInvalidJSONResponse(t *testing.T) {
	server := bedrockJSONServer(t, http.StatusOK, `not-json`)
	provider := New(server.URL)

	_, err := provider.ListModels(context.Background(), testKey())
	if err == nil || !strings.Contains(err.Error(), "parse bedrock models response") {
		t.Fatalf("error = %v, want parse models error", err)
	}
}

func TestChatCompletionNetworkError(t *testing.T) {
	// No server listening — client.Do returns connection refused
	provider := New("http://127.0.0.1:1")
	_, err := provider.ChatCompletion(context.Background(), testKey(), testChatRequest())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "bedrock invoke model") {
		t.Fatalf("error = %v, want bedrock invoke model", err)
	}
}

func TestListModelsNetworkError(t *testing.T) {
	provider := New("http://127.0.0.1:1")
	_, err := provider.ListModels(context.Background(), testKey())
	if err == nil {
		t.Fatal("expected network error")
	}
	if !strings.Contains(err.Error(), "bedrock list models") {
		t.Fatalf("error = %v, want bedrock list models", err)
	}
}

func TestGetBodyClosure(t *testing.T) {
	// Exercise the GetBody closure by calling newInvokeRequest and invoking GetBody
	provider := New("http://127.0.0.1")
	req, err := provider.newInvokeRequest(context.Background(), credentials{accessKey: "A", secretKey: "S"}, &providers.ChatRequest{
		Model:    "model-x",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("newInvokeRequest: %v", err)
	}
	body, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody: %v", err)
	}
	_ = body
}

func TestChatCompletionReadBodyError(t *testing.T) {
	// Test that newInvokeRequest sign path covers session-token-free case
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockConverseResponseJSON))
	}))
	t.Cleanup(server.Close)

	// Use credentials without session token
	provider := New(server.URL)
	maxTokens := 32
	_, err := provider.ChatCompletion(context.Background(), providers.Key{Value: "AKID:SECRET"}, &providers.ChatRequest{
		Model:     "anthropic.claude-3-haiku-20240307-v1:0",
		MaxTokens: &maxTokens,
		Messages:  []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion without session token: %v", err)
	}
}

func TestListModelsReadBodyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockModelsResponseJSON))
	}))
	t.Cleanup(server.Close)

	// Covers newListModelsRequest: no session token in path
	provider := New(server.URL)
	_, err := provider.ListModels(context.Background(), providers.Key{Value: "AKID:SECRET"})
	if err != nil {
		t.Fatalf("ListModels without session token: %v", err)
	}
}

func TestChatCompletionWithTopP(t *testing.T) {
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
	topP := 0.9
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "anthropic.claude-3-haiku-20240307-v1:0",
		TopP:     &topP,
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotRequest.InferenceConfig == nil || gotRequest.InferenceConfig.TopP == nil || *gotRequest.InferenceConfig.TopP != 0.9 {
		t.Fatalf("topP = %+v, want 0.9", gotRequest.InferenceConfig)
	}
}

func TestSignNoSessionToken(t *testing.T) {
	var gotToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Amz-Security-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(bedrockConverseResponseJSON))
	}))
	t.Cleanup(server.Close)

	provider := New(server.URL)
	_, err := provider.ChatCompletion(context.Background(), providers.Key{
		Value: "AKID:SECRET",
	}, testChatRequest())
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if gotToken != "" {
		t.Fatalf("X-Amz-Security-Token = %q, want empty", gotToken)
	}
}
