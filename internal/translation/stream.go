package translation

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
)

// StreamSummary holds aggregated data from a processed stream.
type StreamSummary struct {
	Content string
	Thinking string
	Usage   map[string]any
	TTFT    time.Time
}

// ProcessTranslateStream consumes a channel of stream chunks, translates each
// chunk through reg, and writes framed SSE to w. It returns a summary of the
// stream and a non-nil error if the stream aborted on an error chunk or write
// failure.
func ProcessTranslateStream(w io.Writer, ch <-chan *schemas.StreamChunk, reg *Registry, from, to Format, state *StreamState) (StreamSummary, error) {
	var summary StreamSummary

	for chunk := range ch {
		if summary.TTFT.IsZero() {
			summary.TTFT = time.Now()
		}

		if chunk.Error != nil {
			return summary, fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Marshal chunk to map[string]any for translation.
		b, err := json.Marshal(chunk)
		if err != nil {
			return summary, fmt.Errorf("marshal chunk: %w", err)
		}
		var openaiChunk map[string]any
		if err := json.Unmarshal(b, &openaiChunk); err != nil {
			return summary, fmt.Errorf("unmarshal chunk: %w", err)
		}

		// Accumulate content.
		accumulateContent(openaiChunk, &summary)

		// Translate: provider format -> client format.
		// Registry.TranslateResponse(to, from, ...) translates from `to` toward `from`.
		// ProcessTranslateStream passes (from=provider, to=client).
		results, err := reg.TranslateResponse(from, to, openaiChunk, state)
		if err != nil {
			return summary, fmt.Errorf("translate response: %w", err)
		}

		for _, item := range results {
			if !HasValuableContent(item, to) {
				continue
			}

			// Attach usage on finish chunk if state has usage.
			if state.Usage != nil && isFinishChunk(item) {
				item["usage"] = state.Usage
				summary.Usage = state.Usage
			}

			if _, werr := w.Write(FormatSSE(to, item)); werr != nil {
				return summary, fmt.Errorf("write translated chunk: %w", werr)
			}
		}
	}

	// Flush buffered translator state.
	flushed, err := reg.TranslateResponse(from, to, nil, state)
	if err != nil {
		return summary, fmt.Errorf("flush translator state: %w", err)
	}
	for _, item := range flushed {
		if item == nil {
			continue
		}
		if !HasValuableContent(item, to) {
			continue
		}
		if _, werr := w.Write(FormatSSE(to, item)); werr != nil {
			return summary, fmt.Errorf("write flushed chunk: %w", werr)
		}
	}

	if _, werr := w.Write([]byte("data: [DONE]\n\n")); werr != nil {
		return summary, fmt.Errorf("write done: %w", werr)
	}

	return summary, nil
}

// ProcessPassthroughStream consumes a channel of stream chunks, normalizes
// each chunk for OpenAI compatibility, and writes framed SSE to w.
func ProcessPassthroughStream(w io.Writer, ch <-chan *schemas.StreamChunk) (StreamSummary, error) {
	var summary StreamSummary

	for chunk := range ch {
		if summary.TTFT.IsZero() {
			summary.TTFT = time.Now()
		}

		b, err := json.Marshal(chunk)
		if err != nil {
			return summary, fmt.Errorf("marshal chunk: %w", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(b, &payload); err != nil {
			return summary, fmt.Errorf("unmarshal chunk: %w", err)
		}

		if chunk.Error != nil {
			return summary, fmt.Errorf("stream error: %w", chunk.Error)
		}

		// Normalize.
		FixInvalidID(payload)
		injectRequiredFields(payload)
		stripAzureFields(payload)

		if !HasValuableContent(payload, FormatOpenAI) {
			continue
		}

		accumulateContent(payload, &summary)

		data, err := json.Marshal(payload)
		if err != nil {
			return summary, fmt.Errorf("marshal normalized chunk: %w", err)
		}
		if _, werr := w.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); werr != nil {
			return summary, fmt.Errorf("write chunk: %w", werr)
		}
	}

	if _, werr := w.Write([]byte("data: [DONE]\n\n")); werr != nil {
		return summary, fmt.Errorf("write done: %w", werr)
	}

	return summary, nil
}

// accumulateContent adds delta.content from a chunk map into the summary.
func accumulateContent(chunk map[string]any, summary *StreamSummary) {
	choicesRaw, ok := chunk["choices"]
	if !ok {
		return
	}
	choices, ok := choicesRaw.([]any)
	if !ok || len(choices) == 0 {
		return
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return
	}
	deltaRaw, ok := choice["delta"]
	if !ok {
		return
	}
	delta, ok := deltaRaw.(map[string]any)
	if !ok {
		return
	}
	if content, ok := delta["content"].(string); ok {
		summary.Content += content
	}
}

// isFinishChunk reports whether a translated item is a terminal chunk.
func isFinishChunk(item map[string]any) bool {
	if typ, ok := item["type"].(string); ok && typ == "message_delta" {
		return true
	}
	choicesRaw, ok := item["choices"]
	if !ok {
		return false
	}
	choices, ok := choicesRaw.([]any)
	if !ok || len(choices) == 0 {
		return false
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return false
	}
	if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
		return true
	}
	return false
}

// injectRequiredFields ensures object and created are present on streaming
// chunks that carry choices.
func injectRequiredFields(payload map[string]any) {
	choicesRaw, ok := payload["choices"]
	if !ok {
		return
	}
	choices, ok := choicesRaw.([]any)
	if !ok || len(choices) == 0 {
		return
	}
	if obj, ok := payload["object"]; !ok || obj == "" {
		payload["object"] = "chat.completion.chunk"
	}
	if created, ok := payload["created"]; !ok || created == nil || created == float64(0) {
		payload["created"] = time.Now().Unix()
	}
}

// stripAzureFields removes Azure-specific filtering fields from a chunk.
func stripAzureFields(payload map[string]any) {
	delete(payload, "prompt_filter_results")
	choicesRaw, ok := payload["choices"]
	if !ok {
		return
	}
	choices, ok := choicesRaw.([]any)
	if !ok {
		return
	}
	for _, c := range choices {
		choice, ok := c.(map[string]any)
		if ok {
			delete(choice, "content_filter_results")
		}
	}
}
