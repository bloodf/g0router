package translation

import (
	"strings"
	"testing"
)

func TestOpenAIResponsesResponseLifecycle(t *testing.T) {
	state := NewStreamState()

	// First chunk should emit created + in_progress
	chunk := map[string]any{
		"id": "chunk1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"role": "assistant"}},
		},
	}
	events, err := openaiToResponsesResponse(chunk, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(events), events)
	}
	if events[0]["event"] != "response.created" {
		t.Errorf("event0 = %v", events[0]["event"])
	}
	if events[1]["event"] != "response.in_progress" {
		t.Errorf("event1 = %v", events[1]["event"])
	}

	// Verify sequence numbers are increasing
	seq0 := events[0]["data"].(map[string]any)["sequence_number"].(int)
	seq1 := events[1]["data"].(map[string]any)["sequence_number"].(int)
	if seq1 <= seq0 {
		t.Errorf("sequence numbers not increasing: %d, %d", seq0, seq1)
	}

	// Verify response id format
	resp := events[0]["data"].(map[string]any)["response"].(map[string]any)
	if !strings.HasPrefix(resp["id"].(string), "resp_") {
		t.Errorf("unexpected response id: %v", resp["id"])
	}

	// Second chunk should NOT emit created/in_progress again
	chunk2 := map[string]any{
		"id": "chunk1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"content": "hi"}},
		},
	}
	events2, err := openaiToResponsesResponse(chunk2, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, ev := range events2 {
		if ev["event"] == "response.created" || ev["event"] == "response.in_progress" {
			t.Errorf("created/in_progress should only fire once, got %v", ev["event"])
		}
	}
}

func TestOpenAIResponsesResponseTextDeltas(t *testing.T) {
	state := NewStreamState()

	// First chunk triggers created/in_progress + text delta
	chunk1 := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"content": "Hello"}},
		},
	}
	events1, err := openaiToResponsesResponse(chunk1, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have created, in_progress, output_item.added, content_part.added, output_text.delta
	if len(events1) != 5 {
		t.Fatalf("expected 5 events, got %d: %v", len(events1), events1)
	}
	if events1[2]["event"] != "response.output_item.added" {
		t.Errorf("expected output_item.added, got %v", events1[2]["event"])
	}
	if events1[3]["event"] != "response.content_part.added" {
		t.Errorf("expected content_part.added, got %v", events1[3]["event"])
	}
	if events1[4]["event"] != "response.output_text.delta" {
		t.Errorf("expected output_text.delta, got %v", events1[4]["event"])
	}

	// Second text delta should only emit delta (no re-adding)
	chunk2 := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"content": " world"}},
		},
	}
	events2, err := openaiToResponsesResponse(chunk2, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events2) != 1 {
		t.Fatalf("expected 1 event, got %d: %v", len(events2), events2)
	}
	if events2[0]["event"] != "response.output_text.delta" {
		t.Errorf("expected output_text.delta, got %v", events2[0]["event"])
	}

	// Finish reason should close message and emit completed
	chunk3 := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"},
		},
	}
	events3, err := openaiToResponsesResponse(chunk3, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sawDone, sawCompleted bool
	for _, ev := range events3 {
		switch ev["event"] {
		case "response.output_text.done":
			sawDone = true
		case "response.output_item.done":
			// message done
		case "response.completed":
			sawCompleted = true
		}
	}
	if !sawDone {
		t.Errorf("expected output_text.done")
	}
	if !sawCompleted {
		t.Errorf("expected response.completed")
	}
}

func TestOpenAIResponsesResponseReasoning(t *testing.T) {
	state := NewStreamState()

	chunk := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"reasoning_content": "thinking"}},
		},
	}
	events, err := openaiToResponsesResponse(chunk, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// created, in_progress, output_item.added (rs_), reasoning_summary_part.added, reasoning_summary_text.delta
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d: %v", len(events), events)
	}
	if events[2]["event"] != "response.output_item.added" {
		t.Errorf("expected output_item.added, got %v", events[2]["event"])
	}
	item := events[2]["data"].(map[string]any)["item"].(map[string]any)
	if !strings.HasPrefix(item["id"].(string), "rs_") {
		t.Errorf("expected rs_ prefix, got %v", item["id"])
	}
	if events[3]["event"] != "response.reasoning_summary_part.added" {
		t.Errorf("expected reasoning_summary_part.added, got %v", events[3]["event"])
	}
	if events[4]["event"] != "response.reasoning_summary_text.delta" {
		t.Errorf("expected reasoning_summary_text.delta, got %v", events[4]["event"])
	}

	// Finish should close reasoning
	finish := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"},
		},
	}
	events2, err := openaiToResponsesResponse(finish, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sawReasoningDone bool
	for _, ev := range events2 {
		if ev["event"] == "response.output_item.done" {
			item := ev["data"].(map[string]any)["item"].(map[string]any)
			if item["type"] == "reasoning" {
				sawReasoningDone = true
			}
		}
	}
	if !sawReasoningDone {
		t.Errorf("expected reasoning output_item.done")
	}
}

func TestOpenAIResponsesResponseThinkMarkers(t *testing.T) {
	state := NewStreamState()

	chunk := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"content": "<think>deep thought</think>answer"}},
		},
	}
	events, err := openaiToResponsesResponse(chunk, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have reasoning open, reasoning delta, reasoning close, then text events
	var sawReasoningAdded, sawReasoningDelta, sawReasoningDone bool
	var sawTextDelta bool
	for _, ev := range events {
		switch ev["event"] {
		case "response.output_item.added":
			item := ev["data"].(map[string]any)["item"].(map[string]any)
			if item["type"] == "reasoning" {
				sawReasoningAdded = true
			}
		case "response.reasoning_summary_text.delta":
			sawReasoningDelta = true
		case "response.output_item.done":
			item := ev["data"].(map[string]any)["item"].(map[string]any)
			if item["type"] == "reasoning" {
				sawReasoningDone = true
			}
		case "response.output_text.delta":
			sawTextDelta = true
		}
	}
	if !sawReasoningAdded {
		t.Errorf("expected reasoning output_item.added")
	}
	if !sawReasoningDelta {
		t.Errorf("expected reasoning_summary_text.delta")
	}
	if !sawReasoningDone {
		t.Errorf("expected reasoning output_item.done")
	}
	if !sawTextDelta {
		t.Errorf("expected output_text.delta after </think>")
	}
}

func TestOpenAIResponsesResponseToolCalls(t *testing.T) {
	state := NewStreamState()

	// First chunk: tool call starts
	chunk1 := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": 0,
							"id":    "call_1",
							"type":  "function",
							"function": map[string]any{
								"name":      "get_weather",
								"arguments": "",
							},
						},
					},
				},
			},
		},
	}
	events1, err := openaiToResponsesResponse(chunk1, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sawToolAdded bool
	for _, ev := range events1 {
		if ev["event"] == "response.output_item.added" {
			item := ev["data"].(map[string]any)["item"].(map[string]any)
			if item["type"] == "function_call" {
				sawToolAdded = true
				if !strings.HasPrefix(item["id"].(string), "fc_") {
					t.Errorf("expected fc_ prefix, got %v", item["id"])
				}
			}
		}
	}
	if !sawToolAdded {
		t.Errorf("expected function_call output_item.added")
	}

	// Second chunk: arguments delta
	chunk2 := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{
				"index": 0,
				"delta": map[string]any{
					"tool_calls": []any{
						map[string]any{
							"index": 0,
							"function": map[string]any{
								"arguments": `{"loc":`,
							},
						},
					},
				},
			},
		},
	}
	events2, err := openaiToResponsesResponse(chunk2, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events2) != 1 || events2[0]["event"] != "response.function_call_arguments.delta" {
		t.Fatalf("expected function_call_arguments.delta, got %v", events2)
	}

	// Finish
	chunk3 := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "tool_calls"},
		},
	}
	events3, err := openaiToResponsesResponse(chunk3, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sawToolDone, sawCompleted bool
	for _, ev := range events3 {
		if ev["event"] == "response.output_item.done" {
			item := ev["data"].(map[string]any)["item"].(map[string]any)
			if item["type"] == "function_call" {
				sawToolDone = true
			}
		}
		if ev["event"] == "response.completed" {
			sawCompleted = true
		}
	}
	if !sawToolDone {
		t.Errorf("expected function_call output_item.done")
	}
	if !sawCompleted {
		t.Errorf("expected response.completed")
	}
}

func TestOpenAIResponsesResponseFlush(t *testing.T) {
	state := NewStreamState()

	// Start with a text chunk
	chunk := map[string]any{
		"id": "c1",
		"choices": []any{
			map[string]any{"index": 0, "delta": map[string]any{"content": "hi"}},
		},
	}
	_, err := openaiToResponsesResponse(chunk, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil chunk should flush everything and emit completed
	events, err := openaiToResponsesResponse(nil, state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var sawTextDone, sawItemDone, sawCompleted bool
	for _, ev := range events {
		switch ev["event"] {
		case "response.output_text.done":
			sawTextDone = true
		case "response.output_item.done":
			item := ev["data"].(map[string]any)["item"].(map[string]any)
			if item["type"] == "message" {
				sawItemDone = true
			}
		case "response.completed":
			sawCompleted = true
		}
	}
	if !sawTextDone {
		t.Errorf("expected output_text.done on flush")
	}
	if !sawItemDone {
		t.Errorf("expected message output_item.done on flush")
	}
	if !sawCompleted {
		t.Errorf("expected response.completed on flush")
	}
}

func TestFormatSSEResponsesEventFraming(t *testing.T) {
	event := map[string]any{
		"event": "response.created",
		"data":  map[string]any{"type": "response.created", "sequence_number": 1},
	}
	got := string(FormatSSE(FormatOpenAIResponses, event))
	wantPrefix := "event: response.created\ndata: "
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("output = %q, want prefix %q", got, wantPrefix)
	}
	if !strings.HasSuffix(got, "\n\n") {
		t.Errorf("output missing terminating blank line: %q", got)
	}
}
