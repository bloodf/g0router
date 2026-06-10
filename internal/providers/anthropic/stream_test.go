package anthropic

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestChatCompletionStreamAbortsOnMalformedChunk verifies AUD-045: a stream
// event that fails JSON unmarshal aborts the stream instead of being
// silently skipped. The valid event after the malformed line must never
// arrive.
func TestChatCompletionStreamAbortsOnMalformedChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"a\"}}\n\n")
		io.WriteString(w, "data: not-json{\n\n")
		io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"b\"}}\n\n")
		io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "claude-3-opus"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var content, errChunks int
	for chunk := range ch {
		if chunk.Error != nil {
			errChunks++
			continue
		}
		content++
	}
	if content != 1 {
		t.Errorf("content chunks = %d, want 1 (stream must abort at malformed event)", content)
	}
	if errChunks != 1 {
		t.Errorf("error chunks = %d, want 1 (abort must be distinguishable from clean EOF)", errChunks)
	}
}

// errorHook is a PostHookRunner that fails on every chunk.
type errorHook struct{}

func (errorHook) Run(ctx *schemas.GatewayContext, response any) error {
	return errors.New("simulated hook failure")
}

// TestChatCompletionStreamSurfacesHookError verifies AUD-047: a failing
// post-hook aborts the stream with an in-band error chunk instead of being
// silently discarded.
func TestChatCompletionStreamSurfacesHookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"a\"}}\n\n")
		io.WriteString(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"b\"}}\n\n")
		io.WriteString(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, errorHook{}, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "claude-3-opus"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var content, errChunks int
	for chunk := range ch {
		if chunk.Error != nil {
			errChunks++
			continue
		}
		content++
	}
	if content != 1 {
		t.Errorf("content chunks = %d, want 1 (stream must stop after failed hook)", content)
	}
	if errChunks != 1 {
		t.Errorf("error chunks = %d, want 1 (hook error must be surfaced)", errChunks)
	}
}
