package kiro

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// kiroEventStreamBody assembles a canned eventstream response body: an assistant
// text frame followed by a messageStop frame.
func kiroEventStreamBody() []byte {
	var buf []byte
	buf = append(buf, buildFrame(
		[][2]string{{":message-type", "event"}, {":event-type", "assistantResponseEvent"}},
		[]byte(`{"content":"Hello world"}`),
	)...)
	buf = append(buf, buildFrame(
		[][2]string{{":message-type", "event"}, {":event-type", "messageStopEvent"}},
		[]byte(`{}`),
	)...)
	return buf
}

// TestKiroNewRejectsWrongFormat verifies the adapter constructor enforces the
// catalog Format.
func TestKiroNewRejectsWrongFormat(t *testing.T) {
	reg := translation.NewRegistry()
	if _, err := New("openai", reg); err == nil {
		t.Fatal("New(openai) error = nil, want error (format mismatch)")
	}
}

// TestKiroChatCompletionStream verifies the adapter POSTs to the kiro endpoint,
// decodes the eventstream frames, runs them through kiroToOpenAIResponse via the
// registry, and emits OpenAI stream chunks.
func TestKiroChatCompletionStream(t *testing.T) {
	var gotAuth, gotAccept, gotTarget string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotTarget = r.Header.Get("X-Amz-Target")
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		w.Write(kiroEventStreamBody())
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, err := New("kiro", reg)
	if err != nil {
		t.Fatalf("New(kiro) error: %v", err)
	}
	p.urlOverride = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "kiro-token"},
		&schemas.ChatRequest{Model: "claude-sonnet-4.5", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var content string
	var finishSeen bool
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("error chunk: %v", chunk.Error.Message)
		}
		for _, c := range chunk.Choices {
			content += c.Delta.Content
			if c.FinishReason != nil && *c.FinishReason == "stop" {
				finishSeen = true
			}
		}
	}
	if content != "Hello world" {
		t.Errorf("content = %q, want %q", content, "Hello world")
	}
	if !finishSeen {
		t.Error("no stop finish_reason chunk emitted")
	}

	if gotAuth != "Bearer kiro-token" {
		t.Errorf("Authorization = %q, want Bearer kiro-token", gotAuth)
	}
	if gotAccept != "application/vnd.amazon.eventstream" {
		t.Errorf("Accept = %q, want application/vnd.amazon.eventstream", gotAccept)
	}
	if gotTarget != "AmazonCodeWhispererStreamingService.GenerateAssistantResponse" {
		t.Errorf("X-Amz-Target = %q, want the CodeWhisperer target", gotTarget)
	}
}

// TestKiroChatCompletion verifies the non-streaming path aggregates the decoded
// frames into a single ChatResponse.
func TestKiroChatCompletion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		w.Write(kiroEventStreamBody())
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, _ := New("kiro", reg)
	p.urlOverride = srv.URL

	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "k"},
		&schemas.ChatRequest{Model: "claude-sonnet-4.5", Messages: []schemas.Message{{Role: "user", Content: "hi"}}})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if resp == nil || len(resp.Choices) == 0 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Choices[0].Message.Content != "Hello world" {
		t.Errorf("content = %q, want %q", resp.Choices[0].Message.Content, "Hello world")
	}
}

// TestKiroErrorStatus verifies a non-200 upstream surfaces a provider error.
func TestKiroErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer srv.Close()

	reg := translation.NewRegistry()
	p, _ := New("kiro", reg)
	p.urlOverride = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "k"},
		&schemas.ChatRequest{Model: "claude-sonnet-4.5"})
	if perr == nil {
		t.Fatal("expected provider error for 429, got nil")
	}
	if perr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", perr.StatusCode)
	}
}
