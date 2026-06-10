package api

import (
	"errors"
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
	writeSSEStream(w, ch)

	out := w.sb.String()
	if got := strings.Count(out, "data: "); got != 3 {
		t.Errorf("frame count = %d, want 3 (2 chunks + [DONE])", got)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("output missing [DONE] terminator: %q", out)
	}
}

// TestWriteSSEStreamAbortsOnMarshalError verifies AUD-007: a chunk that
// fails to marshal aborts the stream — no partial frame, no [DONE].
func TestWriteSSEStreamAbortsOnMarshalError(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1"}
	ch <- &schemas.StreamChunk{ID: "c2"}
	close(ch)

	w := &failingWriter{writesLeft: 100}
	writeSSEStream(w, ch)

	if out := w.sb.String(); out != "" {
		t.Errorf("output = %q, want empty (stream must abort before writing)", out)
	}
}

// TestWriteSSEStreamAbortsOnWriteError verifies AUD-008: a write failure
// stops the loop instead of continuing to consume and emit frames.
func TestWriteSSEStreamAbortsOnWriteError(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1"}
	ch <- &schemas.StreamChunk{ID: "c2"}
	close(ch)

	w := &failingWriter{writesLeft: 1} // "data: " succeeds, chunk body write fails
	writeSSEStream(w, ch)

	out := w.sb.String()
	if out != "data: " {
		t.Errorf("output = %q, want %q (must stop at first write failure)", out, "data: ")
	}
	if strings.Contains(out, "[DONE]") {
		t.Errorf("output contains [DONE] after write failure: %q", out)
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
