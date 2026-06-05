package replicate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

func collectStream(t *testing.T, ch <-chan providers.StreamChunk) []providers.StreamChunk {
	t.Helper()
	var chunks []providers.StreamChunk
	timeout := time.After(2 * time.Second)
	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				return chunks
			}
			chunks = append(chunks, chunk)
		case <-timeout:
			t.Fatal("timed out waiting for stream to close")
		}
	}
}

// replicateStreamServer wires a predict endpoint that returns a prediction whose
// urls.stream points at a second SSE handler emitting the token events.
func replicateStreamServer(t *testing.T, sse string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/v1/predictions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := fmt.Sprintf(`{"id":"pred-1","status":"processing","urls":{"stream":"%s/stream","get":"/v1/predictions/pred-1"}}`, server.URL)
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sse))
	})
	return server
}

func TestChatCompletionStreamHappyMultiToken(t *testing.T) {
	sse := "event: output\ndata: Hello\n\n" +
		"event: output\ndata:  world\n\n" +
		"event: done\ndata: {}\n\n"
	server := replicateStreamServer(t, sse)

	provider := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 5})
	ch, err := provider.ChatCompletionStream(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "owner/model",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	chunks := collectStream(t, ch)

	var content string
	var finish *string
	for _, chunk := range chunks {
		if chunk.Error != nil {
			t.Fatalf("unexpected error chunk: %+v", chunk.Error)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != nil {
				content += *choice.Delta.Content
			}
			if choice.FinishReason != nil {
				finish = choice.FinishReason
			}
		}
	}
	if content != "Hello world" {
		t.Errorf("content = %q, want %q", content, "Hello world")
	}
	if finish == nil || *finish != "stop" {
		t.Errorf("finish = %v, want stop", finish)
	}
}

func TestChatCompletionStreamEmitsErrorEvent(t *testing.T) {
	sse := "event: output\ndata: partial\n\n" +
		"event: error\ndata: model exploded\n\n"
	server := replicateStreamServer(t, sse)

	provider := New(Config{BaseURL: server.URL, PollInterval: time.Nanosecond, MaxPolls: 5})
	ch, err := provider.ChatCompletionStream(context.Background(), testKey(), &providers.ChatRequest{
		Model:    "owner/model",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletionStream: %v", err)
	}
	chunks := collectStream(t, ch)

	var content string
	var sawError bool
	for _, chunk := range chunks {
		if chunk.Error != nil {
			sawError = true
			if chunk.Error.Message == "" {
				t.Errorf("error chunk missing message")
			}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != nil {
				content += *choice.Delta.Content
			}
		}
	}
	if content != "partial" {
		t.Errorf("content = %q, want partial before error", content)
	}
	if !sawError {
		t.Fatal("expected an error chunk for the error event")
	}
}

func TestChatCompletionStreamRequiresKey(t *testing.T) {
	provider := New(Config{BaseURL: "https://replicate.example"})
	_, err := provider.ChatCompletionStream(context.Background(), providers.Key{}, &providers.ChatRequest{Model: "owner/model"})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestChatCompletionStreamRejectsNilRequest(t *testing.T) {
	provider := New(Config{BaseURL: "https://replicate.example"})
	_, err := provider.ChatCompletionStream(context.Background(), testKey(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}
