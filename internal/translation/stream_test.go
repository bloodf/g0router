package translation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
)

// fakeWriter implements io.Writer and can optionally fail.
type fakeWriter struct {
	buf        bytes.Buffer
	failAfter  int
	writeCount int
}

func (f *fakeWriter) Write(p []byte) (int, error) {
	if f.failAfter > 0 && f.writeCount >= f.failAfter {
		return 0, errors.New("simulated write failure")
	}
	f.writeCount++
	return f.buf.Write(p)
}

// --- Task 4: ProcessTranslateStream tests ---

func TestProcessTranslateStreamFiltersEmptyChunks(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}}}} // empty
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	_, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if !strings.Contains(out, "hi") {
		t.Errorf("output missing 'hi': %q", out)
	}
	// Empty OpenAI delta → translated to empty content_block_delta → filtered by HasValuableContent.
	// The first chunk fans out to multiple Claude events (message_start, content_block_start, content_block_delta).
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
}

func TestProcessTranslateStreamEmitsDone(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	_, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
}

func TestProcessTranslateStreamFlushesStateOnClose(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 3)
	for _, raw := range []string{
		`{"id":"c1","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"tc1","type":"function","function":{"name":"Read","arguments":""}}]}}]}`,
		`{"id":"c2","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"city\":\"NYC\"}"}}]}}]}`,
		`{"id":"c3","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
	} {
		var chunk schemas.StreamChunk
		if err := json.Unmarshal([]byte(raw), &chunk); err != nil {
			t.Fatalf("unmarshal chunk: %v", err)
		}
		ch <- &chunk
	}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	_, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if !strings.Contains(out, "input_json_delta") && !strings.Contains(out, "content_block_stop") {
		t.Errorf("expected flushed tool args in output: %q", out)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
}

func TestProcessTranslateStreamReturnsErrorOnErrorChunk(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{Error: &schemas.ProviderError{Message: "boom", Type: "stream_error"}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	_, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err == nil {
		t.Fatal("expected error from error chunk")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want containing 'boom'", err)
	}

	out := w.buf.String()
	if strings.Contains(out, "[DONE]") {
		t.Errorf("error chunk should abort before [DONE]: %q", out)
	}
}

func TestProcessTranslateStreamSummaryAccumulates(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello "}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "world"}}}}
	ch <- &schemas.StreamChunk{ID: "c3", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Content != "hello world" {
		t.Errorf("summary.Content = %q, want %q", summary.Content, "hello world")
	}
}

func TestProcessTranslateStreamAttachesStateUsageOnFinish(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	// Claude-shaped usage so the format filter keeps it client-bound.
	state.Usage = map[string]any{"input_tokens": 10, "output_tokens": 5}
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Usage == nil {
		t.Fatal("expected usage in summary")
	}

	out := w.buf.String()
	// The finish chunk should have usage attached (with buffer applied).
	if !strings.Contains(out, `"input_tokens":2010`) {
		t.Errorf("output missing buffered input_tokens=2010 (10+2000); output: %q", out)
	}
}

func TestProcessTranslateStreamRecordsTTFT(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatClaude, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.TTFT.IsZero() {
		t.Error("TTFT should be set after first chunk")
	}
	if time.Since(summary.TTFT) < 0 {
		t.Error("TTFT should not be in the future")
	}
}

// --- Task 5: ProcessPassthroughStream tests ---

func TestProcessPassthroughFixesInvalidID(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "chat", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	summary, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Content != "hi" {
		t.Errorf("summary.Content = %q, want %q", summary.Content, "hi")
	}

	out := w.buf.String()
	if strings.Contains(out, `"id":"chat"`) {
		t.Errorf("invalid id not fixed: %q", out)
	}
	if !strings.Contains(out, "chatcmpl-") {
		t.Errorf("expected chatcmpl- prefix in output: %q", out)
	}
}

func TestProcessPassthroughInjectsRequiredFields(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "chatcmpl-12345678", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	_, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	// Extract the JSON payload from the data: line.
	lines := strings.Split(out, "\n")
	var payload map[string]any
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") && !strings.HasPrefix(line, "data: [DONE]") {
			data := strings.TrimPrefix(line, "data: ")
			if err := json.Unmarshal([]byte(data), &payload); err == nil {
				break
			}
		}
	}
	if payload == nil {
		t.Fatalf("no payload found in output: %q", out)
	}
	if payload["object"] != "chat.completion.chunk" {
		t.Errorf("object = %v, want chat.completion.chunk", payload["object"])
	}
	if payload["created"] == nil || payload["created"] == float64(0) {
		t.Errorf("created should be injected, got %v", payload["created"])
	}
}

func TestProcessPassthroughStripsAzureFields(t *testing.T) {
	// Processor-level: feed a chunk carrying both Azure fields.
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{
		ID:                  "chatcmpl-12345678",
		Choices:             []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}, ContentFilterResults: map[string]any{"hate": map[string]any{}}}},
		PromptFilterResults: []any{map[string]any{}},
	}
	close(ch)

	_, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	// Content retained; Azure keys stripped.
	if !strings.Contains(out, "hi") {
		t.Errorf("expected chunk content in SSE: %q", out)
	}
	if strings.Contains(out, "prompt_filter_results") {
		t.Errorf("prompt_filter_results should not appear in SSE: %q", out)
	}
	if strings.Contains(out, "content_filter_results") {
		t.Errorf("content_filter_results should not appear in SSE: %q", out)
	}

	// Secondary helper-level check: stripAzureFields on a map with Azure fields.
	payload := map[string]any{
		"id":      "chatcmpl-12345678",
		"object":  "chat.completion.chunk",
		"created": float64(123),
		"choices": []any{map[string]any{
			"index": float64(0),
			"delta": map[string]any{"content": "hi"},
			"content_filter_results": map[string]any{"hate": map[string]any{}},
		}},
		"prompt_filter_results": []any{map[string]any{}},
	}
	stripAzureFields(payload)

	if _, ok := payload["prompt_filter_results"]; ok {
		t.Errorf("prompt_filter_results should be stripped")
	}
	choices := payload["choices"].([]any)
	choice := choices[0].(map[string]any)
	if _, ok := choice["content_filter_results"]; ok {
		t.Errorf("content_filter_results should be stripped")
	}
}

func TestProcessPassthroughFiltersEmptyChunks(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}}}} // empty
	close(ch)

	_, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if strings.Count(out, "data: ") != 2 {
		t.Errorf("expected 2 data frames (1 chunk + DONE), got %d; output:\n%s", strings.Count(out, "data: "), out)
	}
}

func TestProcessPassthroughEmitsDone(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	_, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
}

func TestProcessPassthroughReturnsErrorOnErrorChunk(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{Error: &schemas.ProviderError{Message: "boom", Type: "stream_error"}}
	close(ch)

	_, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err == nil {
		t.Fatal("expected error from error chunk")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want containing 'boom'", err)
	}

	out := w.buf.String()
	if strings.Contains(out, "[DONE]") {
		t.Errorf("error chunk should abort before [DONE]: %q", out)
	}
}

// --- Task 2 (w5-f): StreamSummary.ContentLen + estimate-on-finish (PAR-TRANS-046) ---

func TestStreamAccumulatesContentLen(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 3)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello "}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "world"}}}}
	ch <- &schemas.StreamChunk{ID: "c3", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatOpenAI, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.ContentLen != 11 {
		t.Errorf("summary.ContentLen = %d, want 11 (len('hello ')+len('world'))", summary.ContentLen)
	}
}

func TestStreamAccumulatesUsage(t *testing.T) {
	// Chunk carries a real usage payload; finish chunk should attach the
	// buffered+filtered usage and the summary should expose the original.
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}, Usage: &schemas.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatOpenAI, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Usage == nil {
		t.Fatal("expected usage in summary")
	}
	if summary.Usage["prompt_tokens"] != 100 {
		t.Errorf("summary.Usage.prompt_tokens = %v, want 100", summary.Usage["prompt_tokens"])
	}

	out := w.buf.String()
	// The finish chunk should have a buffered usage attached.
	if !strings.Contains(out, `"prompt_tokens":2100`) {
		t.Errorf("finish chunk should carry buffered prompt_tokens=2100 (100+2000); output: %q", out)
	}
}

func TestStreamEstimatesOnFinish(t *testing.T) {
	// No usage in any chunk, content present → finish chunk + summary carry
	// estimated usage with `estimated` flag.
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "0123456789"}}}
	src := &EstimateSource{Body: body, Format: FormatOpenAI}
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatOpenAI, state, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Usage == nil {
		t.Fatal("expected estimated usage in summary")
	}
	if summary.Usage["estimated"] != true {
		t.Errorf("summary.Usage.estimated = %v, want true", summary.Usage["estimated"])
	}
	// contentLength 5 → 1 output token. Body is small → a few input tokens + buffer.
	if summary.Usage["completion_tokens"] != 1 {
		t.Errorf("summary.Usage.completion_tokens = %v, want 1", summary.Usage["completion_tokens"])
	}

	out := w.buf.String()
	if !strings.Contains(out, `"estimated":true`) {
		t.Errorf("client-bound finish chunk should carry estimated usage; output: %q", out)
	}
}

func TestPassthroughStreamEstimatesOnFinish(t *testing.T) {
	// Passthrough mode → no usage in any chunk, content present → finish
	// chunk + summary carry estimated usage.
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "0123456789"}}}
	src := &EstimateSource{Body: body, Format: FormatOpenAI}
	summary, err := ProcessPassthroughStream(context.Background(), w, ch, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Usage == nil {
		t.Fatal("expected estimated usage in summary")
	}
	if summary.Usage["estimated"] != true {
		t.Errorf("summary.Usage.estimated = %v, want true", summary.Usage["estimated"])
	}

	out := w.buf.String()
	if !strings.Contains(out, `"estimated":true`) {
		t.Errorf("client-bound finish chunk should carry estimated usage; output: %q", out)
	}
}

// TestPassthroughSummaryUsage verifies that passthrough mode preserves real
// provider usage from upstream chunks instead of falling back to estimation.
func TestPassthroughSummaryUsage(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{
		ID: "c2",
		Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}},
		Usage: &schemas.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
	}
	close(ch)

	summary, err := ProcessPassthroughStream(context.Background(), w, ch, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Usage == nil {
		t.Fatal("expected usage in summary")
	}
	if summary.Usage["estimated"] == true {
		t.Errorf("summary.Usage.estimated = %v, want false for real provider usage", summary.Usage["estimated"])
	}
	if summary.Usage["prompt_tokens"] != 100 {
		t.Errorf("summary.Usage.prompt_tokens = %v, want 100", summary.Usage["prompt_tokens"])
	}
	if summary.Usage["completion_tokens"] != 50 {
		t.Errorf("summary.Usage.completion_tokens = %v, want 50", summary.Usage["completion_tokens"])
	}

	out := w.buf.String()
	if !strings.Contains(out, `"prompt_tokens":2100`) {
		t.Errorf("finish chunk should carry buffered real usage; output: %q", out)
	}
	if strings.Contains(out, `"estimated":true`) {
		t.Errorf("finish chunk should not carry estimated flag for real usage; output: %q", out)
	}
}

func TestStreamNilEstimateSourceSkipsEstimation(t *testing.T) {
	// No EstimateSource → no estimation runs, summary.Usage is nil.
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 2)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hello"}}}}
	ch <- &schemas.StreamChunk{ID: "c2", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{}, FinishReason: strPtr("stop")}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	summary, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatOpenAI, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Usage != nil {
		t.Errorf("summary.Usage = %+v, want nil when no EstimateSource", summary.Usage)
	}
	if summary.ContentLen != 5 {
		t.Errorf("summary.ContentLen = %d, want 5", summary.ContentLen)
	}
}

// --- Row 050: translate-mode response.failed flush tests ---

func TestProcessTranslateStreamSynthesizesResponseFailed(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	state.ResponsesCompletedSent = true // suppress flush from emitting response.completed

	_, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatOpenAIResponses, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if !strings.Contains(out, "event: response.failed") {
		t.Errorf("missing synthesized response.failed: %q", out)
	}
	if !strings.Contains(out, `"code":"stream_disconnected"`) {
		t.Errorf("missing stream_disconnected code: %q", out)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
	// Synthesized failure must appear before [DONE].
	failedIdx := strings.Index(out, "event: response.failed")
	doneIdx := strings.Index(out, "data: [DONE]")
	if failedIdx == -1 || doneIdx == -1 || failedIdx > doneIdx {
		t.Errorf("response.failed should appear before [DONE]: %q", out)
	}
}

func TestProcessTranslateStreamNoSynthesisWhenCompleted(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	// Normal state: flush will emit response.completed, so terminal is seen.

	_, err := ProcessTranslateStream(context.Background(), w, ch, reg, FormatOpenAI, FormatOpenAIResponses, state, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.buf.String()
	if strings.Contains(out, "event: response.failed") {
		t.Errorf("should not synthesize response.failed when terminal seen: %q", out)
	}
	if !strings.Contains(out, "event: response.completed") {
		t.Errorf("expected response.completed from flush: %q", out)
	}
	if !strings.HasSuffix(out, "data: [DONE]\n\n") {
		t.Errorf("missing [DONE] terminator: %q", out)
	}
}
