package translation

import (
	"strings"
	"testing"
)

func TestResponsesOpenAIResponseTextDelta(t *testing.T) {
	state := NewStreamState()

	event := map[string]any{
		"type": "response.output_text.delta",
		"data": map[string]any{"delta": "hello"},
	}
	chunks, err := responsesToOpenAIResponse(event, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	chunk := chunks[0]
	choices := chunk["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["content"] != "hello" {
		t.Errorf("expected content='hello', got %v", delta["content"])
	}
	// id should be initialized
	if !strings.HasPrefix(chunk["id"].(string), "chatcmpl-") {
		t.Errorf("expected chatcmpl- prefix, got %v", chunk["id"])
	}
}

func TestResponsesOpenAIResponseToolCallLifecycle(t *testing.T) {
	state := NewStreamState()

	// Tool call added
	event1 := map[string]any{
		"type": "response.output_item.added",
		"data": map[string]any{
			"item": map[string]any{
				"type":    "function_call",
				"call_id": "call_1",
				"name":    "get_weather",
			},
		},
	}
	chunks1, err := responsesToOpenAIResponse(event1, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks1) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks1))
	}
	toolCalls := chunks1[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["tool_calls"].([]any)
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool_call, got %d", len(toolCalls))
	}
	tc := toolCalls[0].(map[string]any)
	if tc["id"] != "call_1" {
		t.Errorf("expected call_1, got %v", tc["id"])
	}

	// Arguments delta
	event2 := map[string]any{
		"type": "response.function_call_arguments.delta",
		"data": map[string]any{"delta": `{"loc":`},
	}
	chunks2, err := responsesToOpenAIResponse(event2, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks2) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks2))
	}
	fn := chunks2[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["tool_calls"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if fn["arguments"] != `{"loc":` {
		t.Errorf("unexpected arguments: %v", fn["arguments"])
	}

	// Tool call done (nil)
	event3 := map[string]any{
		"type": "response.output_item.done",
		"data": map[string]any{
			"item": map[string]any{"type": "function_call"},
		},
	}
	chunks3, err := responsesToOpenAIResponse(event3, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks3) != 0 {
		t.Fatalf("expected 0 chunks on done, got %d", len(chunks3))
	}

	// Completed → finish_reason tool_calls
	event4 := map[string]any{
		"type": "response.completed",
		"data": map[string]any{},
	}
	chunks4, err := responsesToOpenAIResponse(event4, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks4) != 1 {
		t.Fatalf("expected 1 final chunk, got %d", len(chunks4))
	}
	finishReason := chunks4[0]["choices"].([]any)[0].(map[string]any)["finish_reason"]
	if finishReason != "tool_calls" {
		t.Errorf("expected finish_reason=tool_calls, got %v", finishReason)
	}
}

func TestResponsesOpenAIResponseCompletedUsage(t *testing.T) {
	state := NewStreamState()

	// Text delta to initialize
	_, _ = responsesToOpenAIResponse(map[string]any{
		"type": "response.output_text.delta",
		"data": map[string]any{"delta": "ok"},
	}, state)

	event := map[string]any{
		"type": "response.completed",
		"data": map[string]any{
			"response": map[string]any{
				"usage": map[string]any{
					"input_tokens":  10,
					"output_tokens": 5,
					"input_tokens_details": map[string]any{
						"cached_tokens": 2,
					},
				},
			},
		},
	}
	chunks, err := responsesToOpenAIResponse(event, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	usage := chunks[0]["usage"].(map[string]any)
	if usage["prompt_tokens"] != 10 {
		t.Errorf("expected prompt_tokens=10, got %v", usage["prompt_tokens"])
	}
	if usage["completion_tokens"] != 5 {
		t.Errorf("expected completion_tokens=5, got %v", usage["completion_tokens"])
	}
	if usage["total_tokens"] != 15 {
		t.Errorf("expected total_tokens=15, got %v", usage["total_tokens"])
	}
	details := usage["prompt_tokens_details"].(map[string]any)
	if details["cached_tokens"] != 2 {
		t.Errorf("expected cached_tokens=2, got %v", details["cached_tokens"])
	}
}

func TestResponsesOpenAIResponseErrorEvent(t *testing.T) {
	state := NewStreamState()

	// First error
	event1 := map[string]any{
		"type": "error",
		"data": map[string]any{
			"error": map[string]any{"message": "model_not_found"},
		},
	}
	chunks1, err := responsesToOpenAIResponse(event1, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks1) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks1))
	}
	delta := chunks1[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	if delta["content"] != "[Error] model_not_found" {
		t.Errorf("unexpected content: %v", delta["content"])
	}

	// Second error (response.failed) should be deduplicated
	event2 := map[string]any{
		"type": "response.failed",
		"data": map[string]any{
			"response": map[string]any{
				"error": map[string]any{"message": "model_not_found"},
			},
		},
	}
	chunks2, err := responsesToOpenAIResponse(event2, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks2) != 0 {
		t.Errorf("expected 0 chunks after dedup, got %d", len(chunks2))
	}
}

func TestResponsesOpenAIResponseReasoningDelta(t *testing.T) {
	state := NewStreamState()

	event := map[string]any{
		"type": "response.reasoning_summary_text.delta",
		"data": map[string]any{"delta": "thinking..."},
	}
	chunks, err := responsesToOpenAIResponse(event, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	delta := chunks[0]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	if delta["reasoning_content"] != "thinking..." {
		t.Errorf("expected reasoning_content='thinking...', got %v", delta["reasoning_content"])
	}
}

func TestResponsesOpenAIResponseFlushFinish(t *testing.T) {
	state := NewStreamState()

	// Initialize with a text delta
	_, _ = responsesToOpenAIResponse(map[string]any{
		"type": "response.output_text.delta",
		"data": map[string]any{"delta": "ok"},
	}, state)

	// nil chunk should emit final finish chunk
	chunks, err := responsesToOpenAIResponse(nil, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	finishReason := chunks[0]["choices"].([]any)[0].(map[string]any)["finish_reason"]
	if finishReason != "stop" {
		t.Errorf("expected finish_reason=stop, got %v", finishReason)
	}

	// Second nil should not emit again
	chunks2, err := responsesToOpenAIResponse(nil, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks2) != 0 {
		t.Errorf("expected 0 chunks on second flush, got %d", len(chunks2))
	}
}
