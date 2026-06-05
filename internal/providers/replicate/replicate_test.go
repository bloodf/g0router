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

func TestChatCompletionStreamUnsupported(t *testing.T) {
	provider := New(Config{BaseURL: "https://replicate.example"})
	_, err := provider.ChatCompletionStream(context.Background(), testKey(), &providers.ChatRequest{Model: "owner/model"})
	if err == nil || !strings.Contains(err.Error(), "streaming unsupported") {
		t.Fatalf("error = %v, want streaming unsupported", err)
	}
	if !errors.Is(err, providers.ErrStreamingUnsupported) {
		t.Fatalf("error = %v, want ErrStreamingUnsupported", err)
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
