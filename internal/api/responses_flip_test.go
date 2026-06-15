package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// TestResponsesStreamEventTypingAndDone regression-locks the already-satisfied
// bf-openai parity rows that the matrix listed MISSING but the live tree HAS:
//
//   - PAR-BF-OAI-003 — POST /v1/responses route is wired (the handler runs and
//     translates the responses body to a chat dispatch).
//   - PAR-BF-OAI-208 — the responses path streams (ChatCompletionStream is opened
//     and piped through ProcessTranslateStream).
//   - PAR-BF-OAI-203 — `event: <type>` SSE typing is emitted for the responses
//     stream (response.created / response.output_text.delta / response.completed).
//   - PAR-BF-OAI-204 — the responses stream terminates with `data: [DONE]`.
//
// The behavior is already present (sse.go:144-153, stream.go:136); this test
// pins it so a future change cannot silently regress the flip.
func TestResponsesStreamEventTypingAndDone(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{ID: "chatcmpl-x", Model: "gpt-4", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	fake := &fakeResponsesResolver{streamCh: ch}
	h := &ResponsesHandler{router: fake, registry: translation.NewRegistry()}

	body := `{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(body))
	h.Handle(&ctx)

	// 003 + 208: the handler ran, resolved, and opened the provider stream.
	if !fake.streamCalled {
		t.Fatal("003/208: responses handler did not open the provider stream")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	got := string(ctx.Response.Body())

	// 203: typed SSE event framing for the responses stream.
	for _, want := range []string{
		"event: response.created",
		"event: response.output_text.delta",
		"event: response.completed",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("203: missing %q in responses stream framing; output:\n%s", want, got)
		}
	}

	// 204: the responses stream terminates with the [DONE] marker.
	if !strings.Contains(got, "data: [DONE]") {
		t.Errorf("204: missing terminal data: [DONE] in responses stream; output:\n%s", got)
	}
}
