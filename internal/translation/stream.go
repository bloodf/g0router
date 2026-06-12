package translation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/bloodf/g0router/internal/schemas"
)

// StreamSummary holds aggregated data from a processed stream.
type StreamSummary struct {
	Content   string
	ContentLen int
	Thinking  string
	Usage     map[string]any
	TTFT      time.Time
}

// EstimateSource carries the request body and the client format used to
// estimate token usage when the provider stream omits usage (PAR-TRANS-046
// usage clause). A nil source disables estimation entirely.
type EstimateSource struct {
	Body   map[string]any
	Format Format
}

// ProcessTranslateStream consumes a channel of stream chunks, translates each
// chunk through reg, and writes framed SSE to w. It returns a summary of the
// stream and a non-nil error if the stream aborted on an error chunk or write
// failure. The loop also watches ctx.Done() so the caller can abort the stream
// without waiting for the next chunk.
//
// src optionally carries the request body and client format for usage
// estimation. When non-nil and the upstream emits no usage, the finish chunk
// gets a buffered+filtered estimated usage (PAR-TRANS-046 usage clause).
func ProcessTranslateStream(ctx context.Context, w io.Writer, ch <-chan *schemas.StreamChunk, reg *Registry, from, to Format, state *StreamState, src *EstimateSource) (StreamSummary, error) {
	var summary StreamSummary
	clientFormat := to
	if src != nil && src.Format != "" {
		clientFormat = src.Format
	}

loop:
	for {
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				break loop
			}

			if summary.TTFT.IsZero() {
				summary.TTFT = time.Now()
			}

			if chunk.Error != nil {
				return summary, fmt.Errorf("stream error: %w", chunk.Error)
			}

			b, err := json.Marshal(chunk)
			if err != nil {
				return summary, fmt.Errorf("marshal chunk: %w", err)
			}
			var openaiChunk map[string]any
			if err := json.Unmarshal(b, &openaiChunk); err != nil {
				return summary, fmt.Errorf("unmarshal chunk: %w", err)
			}

			accumulateContent(openaiChunk, &summary)

			// Extract usage from the upstream chunk and stash the original in
			// state.Usage for logging (the client-bound chunk may carry a
			// buffered+filtered version, but the original is what we want to
			// record for cost).
			if extracted := ExtractUsage(openaiChunk); extracted != nil {
				state.Usage = extracted
			}

			results, err := reg.TranslateResponse(from, to, openaiChunk, state)
			if err != nil {
				return summary, fmt.Errorf("translate response: %w", err)
			}

			for _, item := range results {
				if !HasValuableContent(item, to) {
					continue
				}

				if to == FormatOpenAIResponses && isResponsesTerminalEvent(item) {
					state.ResponsesTerminalSeen = true
				}

				if isFinishChunk(item) {
					applyFinishUsage(item, state, summary.ContentLen, src, clientFormat, &summary)
				}

				if _, werr := w.Write(FormatSSE(to, item)); werr != nil {
					return summary, fmt.Errorf("write translated chunk: %w", werr)
				}
			}
		}
	}

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
		if to == FormatOpenAIResponses && isResponsesTerminalEvent(item) {
			state.ResponsesTerminalSeen = true
		}
		if isFinishChunk(item) {
			applyFinishUsage(item, state, summary.ContentLen, src, clientFormat, &summary)
		}
		if _, werr := w.Write(FormatSSE(to, item)); werr != nil {
			return summary, fmt.Errorf("write flushed chunk: %w", werr)
		}
	}

	if to == FormatOpenAIResponses && !state.ResponsesTerminalSeen {
		if _, werr := w.Write(formatIncompleteResponsesStreamFailure()); werr != nil {
			return summary, fmt.Errorf("write failed chunk: %w", werr)
		}
	}

	if _, werr := w.Write([]byte("data: [DONE]\n\n")); werr != nil {
		return summary, fmt.Errorf("write done: %w", werr)
	}

	// Final fallback: if no usage was seen at all and we have content + a
	// body to estimate from, fill the summary so the caller can still record.
	if summary.Usage == nil && src != nil && summary.ContentLen > 0 {
		summary.Usage = EstimateUsage(src.Body, summary.ContentLen, clientFormat)
	}

	return summary, nil
}

// applyFinishUsage mirrors open-sse/utils/stream.js:152-159, 295-305:
//   - if the translated item carries valid usage, attach the buffered+filtered
//     version to the client-bound chunk and keep the original in summary.Usage
//   - otherwise, if state.Usage is already populated (e.g. injected by a
//     caller or by the translator), use it as the base and attach the
//     buffered+filtered version
//   - otherwise, if the chunk has no valid usage and we accumulated content,
//     build an estimated usage from the body + content length, attach the
//     filtered version, and keep the estimated in summary.Usage
func applyFinishUsage(item map[string]any, state *StreamState, contentLen int, src *EstimateSource, clientFormat Format, summary *StreamSummary) {
	hasUsage := HasValidUsage(usageFromItem(item))
	if hasUsage {
		// Prefer state.Usage (the original extracted payload) so we preserve
		// details objects the translator may have dropped.
		base := state.Usage
		if base == nil {
			base = usageFromItem(item)
		}
		buffered := AddBufferToUsage(base)
		item["usage"] = FilterUsageForFormat(buffered, clientFormat)
		summary.Usage = base
		return
	}
	// State-injected usage (tests, manual override). Buffer+filter and ship.
	if state.Usage != nil {
		buffered := AddBufferToUsage(state.Usage)
		item["usage"] = FilterUsageForFormat(buffered, clientFormat)
		summary.Usage = state.Usage
		return
	}
	if src == nil || contentLen <= 0 {
		return
	}
	estimated := EstimateUsage(src.Body, contentLen, clientFormat)
	item["usage"] = FilterUsageForFormat(estimated, clientFormat)
	state.Usage = estimated
	summary.Usage = estimated
}

// usageFromItem returns the item's usage field as a map, or nil.
func usageFromItem(item map[string]any) map[string]any {
	if item == nil {
		return nil
	}
	if u, ok := item["usage"].(map[string]any); ok {
		return u
	}
	return nil
}

// ProcessPassthroughStream consumes a channel of stream chunks, normalizes
// each chunk for OpenAI compatibility, and writes framed SSE to w. The loop
// also watches ctx.Done() so the caller can abort the stream without waiting
// for the next chunk.
//
// src optionally carries the request body and client format for usage
// estimation. When non-nil and the upstream emits no usage, the finish chunk
// gets a buffered+filtered estimated usage (PAR-TRANS-046 usage clause).
func ProcessPassthroughStream(ctx context.Context, w io.Writer, ch <-chan *schemas.StreamChunk, src *EstimateSource) (StreamSummary, error) {
	var summary StreamSummary
	clientFormat := FormatOpenAI
	if src != nil && src.Format != "" {
		clientFormat = src.Format
	}
	var lastUsage map[string]any

	for {
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				if _, werr := w.Write([]byte("data: [DONE]\n\n")); werr != nil {
					return summary, fmt.Errorf("write done: %w", werr)
				}
				// Final fallback: estimate on stream close (parity with
				// stream.js:327-329).
				if summary.Usage == nil && src != nil && summary.ContentLen > 0 {
					summary.Usage = EstimateUsage(src.Body, summary.ContentLen, clientFormat)
				}
				return summary, nil
			}

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

			FixInvalidID(payload)
			injectRequiredFields(payload)
			stripAzureFields(payload)

			if !HasValuableContent(payload, FormatOpenAI) {
				continue
			}

			accumulateContent(payload, &summary)

			// Extract usage from the upstream chunk.
			if extracted := ExtractUsage(payload); extracted != nil {
				lastUsage = extracted
			}

			// Finish-chunk handling (parity with stream.js:151-159).
			if isFinishReasonSet(payload) {
				if lastUsage != nil && HasValidUsage(lastUsage) {
					buffered := AddBufferToUsage(lastUsage)
					payload["usage"] = FilterUsageForFormat(buffered, clientFormat)
					summary.Usage = lastUsage
				} else if src != nil && summary.ContentLen > 0 {
					estimated := EstimateUsage(src.Body, summary.ContentLen, clientFormat)
					payload["usage"] = FilterUsageForFormat(estimated, clientFormat)
					lastUsage = estimated
					summary.Usage = estimated
				}
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return summary, fmt.Errorf("marshal normalized chunk: %w", err)
			}
			if _, werr := w.Write([]byte(fmt.Sprintf("data: %s\n\n", data))); werr != nil {
				return summary, fmt.Errorf("write chunk: %w", werr)
			}
		}
	}
}

// isFinishReasonSet reports whether a passthrough chunk carries a non-empty
// finish_reason on its first choice.
func isFinishReasonSet(payload map[string]any) bool {
	choicesRaw, ok := payload["choices"]
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
	fr, ok := choice["finish_reason"].(string)
	return ok && fr != ""
}

// accumulateContent adds delta.content from a chunk map into the summary.
// It also tracks ContentLen (total characters across content + reasoning
// content) which the estimate-on-finish path uses to size the output token
// estimate (PAR-TRANS-046 usage clause).
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
		summary.ContentLen += len(content)
	}
	if reasoning, ok := delta["reasoning_content"].(string); ok {
		summary.ContentLen += len(reasoning)
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
