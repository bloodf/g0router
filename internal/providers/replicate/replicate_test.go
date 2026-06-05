package replicate

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
)

func TestChatCompletionCreatesAndPollsPrediction(t *testing.T) {
	var gotAuth string
	var gotCreate predictionCreateRequest
	var polls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/predictions":
			gotAuth = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&gotCreate); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"processing","urls":{"get":"/v1/predictions/pred-1"}}`))
		case "/v1/predictions/pred-1":
			polls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"succeeded","output":["hello ","from replicate"]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	provider := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 2})
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "owner/model",
		Messages: []providers.Message{{Role: "system", Content: "be terse"}, {Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if provider.Name() != providers.ProviderReplicate {
		t.Fatalf("Name = %q, want replicate", provider.Name())
	}
	if gotAuth != "Bearer replicate-key" {
		t.Fatalf("Authorization = %q, want bearer key", gotAuth)
	}
	if gotCreate.Model != "owner/model" {
		t.Fatalf("create model = %q, want owner/model", gotCreate.Model)
	}
	if gotCreate.Input["prompt"] != "system: be terse\nuser: hello" {
		t.Fatalf("prompt = %q, want flattened messages", gotCreate.Input["prompt"])
	}
	if polls != 1 {
		t.Fatalf("polls = %d, want one terminal poll", polls)
	}
	if resp.Provider != providers.ProviderReplicate || resp.ConnectionID != "conn-1" || resp.AuthType != "api_key" {
		t.Fatalf("response metadata = %+v, want replicate connection metadata", resp)
	}
	if resp.Model != "owner/model" || len(resp.Choices) != 1 || resp.Choices[0].Message.Content != "hello from replicate" {
		t.Fatalf("response = %+v, want mapped prediction output", resp)
	}
}

func TestChatCompletionMapsStringPredictionOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/predictions":
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"succeeded","output":"plain output"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	provider := New(Config{BaseURL: server.URL})
	resp, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Choices[0].Message.Content != "plain output" {
		t.Fatalf("content = %q, want string output", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletionReportsFailedPrediction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pred-1","status":"failed","error":"bad replicate input"}`))
	}))
	t.Cleanup(server.Close)

	provider := New(Config{BaseURL: server.URL})
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil || !strings.Contains(err.Error(), "prediction failed") {
		t.Fatalf("error = %v, want failed prediction", err)
	}
}

func TestChatCompletionTimesOutPendingPrediction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pred-1","status":"processing","urls":{"get":"/v1/predictions/pred-1"}}`))
	}))
	t.Cleanup(server.Close)

	provider := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 1})
	_, err := provider.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error = %v, want timeout", err)
	}
}

func TestChatCompletionStreamIsSupported(t *testing.T) {
	provider := New(Config{BaseURL: "https://replicate.example"})
	// Streaming is implemented now: a missing key must fail fast with a
	// configuration error, never the ErrStreamingUnsupported capability
	// sentinel.
	_, err := provider.ChatCompletionStream(context.Background(), providers.Key{}, &providers.ChatRequest{Model: "owner/model"})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if errors.Is(err, providers.ErrStreamingUnsupported) {
		t.Fatalf("error = %v, streaming should be supported", err)
	}
}

func TestListModelsUnsupported(t *testing.T) {
	provider := New(Config{BaseURL: "https://replicate.example"})
	_, err := provider.ListModels(context.Background(), testKey())
	if err == nil || !strings.Contains(err.Error(), "list models unsupported") {
		t.Fatalf("error = %v, want list models unsupported", err)
	}
}

func testKey() providers.Key {
	return providers.Key{Value: "replicate-key", Provider: providers.ProviderReplicate, ConnID: "conn-1", AuthType: "api_key"}
}

// ---- additional coverage tests ----

func TestNewDefaultValues(t *testing.T) {
	p := NewDefault()
	if p.Name() != providers.ProviderReplicate {
		t.Fatalf("Name = %q, want replicate", p.Name())
	}
	if p.baseURL != defaultBaseURL {
		t.Fatalf("baseURL = %q", p.baseURL)
	}
	if p.pollInterval != defaultPollInterval {
		t.Fatalf("pollInterval = %v", p.pollInterval)
	}
	if p.maxPolls != defaultMaxPolls {
		t.Fatalf("maxPolls = %d", p.maxPolls)
	}
}

func TestNewTrimsBaseURL(t *testing.T) {
	p := New(Config{BaseURL: "https://custom.example.com/"})
	if p.baseURL != "https://custom.example.com" {
		t.Fatalf("baseURL = %q", p.baseURL)
	}
}

func TestChatCompletionNilRequest(t *testing.T) {
	p := New(Config{BaseURL: "https://replicate.example"})
	_, err := p.ChatCompletion(context.Background(), testKey(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestChatCompletionHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("bad token"))
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL})
	_, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil {
		t.Fatal("expected error for HTTP 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("error = %v, want status 401", err)
	}
}

func TestChatCompletionMissingIDAndURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"","status":"processing","urls":{}}`))
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 2})
	_, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil {
		t.Fatal("expected error for missing id and poll URL")
	}
	if !strings.Contains(err.Error(), "missing id and poll URL") {
		t.Fatalf("error = %v", err)
	}
}

func TestChatCompletionCanceledPrediction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pred-1","status":"canceled","error":"user canceled"}`))
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL})
	_, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil || !strings.Contains(err.Error(), "prediction failed") {
		t.Fatalf("error = %v, want prediction failed", err)
	}
}

func TestChatCompletionContextCanceled(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(func() { close(release); server.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	p := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 5})

	errc := make(chan error, 1)
	go func() {
		_, err := p.ChatCompletion(ctx, testKey(), &providers.ChatRequest{Model: "owner/model"})
		errc <- err
	}()
	cancel()
	err := <-errc
	if err == nil {
		t.Fatal("expected error after context cancel")
	}
}

func TestChatCompletionPollUsesURLsGet(t *testing.T) {
	var pollPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/predictions":
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"processing","urls":{"get":"/v1/predictions/pred-1"}}`))
		case "/v1/predictions/pred-1":
			pollPath = r.URL.Path
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"succeeded","output":"done"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 5})
	resp, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if pollPath != "/v1/predictions/pred-1" {
		t.Fatalf("pollPath = %q", pollPath)
	}
	if resp.Choices[0].Message.Content != "done" {
		t.Fatalf("content = %q", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletionPollHTTPError(t *testing.T) {
	var count int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		count++
		if count == 1 {
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"processing","urls":{"get":"/v1/predictions/pred-1"}}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		}
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 5})
	_, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("error = %v, want poll HTTP error", err)
	}
}

func TestPredictionOutputTextUnsupported(t *testing.T) {
	_, err := predictionOutputText(42)
	if err == nil {
		t.Fatal("expected error for unsupported output type")
	}
}

func TestPredictionOutputTextArrayNonString(t *testing.T) {
	_, err := predictionOutputText([]any{42})
	if err == nil {
		t.Fatal("expected error for non-string array item")
	}
}

func TestPredictionOutputTextNil(t *testing.T) {
	text, err := predictionOutputText(nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty string, got %q", text)
	}
}

func TestFlattenMessagesSkipsEmpty(t *testing.T) {
	msgs := []providers.Message{
		{Role: "user", Content: ""},
		{Role: "user", Content: "hello"},
	}
	result := flattenMessages(msgs)
	if result != "user: hello" {
		t.Fatalf("flattenMessages = %q", result)
	}
}

func TestFlattenMessagesEmptyRole(t *testing.T) {
	msgs := []providers.Message{
		{Role: "", Content: "hi"},
	}
	result := flattenMessages(msgs)
	if result != "user: hi" {
		t.Fatalf("flattenMessages = %q, want user role fallback", result)
	}
}

func TestContentTextTypes(t *testing.T) {
	if contentText(nil) != "" {
		t.Fatal("nil should give empty string")
	}
	if contentText([]byte("bytes")) != "bytes" {
		t.Fatal("[]byte should give string")
	}
	// non-string marshallable value
	result := contentText(map[string]any{"k": "v"})
	if result == "" {
		t.Fatal("map should give non-empty JSON string")
	}
}

func TestNonEmptyHelper(t *testing.T) {
	if nonEmpty("", "b", "c") != "b" {
		t.Fatal("expected first non-empty")
	}
	if nonEmpty("", "", "") != "" {
		t.Fatal("expected empty when all empty")
	}
}

func TestSleepContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sleepContext(ctx, time.Hour)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestChatCompletionOutputNil(t *testing.T) {
	// succeeded but output is null → empty content, no error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/predictions":
			_, _ = w.Write([]byte(`{"id":"pred-1","status":"succeeded","output":null}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL})
	resp, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if resp.Choices[0].Message.Content != "" {
		t.Fatalf("content = %q, want empty", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletionOutputUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"pred-1","status":"succeeded","output":{"nested":"object"}}`))
	}))
	t.Cleanup(server.Close)

	p := New(Config{BaseURL: server.URL})
	_, err := p.ChatCompletion(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil {
		t.Fatal("expected error for unsupported output type")
	}
}

func TestNewRequestAbsoluteURL(t *testing.T) {
	p := New(Config{BaseURL: "https://base.example.com"})
	req, err := p.newRequest(context.Background(), http.MethodGet, "https://other.example.com/path", testKey(), nil)
	if err != nil {
		t.Fatalf("newRequest: %v", err)
	}
	if req.URL.Host != "other.example.com" {
		t.Fatalf("host = %q, want other.example.com (absolute URL not modified)", req.URL.Host)
	}
}
