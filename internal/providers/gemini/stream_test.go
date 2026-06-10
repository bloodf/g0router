package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
)

// TestChatCompletionStreamAbortsOnMalformedChunk verifies AUD-045: a chunk
// that fails JSON unmarshal aborts the stream instead of being silently
// skipped. The valid chunk after the malformed line must never arrive.
func TestChatCompletionStreamAbortsOnMalformedChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"a\"}]}}]}\n\n")
		io.WriteString(w, "data: not-json{\n\n")
		io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"b\"}]}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	p := NewProvider()
	p.baseURL = srv.URL

	ch, perr := p.ChatCompletionStream(&schemas.GatewayContext{}, nil, schemas.Key{Value: "test-key"}, &schemas.ChatRequest{Model: "gemini-pro"})
	if perr != nil {
		t.Fatalf("ChatCompletionStream error: %v", perr.Message)
	}

	var got int
	for range ch {
		got++
	}
	if got != 1 {
		t.Errorf("chunks received = %d, want 1 (stream must abort at malformed chunk)", got)
	}
}
