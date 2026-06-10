package translation

import (
	"fmt"
	"strconv"
	"time"
)

// openaiToResponsesResponse converts OpenAI Chat Completions streaming chunks
// into OpenAI Responses API events.
func openaiToResponsesResponse(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
	if chunk == nil {
		return flushResponsesEvents(state), nil
	}

	choicesRaw, ok := chunk["choices"].([]any)
	if !ok || len(choicesRaw) == 0 {
		return nil, nil
	}
	choice, ok := choicesRaw[0].(map[string]any)
	if !ok {
		return nil, nil
	}

	idx := 0
	if n, ok := choice["index"].(float64); ok {
		idx = int(n)
	} else if n, ok := choice["index"].(int); ok {
		idx = n
	}

	delta := map[string]any{}
	if d, ok := choice["delta"].(map[string]any); ok {
		delta = d
	}

	events := []map[string]any{}
	nextSeq := func() int {
		state.ResponsesSeq++
		return state.ResponsesSeq
	}
	emit := func(eventType string, data map[string]any) {
		data["sequence_number"] = nextSeq()
		events = append(events, map[string]any{
			"event": eventType,
			"data":  data,
		})
	}

	// Emit initial events
	if !state.ResponsesStarted {
		state.ResponsesStarted = true
		if id, ok := chunk["id"].(string); ok && id != "" {
			state.ResponsesID = fmt.Sprintf("resp_%s", id)
		}
		if state.ResponsesCreated == 0 {
			state.ResponsesCreated = time.Now().Unix()
		}

		emit("response.created", map[string]any{
			"type": "response.created",
			"response": map[string]any{
				"id":         state.ResponsesID,
				"object":     "response",
				"created_at": state.ResponsesCreated,
				"status":     "in_progress",
				"background": false,
				"error":      nil,
				"output":     []any{},
			},
		})

		emit("response.in_progress", map[string]any{
			"type": "response.in_progress",
			"response": map[string]any{
				"id":         state.ResponsesID,
				"object":     "response",
				"created_at": state.ResponsesCreated,
				"status":     "in_progress",
			},
		})
	}

	// Handle reasoning_content
	if reasoningContent, ok := delta["reasoning_content"].(string); ok && reasoningContent != "" {
		startResponsesReasoning(state, emit, idx)
		emitResponsesReasoningDelta(state, emit, reasoningContent)
	}

	// Handle text content
	if content, ok := delta["content"].(string); ok && content != "" {
		if thinkIdx := indexOfThinkOpen(content); thinkIdx >= 0 {
			state.ResponsesInThinking = true
			beforeThink := content[:thinkIdx]
			content = content[thinkIdx+len("<think>"):]
			if beforeThink != "" {
				emitResponsesTextContent(state, emit, idx, beforeThink)
			}
			startResponsesReasoning(state, emit, idx)
		}

		if thinkCloseIdx := indexOfThinkClose(content); thinkCloseIdx >= 0 {
			parts := splitAtThinkClose(content)
			thinkPart := parts[0]
			textPart := parts[1]
			if thinkPart != "" {
				emitResponsesReasoningDelta(state, emit, thinkPart)
			}
			closeResponsesReasoning(state, emit)
			state.ResponsesInThinking = false
			content = textPart
		}

		if state.ResponsesInThinking && content != "" {
			emitResponsesReasoningDelta(state, emit, content)
			return events, nil
		}

		if content != "" {
			emitResponsesTextContent(state, emit, idx, content)
		}
	}

	// Handle tool_calls
	if rawToolCalls, ok := delta["tool_calls"].([]any); ok && len(rawToolCalls) > 0 {
		closeResponsesMessage(state, emit, idx)
		for _, rawTC := range rawToolCalls {
			tc, ok := rawTC.(map[string]any)
			if !ok {
				continue
			}
			emitResponsesToolCall(state, emit, tc)
		}
	}

	// Handle finish_reason
	if finishReason, ok := choice["finish_reason"].(string); ok && finishReason != "" {
		for i := range state.ResponsesMsgItemAdded {
			closeResponsesMessage(state, emit, i)
		}
		closeResponsesReasoning(state, emit)
		for i := range state.ResponsesFuncCallIDs {
			closeResponsesToolCall(state, emit, i)
		}
		sendResponsesCompleted(state, emit)
	}

	return events, nil
}

func indexOfThinkOpen(s string) int {
	for i := 0; i <= len(s)-len("<think>"); i++ {
		if s[i:i+len("<think>")] == "<think>" {
			return i
		}
	}
	return -1
}

func indexOfThinkClose(s string) int {
	for i := 0; i <= len(s)-len("</think>"); i++ {
		if s[i:i+len("</think>")] == "</think>" {
			return i
		}
	}
	return -1
}

func splitAtThinkClose(s string) [2]string {
	idx := indexOfThinkClose(s)
	if idx < 0 {
		return [2]string{s, ""}
	}
	return [2]string{s[:idx], s[idx+len("</think>"):]}
}



func startResponsesReasoning(state *StreamState, emit func(string, map[string]any), idx int) {
	if state.ResponsesReasoningID != "" {
		return
	}
	state.ResponsesReasoningID = fmt.Sprintf("rs_%s_%d", state.ResponsesID, idx)
	state.ResponsesReasoningIndex = idx

	emit("response.output_item.added", map[string]any{
		"type":        "response.output_item.added",
		"output_index": idx,
		"item": map[string]any{
			"id":      state.ResponsesReasoningID,
			"type":    "reasoning",
			"summary": []any{},
		},
	})

	emit("response.reasoning_summary_part.added", map[string]any{
		"type":          "response.reasoning_summary_part.added",
		"item_id":       state.ResponsesReasoningID,
		"output_index":   idx,
		"summary_index": 0,
		"part": map[string]any{
			"type": "summary_text",
			"text": "",
		},
	})
}

func emitResponsesReasoningDelta(state *StreamState, emit func(string, map[string]any), text string) {
	if text == "" {
		return
	}
	state.ResponsesReasoningBuf += text
	emit("response.reasoning_summary_text.delta", map[string]any{
		"type":          "response.reasoning_summary_text.delta",
		"item_id":       state.ResponsesReasoningID,
		"output_index":   state.ResponsesReasoningIndex,
		"summary_index": 0,
		"delta":         text,
	})
}

func closeResponsesReasoning(state *StreamState, emit func(string, map[string]any)) {
	if state.ResponsesReasoningID == "" || state.ResponsesReasoningDone {
		return
	}
	state.ResponsesReasoningDone = true

	emit("response.reasoning_summary_text.done", map[string]any{
		"type":          "response.reasoning_summary_text.done",
		"item_id":       state.ResponsesReasoningID,
		"output_index":   state.ResponsesReasoningIndex,
		"summary_index": 0,
		"text":          state.ResponsesReasoningBuf,
	})

	emit("response.reasoning_summary_part.done", map[string]any{
		"type":          "response.reasoning_summary_part.done",
		"item_id":       state.ResponsesReasoningID,
		"output_index":   state.ResponsesReasoningIndex,
		"summary_index": 0,
		"part": map[string]any{
			"type": "summary_text",
			"text": state.ResponsesReasoningBuf,
		},
	})

	emit("response.output_item.done", map[string]any{
		"type":         "response.output_item.done",
		"output_index":  state.ResponsesReasoningIndex,
		"item": map[string]any{
			"id":      state.ResponsesReasoningID,
			"type":    "reasoning",
			"summary": []any{map[string]any{"type": "summary_text", "text": state.ResponsesReasoningBuf}},
		},
	})
}

func emitResponsesTextContent(state *StreamState, emit func(string, map[string]any), idx int, content string) {
	if !state.ResponsesMsgItemAdded[idx] {
		state.ResponsesMsgItemAdded[idx] = true
		msgID := fmt.Sprintf("msg_%s_%d", state.ResponsesID, idx)
		emit("response.output_item.added", map[string]any{
			"type":         "response.output_item.added",
			"output_index":  idx,
			"item": map[string]any{
				"id":      msgID,
				"type":    "message",
				"content": []any{},
				"role":    "assistant",
			},
		})
	}

	if !state.ResponsesContentAdded[idx] {
		state.ResponsesContentAdded[idx] = true
		msgID := fmt.Sprintf("msg_%s_%d", state.ResponsesID, idx)
		emit("response.content_part.added", map[string]any{
			"type":          "response.content_part.added",
			"item_id":       msgID,
			"output_index":   idx,
			"content_index": 0,
			"part": map[string]any{
				"type":        "output_text",
				"annotations": []any{},
				"logprobs":    []any{},
				"text":        "",
			},
		})
	}

	msgID := fmt.Sprintf("msg_%s_%d", state.ResponsesID, idx)
	emit("response.output_text.delta", map[string]any{
		"type":          "response.output_text.delta",
		"item_id":       msgID,
		"output_index":   idx,
		"content_index": 0,
		"delta":         content,
		"logprobs":      []any{},
	})
	state.ResponsesMsgTextBuf[idx] += content
}

func closeResponsesMessage(state *StreamState, emit func(string, map[string]any), idx int) {
	if !state.ResponsesMsgItemAdded[idx] || state.ResponsesItemDone[idx] {
		return
	}
	state.ResponsesItemDone[idx] = true
	fullText := state.ResponsesMsgTextBuf[idx]
	msgID := fmt.Sprintf("msg_%s_%d", state.ResponsesID, idx)

	emit("response.output_text.done", map[string]any{
		"type":          "response.output_text.done",
		"item_id":       msgID,
		"output_index":   idx,
		"content_index": 0,
		"text":          fullText,
		"logprobs":      []any{},
	})

	emit("response.content_part.done", map[string]any{
		"type":          "response.content_part.done",
		"item_id":       msgID,
		"output_index":   idx,
		"content_index": 0,
		"part": map[string]any{
			"type":        "output_text",
			"annotations": []any{},
			"logprobs":    []any{},
			"text":        fullText,
		},
	})

	emit("response.output_item.done", map[string]any{
		"type":         "response.output_item.done",
		"output_index":  idx,
		"item": map[string]any{
			"id":      msgID,
			"type":    "message",
			"content": []any{map[string]any{"type": "output_text", "annotations": []any{}, "logprobs": []any{}, "text": fullText}},
			"role":    "assistant",
		},
	})
}

func emitResponsesToolCall(state *StreamState, emit func(string, map[string]any), tc map[string]any) {
	tcIdx := 0
	if n, ok := tc["index"].(float64); ok {
		tcIdx = int(n)
	} else if n, ok := tc["index"].(int); ok {
		tcIdx = n
	}
	newCallID := ""
	if id, ok := tc["id"].(string); ok {
		newCallID = id
	}
	funcName := ""
	if fn, ok := tc["function"].(map[string]any); ok {
		if n, ok := fn["name"].(string); ok {
			funcName = n
		}
	}
	if funcName != "" {
		state.ResponsesFuncNames[tcIdx] = funcName
	}

	if state.ResponsesFuncCallIDs[tcIdx] == "" && newCallID != "" {
		state.ResponsesFuncCallIDs[tcIdx] = newCallID
		emit("response.output_item.added", map[string]any{
			"type":         "response.output_item.added",
			"output_index":  tcIdx,
			"item": map[string]any{
				"id":        fmt.Sprintf("fc_%s", newCallID),
				"type":      "function_call",
				"arguments": "",
				"call_id":   newCallID,
				"name":      state.ResponsesFuncNames[tcIdx],
			},
		})
	}

	if fn, ok := tc["function"].(map[string]any); ok {
		if args, ok := fn["arguments"].(string); ok && args != "" {
			refCallID := state.ResponsesFuncCallIDs[tcIdx]
			if refCallID == "" {
				refCallID = newCallID
			}
			if refCallID != "" {
				emit("response.function_call_arguments.delta", map[string]any{
					"type":         "response.function_call_arguments.delta",
					"item_id":      fmt.Sprintf("fc_%s", refCallID),
					"output_index":  tcIdx,
					"delta":        args,
				})
			}
			state.ResponsesFuncArgsBuf[tcIdx] += args
		}
	}
}

func closeResponsesToolCall(state *StreamState, emit func(string, map[string]any), idx int) {
	callID := state.ResponsesFuncCallIDs[idx]
	if callID == "" || state.ResponsesFuncItemDone[idx] {
		return
	}
	args := state.ResponsesFuncArgsBuf[idx]
	if args == "" {
		args = "{}"
	}

	emit("response.function_call_arguments.done", map[string]any{
		"type":         "response.function_call_arguments.done",
		"item_id":      fmt.Sprintf("fc_%s", callID),
		"output_index":  idx,
		"arguments":    args,
	})

	emit("response.output_item.done", map[string]any{
		"type":         "response.output_item.done",
		"output_index":  idx,
		"item": map[string]any{
			"id":        fmt.Sprintf("fc_%s", callID),
			"type":      "function_call",
			"arguments": args,
			"call_id":   callID,
			"name":      state.ResponsesFuncNames[idx],
		},
	})

	state.ResponsesFuncItemDone[idx] = true
	state.ResponsesFuncArgsDone[idx] = true
}

func sendResponsesCompleted(state *StreamState, emit func(string, map[string]any)) {
	if state.ResponsesCompletedSent {
		return
	}
	state.ResponsesCompletedSent = true
	emit("response.completed", map[string]any{
		"type": "response.completed",
		"response": map[string]any{
			"id":         state.ResponsesID,
			"object":     "response",
			"created_at": state.ResponsesCreated,
			"status":     "completed",
			"background": false,
			"error":      nil,
		},
	})
}

func flushResponsesEvents(state *StreamState) []map[string]any {
	if state.ResponsesCompletedSent {
		return nil
	}

	events := []map[string]any{}
	nextSeq := func() int {
		state.ResponsesSeq++
		return state.ResponsesSeq
	}
	emit := func(eventType string, data map[string]any) {
		data["sequence_number"] = nextSeq()
		events = append(events, map[string]any{
			"event": eventType,
			"data":  data,
		})
	}

	for i := range state.ResponsesMsgItemAdded {
		closeResponsesMessage(state, emit, i)
	}
	closeResponsesReasoning(state, emit)
	for i := range state.ResponsesFuncCallIDs {
		closeResponsesToolCall(state, emit, i)
	}
	sendResponsesCompleted(state, emit)

	return events
}

func parseIntAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}
