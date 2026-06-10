package api

import (
	"errors"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// failingWriter errors after allowing a fixed number of writes, so tests
// can simulate a client disconnect mid-stream (AUD-008).
type failingWriter struct {
	sb         strings.Builder
	writesLeft int
}

func (f *failingWriter) Write(p []byte) (int, error) {
	if f.writesLeft <= 0 {
		return 0, errors.New("simulated client disconnect")
	}
	f.writesLeft--
	return f.sb.Write(p)
}

func (f *failingWriter) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

// TestWriteSSEStreamSuccess verifies normal SSE framing plus the [DONE]
// terminator when every marshal and write succeeds.
func TestWriteSSEStreamSuccess(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1"}
	ch <- &schemas.StreamChunk{ID: "c2"}
	close(ch)

	w := &failingWriter{writesLeft: 100}
	if err := writeSSEStream(w, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.sb.String()
	if got := strings.Count(out, "data: "); got != 3 {
		t.Errorf("frame count = %d, want 3 (2 chunks + [DONE])", got)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("output missing [DONE] terminator: %q", out)
	}
}

// TestWriteSSEStreamAbortsBeforeFirstFrame verifies a write failure before any
// frame is emitted leaves empty output and returns an error. Marshal failures on
// the passthrough path are covered by translation/stream_test error-path tests.
func TestWriteSSEStreamAbortsBeforeFirstFrame(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1"}
	close(ch)

	w := &failingWriter{writesLeft: 0}
	if err := writeSSEStream(w, ch); err == nil {
		t.Fatal("writeSSEStream: want error when first write fails")
	}
	if out := w.sb.String(); out != "" {
		t.Errorf("output = %q, want empty when write fails before first frame", out)
	}
}

// TestWriteSSEStreamAbortsOnWriteError verifies AUD-008: a write failure
// stops the loop instead of continuing to consume and emit frames.
func TestWriteSSEStreamAbortsOnWriteError(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1"}
	ch <- &schemas.StreamChunk{ID: "c2"}
	close(ch)

	w := &failingWriter{writesLeft: 1} // first full SSE frame succeeds; [DONE] write fails
	err := writeSSEStream(w, ch)
	if err == nil {
		t.Fatal("writeSSEStream: want error when write fails mid-stream")
	}

	out := w.sb.String()
	if got := strings.Count(out, "data: "); got != 1 {
		t.Errorf("frame count = %d, want 1 (first chunk only before write failure)", got)
	}
	if strings.Contains(out, "[DONE]") {
		t.Errorf("output contains [DONE] after write failure: %q", out)
	}
}

// TestWriteSSEStreamAbortsOnErrorChunk verifies AUD-046/047: an in-band
// terminal error chunk aborts the stream — no frame for it, no [DONE].
func TestWriteSSEStreamAbortsOnErrorChunk(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "c1"}
	ch <- &schemas.StreamChunk{Error: &schemas.ProviderError{Message: "boom", Type: "stream_error"}}
	ch <- &schemas.StreamChunk{ID: "c2"}
	close(ch)

	w := &failingWriter{writesLeft: 100}
	if err := writeSSEStream(w, ch); err == nil {
		t.Fatal("writeSSEStream: want error on in-band error chunk")
	}

	out := w.sb.String()
	if got := strings.Count(out, "data: "); got != 1 {
		t.Errorf("frame count = %d, want 1 (only the chunk before the error)", got)
	}
	if strings.Contains(out, "[DONE]") {
		t.Errorf("output contains [DONE] after error chunk: %q", out)
	}
	if strings.Contains(out, "boom") {
		t.Errorf("error chunk leaked to client: %q", out)
	}
}

// TestWriteSSEStreamAbortsOnMarshalError verifies AUD-007: a chunk that
// fails to marshal aborts the stream — no partial frame, no [DONE].
func TestWriteSSEStreamAbortsOnMarshalError(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{
		ID: "c1",
		Choices: []schemas.StreamChoice{{
			Index: 0,
			Logprobs: &schemas.Logprobs{
				Content: []schemas.LogprobContent{{
					Token:   "x",
					Logprob: math.Inf(1),
				}},
			},
		}},
	}
	ch <- &schemas.StreamChunk{ID: "c2"}
	close(ch)

	w := &failingWriter{writesLeft: 100}
	err := writeSSEStream(w, ch)
	if err == nil {
		t.Fatal("writeSSEStream: want error when marshal fails")
	}
	if out := w.sb.String(); out != "" {
		t.Errorf("output = %q, want empty (stream must abort before writing)", out)
	}
}

// TestChatStreamPassthroughNormalization verifies PAR-TRANS-049 passthrough
// normalization: invalid IDs are fixed (stream.js:106), required fields are
// injected (stream.js:109-112), empty chunks are filtered (stream.js:129-131),
// and the stream terminates with [DONE] (stream.js:339-345).
func TestChatStreamPassthroughNormalization(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "chat", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}}}} // empty → filtered
	close(ch)

	w := &failingWriter{writesLeft: 100}
	if err := writeSSEStream(w, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.sb.String()
	if strings.Contains(out, `"id":"chat"`) {
		t.Errorf("invalid id not fixed: %q", out)
	}
	if !strings.Contains(out, "chatcmpl-") {
		t.Errorf("expected fixed id with chatcmpl- prefix: %q", out)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
	// Empty chunk filtered → only 2 data frames (1 chunk + DONE)
	if got := strings.Count(out, "data: "); got != 2 {
		t.Errorf("frame count = %d, want 2 (1 chunk + [DONE])", got)
	}
	// Required fields injected.
	if !strings.Contains(out, `"object":"chat.completion.chunk"`) {
		t.Errorf("missing object field: %q", out)
	}
	if !strings.Contains(out, `"created":`) {
		t.Errorf("missing created field: %q", out)
	}
}

// TestChatHandlerPassthroughNormalization drives ChatHandler end-to-end with a
// fake provider stream and verifies passthrough normalization at the handler
// level: invalid IDs are fixed, required fields are injected, empty chunks are
// filtered, and the stream terminates with [DONE].
func TestChatHandlerPassthroughNormalization(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{
		ID:                  "chat",
		Choices:             []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}, ContentFilterResults: map[string]any{"hate": map[string]any{}}}},
		PromptFilterResults: []any{map[string]any{}},
	}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}}}} // empty → filtered
	close(ch)

	fakeR := &fakeMessagesResolver{streamCh: ch}
	h := &ChatHandler{router: fakeR}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}],"stream":true}`))
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want %d", ctx.Response.StatusCode(), fasthttp.StatusOK)
	}

	out := string(ctx.Response.Body())
	// Invalid id fixed.
	if strings.Contains(out, `"id":"chat"`) {
		t.Errorf("invalid id not fixed: %q", out)
	}
	if !strings.Contains(out, "chatcmpl-") {
		t.Errorf("expected fixed id with chatcmpl- prefix: %q", out)
	}
	// Required fields injected.
	if !strings.Contains(out, `"object":"chat.completion.chunk"`) {
		t.Errorf("missing object field: %q", out)
	}
	if !strings.Contains(out, `"created":`) {
		t.Errorf("missing created field: %q", out)
	}
	// Azure fields stripped: chunk carried them, output must not contain them.
	if !strings.Contains(out, "hi") {
		t.Errorf("expected chunk content in output: %q", out)
	}
	if strings.Contains(out, "prompt_filter_results") {
		t.Errorf("prompt_filter_results should not appear: %q", out)
	}
	if strings.Contains(out, "content_filter_results") {
		t.Errorf("content_filter_results should not appear: %q", out)
	}
	// Empty chunk filtered → only 2 data frames (1 chunk + DONE).
	if got := strings.Count(out, "data: "); got != 2 {
		t.Errorf("frame count = %d, want 2 (1 chunk + [DONE])", got)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
}

// TestChatHandlerMarshalFailureFallsBackTo500 verifies that when the
// response marshal seam fails, the chat handler eventually writes a 500
// status (AUD-009). The provider will fail with a network error in this
// test environment, which routes through writeError — writeError then
// exercises the same failing jsonMarshal seam, and falls back to a
// plain-text 500 per the AUD-009–012 acceptance contract.
func TestChatHandlerMarshalFailureFallsBackTo500(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	router := inference.NewRouter()
	h := NewChatHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi"}]}`))
	h.Handle(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusInternalServerError)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want %q", got, "internal error")
	}
}

// fakeModelResolver captures the request passed to ResolveForModel and returns
// a predetermined error, so tests can observe preprocessing behaviorally.
type fakeModelResolver struct {
	captured *schemas.ChatRequest
}

func (f *fakeModelResolver) ResolveForModel(req *schemas.ChatRequest) (schemas.Provider, schemas.Key, error) {
	f.captured = req
	return nil, schemas.Key{}, errors.New("simulated resolve error")
}

// TestChatHandlerPreprocessesRequest verifies that Handle calls
// translation.PreprocessChatRequest before ResolveForModel: a body with an
// invalid tool_call ID and no tool response arrives at the resolver with the
// ID sanitized and a role:tool message inserted.
func TestChatHandlerPreprocessesRequest(t *testing.T) {
	fake := &fakeModelResolver{}
	h := &ChatHandler{router: fake}

	body := `{"model":"gpt-4","messages":[{"role":"user","content":"hi"},{"role":"assistant","tool_calls":[{"id":"bad!id","type":"","function":{"name":"foo","arguments":"{}"}}]},{"role":"user","content":"next"}]}`

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(body))

	// Ensure the handler still returns 400 because the resolver errors.
	// The behavioral proof is in the captured request.
	// If PreprocessChatRequest is not called, the captured request will have
	// the original invalid ID and no inserted tool message.
	h.Handle(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", ctx.Response.StatusCode(), fasthttp.StatusBadRequest)
	}

	if fake.captured == nil {
		t.Fatal("resolver was never called")
	}

	msgs := fake.captured.Messages
	if len(msgs) != 4 {
		t.Fatalf("len(messages) = %d, want 4 (user, assistant, tool, user)", len(msgs))
	}

	// Assistant message should have sanitized ID.
	if msgs[1].ToolCalls[0].ID != "badid" {
		t.Errorf("assistant tool_call ID = %q, want %q", msgs[1].ToolCalls[0].ID, "badid")
	}

	// Inserted tool message should carry the sanitized ID.
	if msgs[2].Role != "tool" || msgs[2].ToolCallID == nil || *msgs[2].ToolCallID != "badid" {
		t.Errorf("inserted tool msg = %+v, want role:tool tool_call_id:badid", msgs[2])
	}

	// Verify that the Type was also set by preprocessing.
	if msgs[1].ToolCalls[0].Type != "function" {
		t.Errorf("tool_call type = %q, want %q", msgs[1].ToolCalls[0].Type, "function")
	}
}
