package replicate

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

// TestReplicateErrorChunkEmptyMessage covers the "" → default message branch.
func TestReplicateErrorChunkEmptyMessage(t *testing.T) {
	chunk := replicateErrorChunk("owner/model", "")
	if chunk.Error == nil {
		t.Fatal("expected non-nil Error")
	}
	if chunk.Error.Message != "replicate stream error" {
		t.Fatalf("message = %q, want default", chunk.Error.Message)
	}
}

// TestReplicateErrorChunkExplicitMessage covers the non-empty branch.
func TestReplicateErrorChunkExplicitMessage(t *testing.T) {
	chunk := replicateErrorChunk("owner/model", "quota exceeded")
	if chunk.Error.Message != "quota exceeded" {
		t.Fatalf("message = %q, want quota exceeded", chunk.Error.Message)
	}
	if chunk.Error.Code != "upstream_stream_error" {
		t.Fatalf("code = %q", chunk.Error.Code)
	}
}

// TestHandleSSEEventOutputEmpty covers the data == "" early-return inside "output".
func TestHandleSSEEventOutputEmpty(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 4)
	stop := handleSSEEvent("m", "output", "", chunks)
	if stop {
		t.Fatal("empty output must not stop stream")
	}
	if len(chunks) != 0 {
		t.Fatal("empty output must not emit a chunk")
	}
}

// TestHandleSSEEventUnknown covers the default case (no chunk, no stop).
func TestHandleSSEEventUnknown(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 4)
	stop := handleSSEEvent("m", "logs", "some log line", chunks)
	if stop {
		t.Fatal("unknown event must not stop stream")
	}
	if len(chunks) != 0 {
		t.Fatal("unknown event must not emit a chunk")
	}
}

// TestHandleSSEEventError covers the "error" event path.
func TestHandleSSEEventError(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 4)
	stop := handleSSEEvent("m", "error", "model crashed", chunks)
	if !stop {
		t.Fatal("error event must stop stream")
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 error chunk, got %d", len(chunks))
	}
	c := <-chunks
	if c.Error == nil || c.Error.Message != "model crashed" {
		t.Fatalf("unexpected error chunk: %+v", c.Error)
	}
}

// TestHandleSSEEventDone covers the "done" event stop path.
func TestHandleSSEEventDone(t *testing.T) {
	chunks := make(chan providers.StreamChunk, 4)
	stop := handleSSEEvent("m", "done", "{}", chunks)
	if !stop {
		t.Fatal("done event must stop stream")
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 done chunk, got %d", len(chunks))
	}
	c := <-chunks
	if len(c.Choices) == 0 || c.Choices[0].FinishReason == nil || *c.Choices[0].FinishReason != "stop" {
		t.Fatalf("done chunk has unexpected finish: %+v", c.Choices)
	}
}

// TestStreamSSEScannerErrorPath covers the scanner.Err() != nil branch.
// We feed a body larger than the scanner buffer limit to trigger an error.
func TestStreamSSEScannerErrorPath(t *testing.T) {
	// Build a single data: line exceeding 1 MiB to overflow the scanner.
	giant := "data: " + string(make([]byte, 2*1024*1024)) + "\n\n"
	chunks := make(chan providers.StreamChunk, 8)

	// streamSSE is sync — run it in a goroutine with a buffered channel so it
	// doesn't block on the closed reader.
	done := make(chan struct{})
	go func() {
		defer close(done)
		streamSSE("m", newStringReader(giant), chunks)
		close(chunks)
	}()
	<-done

	// Drain any emitted chunks; we just need no panic and the function returned.
}

// newStringReader is a simple io.Reader backed by a string.
type stringReader struct{ s string; pos int }

func newStringReader(s string) *stringReader { return &stringReader{s: s} }

func (r *stringReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.s) {
		return 0, nil // simulate blocked/empty — caller eventually gives up
	}
	n := copy(p, r.s[r.pos:])
	r.pos += n
	return n, nil
}

// TestStreamSSEDispatchAtEOFWithPendingEvent covers the final dispatch() call
// after the scan loop when the last event has no trailing blank line.
func TestStreamSSEDispatchAtEOFWithPendingEvent(t *testing.T) {
	// SSE without trailing blank line — the final dispatch must still fire.
	sse := "event: output\ndata: token"
	chunks := make(chan providers.StreamChunk, 8)
	streamSSE("m", newStringReader(sse), chunks)
	close(chunks)

	var content string
	for c := range chunks {
		for _, ch := range c.Choices {
			if ch.Delta.Content != nil {
				content += *ch.Delta.Content
			}
		}
	}
	if content != "token" {
		t.Fatalf("content = %q, want token", content)
	}
}
