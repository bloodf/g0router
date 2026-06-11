package ollama

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
)

// TestOllamaChatURLLocalVsCloud verifies chatURL resolves correctly.
func TestOllamaChatURLLocalVsCloud(t *testing.T) {
	reg := translation.NewRegistry()

	local, err := New("ollama-local", reg)
	if err != nil {
		t.Fatal(err)
	}
	if got := local.chatURL(); got != "http://localhost:11434/api/chat" {
		t.Errorf("ollama-local chatURL = %q, want %q", got, "http://localhost:11434/api/chat")
	}

	cloud, err := New("ollama", reg)
	if err != nil {
		t.Fatal(err)
	}
	if got := cloud.chatURL(); got != cloud.config.BaseURL {
		t.Errorf("ollama chatURL = %q, want config.BaseURL %q", got, cloud.config.BaseURL)
	}
}

// TestOllamaChatNoAuthHeader verifies no Authorization header is sent.
func TestOllamaChatNoAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model":"llama3","message":{"role":"assistant","content":"hi"},"done":true}`))
	}))
	defer srv.Close()

	p, _ := New("ollama", translation.NewRegistry())
	p.config.BaseURL = srv.URL

	_, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: "unused"}, &schemas.ChatRequest{Model: "llama3"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty", gotAuth)
	}
}

// TestOllamaChatNonStreaming verifies a single JSON response is translated.
func TestOllamaChatNonStreaming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"model": "llama3",
			"message": {"role": "assistant", "content": "hello world"},
			"done": true,
			"prompt_eval_count": 10,
			"eval_count": 5
		}`))
	}))
	defer srv.Close()

	p, _ := New("ollama", translation.NewRegistry())
	p.config.BaseURL = srv.URL

	resp, perr := p.ChatCompletion(&schemas.GatewayContext{}, schemas.Key{Value: ""}, &schemas.ChatRequest{Model: "llama3"})
	if perr != nil {
		t.Fatalf("ChatCompletion error: %v", perr.Message)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}
	if resp.Choices[0].Message == nil {
		t.Fatal("expected non-nil message")
	}
	if resp.Choices[0].Message.Content != "hello world" {
		t.Errorf("content = %q, want %q", resp.Choices[0].Message.Content, "hello world")
	}
	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("finish_reason = %q, want stop", resp.Choices[0].FinishReason)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("prompt_tokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("completion_tokens = %d, want 5", resp.Usage.CompletionTokens)
	}
}

// TestOllamaStreamNDJSON verifies NDJSON lines are translated to OpenAI chunks.
func TestOllamaStreamNDJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Write([]byte(`{"model":"llama3","message":{"role":"assistant","content":"hello"},"done":false}` + "\n"))
		w.Write([]byte(`{"model":"llama3","message":{"role":"assistant","content":" world"},"done":false}` + "\n"))
		w.Write([]byte(`{"done":true,"prompt_eval_count":10,"eval_count":5}` + "\n"))
	}))
	defer srv.Close()

	p, _ := New("ollama", translation.NewRegistry())
	p.config.BaseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: ""}, &schemas.ChatRequest{Model: "llama3"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var contents []string
	var doneCount, errCount int
	for chunk := range ch {
		if chunk.Error != nil {
			errCount++
			continue
		}
		if len(chunk.Choices) > 0 {
			if chunk.Choices[0].Delta.Content != "" {
				contents = append(contents, chunk.Choices[0].Delta.Content)
			}
			if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason == "stop" {
				doneCount++
			}
		}
	}

	if errCount != 0 {
		t.Errorf("error chunks = %d, want 0", errCount)
	}
	if len(contents) != 2 {
		t.Errorf("content chunks = %d, want 2", len(contents))
	}
	if contents[0] != "hello" {
		t.Errorf("first content = %q, want hello", contents[0])
	}
	if contents[1] != " world" {
		t.Errorf("second content = %q, want \" world\"", contents[1])
	}
	if doneCount != 1 {
		t.Errorf("done chunks = %d, want 1", doneCount)
	}
}

// failingPostHookRunner returns an error on every invocation.
type failingPostHookRunner struct{}

func (f *failingPostHookRunner) Run(ctx *schemas.GatewayContext, response any) error {
	return fmt.Errorf("hook refused")
}

// TestOllamaStreamPostHookError verifies that a post-hook error after a valid
// chunk emits the chunk, then an in-band streamError, then closes the channel.
func TestOllamaStreamPostHookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Write([]byte(`{"model":"llama3","message":{"role":"assistant","content":"hello"},"done":false}` + "\n"))
		w.Write([]byte(`{"done":true}` + "\n"))
	}))
	defer srv.Close()

	p, _ := New("ollama", translation.NewRegistry())
	p.config.BaseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, &failingPostHookRunner{}, schemas.Key{Value: ""}, &schemas.ChatRequest{Model: "llama3"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var contentCount, errCount int
	for chunk := range ch {
		if chunk.Error != nil {
			errCount++
			if chunk.Error.Type != "stream_error" {
				t.Errorf("error type = %q, want stream_error", chunk.Error.Type)
			}
			if chunk.Error.Message == "" {
				t.Error("expected non-empty error message")
			}
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			contentCount++
			if chunk.Choices[0].Delta.Content != "hello" {
				t.Errorf("content = %q, want hello", chunk.Choices[0].Delta.Content)
			}
		}
	}

	if contentCount != 1 {
		t.Errorf("content chunks = %d, want 1", contentCount)
	}
	if errCount != 1 {
		t.Errorf("error chunks = %d, want 1", errCount)
	}
}

// TestOllamaStreamSkipsMalformedNDJSON verifies that the NDJSON scanner skips
// malformed lines (json.Valid failure) without surfacing an in-band error chunk.
func TestOllamaStreamSkipsMalformedNDJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Write([]byte(`{"model":"llama3","message":{"role":"assistant","content":"hello"},"done":false}` + "\n"))
		w.Write([]byte(`{not json` + "\n"))
		w.Write([]byte(`{"done":true}` + "\n"))
	}))
	defer srv.Close()

	p, _ := New("ollama", translation.NewRegistry())
	p.config.BaseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: ""}, &schemas.ChatRequest{Model: "llama3"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var contentCount, errCount int
	for chunk := range ch {
		if chunk.Error != nil {
			errCount++
			continue
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			contentCount++
		}
	}
	if contentCount != 1 {
		t.Errorf("content chunks = %d, want 1 (valid chunk before malformed line)", contentCount)
	}
	if errCount != 0 {
		t.Errorf("error chunks = %d, want 0 (malformed NDJSON must be silently skipped)", errCount)
	}
}
