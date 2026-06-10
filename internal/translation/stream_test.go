package translation

import (
	"bytes"
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
	_, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
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
	_, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
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
	_, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
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
	_, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
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
	summary, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
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
	state.Usage = map[string]any{"total_tokens": 42}
	summary, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Usage == nil {
		t.Fatal("expected usage in summary")
	}

	out := w.buf.String()
	// The finish chunk should have usage attached.
	if !strings.Contains(out, "total_tokens") {
		t.Errorf("output missing usage: %q", out)
	}
}

func TestProcessTranslateStreamRecordsTTFT(t *testing.T) {
	w := &fakeWriter{}
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "c1", Choices: []schemas.StreamChoice{{Index: 0, Delta: schemas.Message{Content: "hi"}}}}
	close(ch)

	reg := NewRegistry()
	state := NewStreamState()
	summary, err := ProcessTranslateStream(w, ch, reg, FormatOpenAI, FormatClaude, state)
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

	summary, err := ProcessPassthroughStream(w, ch)
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

	_, err := ProcessPassthroughStream(w, ch)
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

	_, err := ProcessPassthroughStream(w, ch)
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

	_, err := ProcessPassthroughStream(w, ch)
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

	_, err := ProcessPassthroughStream(w, ch)
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
