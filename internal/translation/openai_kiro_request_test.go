package translation

import (
	"strings"
	"testing"
)

func TestKiroFlattenWhenNoTools(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"tool_calls": []any{
					map[string]any{
						"id":   "call_1",
						"type": "function",
						"function": map[string]any{
							"name":      "read",
							"arguments": `{"file":"x"}`,
						},
					},
				},
			},
			map[string]any{"role": "tool", "tool_call_id": "call_1", "content": `{"ok":true}`},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	content := uim["content"].(string)

	// The tool result should be flattened to text in the current user message.
	if !strings.Contains(content, "[Tool result:") {
		t.Errorf("currentMessage missing flattened tool result: %q", content)
	}

	// The flattened tool call should appear in history as assistant text.
	history := cs["history"].([]any)
	var foundToolCall bool
	for _, h := range history {
		hm := h.(map[string]any)
		if arm, ok := hm["assistantResponseMessage"].(map[string]any); ok {
			if strings.Contains(arm["content"].(string), "[Tool call:") {
				foundToolCall = true
			}
		}
	}
	if !foundToolCall {
		t.Errorf("history missing flattened tool call in assistant message")
	}

	// No structured tool references should remain anywhere.
	for _, h := range history {
		hm := h.(map[string]any)
		if uim, ok := hm["userInputMessage"].(map[string]any); ok {
			if ctx, ok := uim["userInputMessageContext"].(map[string]any); ok {
				if tr, ok := ctx["toolResults"].([]any); ok && len(tr) > 0 {
					t.Errorf("unexpected toolResults in history after flatten: %v", tr)
				}
			}
		}
	}
}

func TestKiroToolSpecInjectionAndMove(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hello"},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "read",
					"description": "read file",
					"parameters":  map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}},
				},
			},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	ctx := uim["userInputMessageContext"].(map[string]any)
	tools := ctx["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool in currentMessage, got %d", len(tools))
	}

	ts := tools[0].(map[string]any)["toolSpecification"].(map[string]any)
	if ts["name"] != "read" {
		t.Errorf("tool name = %v", ts["name"])
	}

	// History must NOT contain tools (cleaned up).
	history := cs["history"].([]any)
	for _, h := range history {
		hm := h.(map[string]any)
		if uim, ok := hm["userInputMessage"].(map[string]any); ok {
			if ctx, ok := uim["userInputMessageContext"].(map[string]any); ok {
				if _, ok := ctx["tools"]; ok {
					t.Errorf("history should not contain tools after cleanup")
				}
			}
		}
	}
}

func TestKiroOrphanToolResultSalvage(t *testing.T) {
	// Client sent tools, but an assistant message with the matching toolUse was
	// compacted away, leaving a dangling tool_result.
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "do it"},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type":        "tool_result",
						"tool_use_id": "orphan_1",
						"content":     "lost result",
					},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": "read",
				},
			},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	content := uim["content"].(string)

	// Orphaned result content should survive as text.
	if !strings.Contains(content, "lost result") {
		t.Errorf("orphaned tool result content not salvaged into text: %q", content)
	}
}

func TestKiroConsecutiveUserMerge(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "a"},
			map[string]any{"role": "system", "content": "b"},
			map[string]any{"role": "assistant", "content": "reply"},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)

	// The merged user message (user + system) becomes currentMessage (popped from history).
	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	content := uim["content"].(string)
	if !strings.Contains(content, "a") || !strings.Contains(content, "b") {
		t.Errorf("merged content missing parts: %q", content)
	}

	// History should have the assistant message only.
	history := cs["history"].([]any)
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
}

func TestKiroImagesDataURIAndHTTP(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": "data:image/png;base64,abc123",
						},
					},
					map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": "https://example.com/img.png",
						},
					},
					map[string]any{
						"type": "image",
						"source": map[string]any{
							"type":       "base64",
							"media_type": "image/jpeg",
							"data":       "def456",
						},
					},
				},
			},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	images := uim["images"].([]any)
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}

	// Data URI → base64 image
	img0 := images[0].(map[string]any)
	if img0["format"] != "png" {
		t.Errorf("img0 format = %v", img0["format"])
	}
	src0 := img0["source"].(map[string]any)
	if src0["bytes"] != "abc123" {
		t.Errorf("img0 source.bytes = %v", src0["bytes"])
	}

	// Claude base64 source → base64 image
	img1 := images[1].(map[string]any)
	if img1["format"] != "jpeg" {
		t.Errorf("img1 format = %v", img1["format"])
	}
	src1 := img1["source"].(map[string]any)
	if src1["bytes"] != "def456" {
		t.Errorf("img1 source.bytes = %v", src1["bytes"])
	}

	// HTTP URL should appear as text in content.
	content := uim["content"].(string)
	if !strings.Contains(content, "[Image: https://example.com/img.png]") {
		t.Errorf("HTTP URL not represented as text: %q", content)
	}
}

func TestKiroAssistantToolUses(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "do it"},
			map[string]any{
				"role": "assistant",
				"content": "",
				"tool_calls": []any{
					map[string]any{
						"id":   "call_1",
						"type": "function",
						"function": map[string]any{
							"name":      "read",
							"arguments": `malformed`,
						},
					},
				},
			},
		},
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{"name": "read"},
			},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	history := cs["history"].([]any)
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}

	arm := history[0].(map[string]any)["assistantResponseMessage"].(map[string]any)
	uses := arm["toolUses"].([]any)
	if len(uses) != 1 {
		t.Fatalf("expected 1 toolUse, got %d", len(uses))
	}

	use0 := uses[0].(map[string]any)
	if use0["name"] != "read" {
		t.Errorf("name = %v", use0["name"])
	}
	// Malformed arguments should fall back to {}
	input := use0["input"].(map[string]any)
	if len(input) != 0 {
		t.Errorf("malformed args should parse to empty object, got %v", input)
	}
	// ID should be present (fallback to generated uuid if missing).
	if use0["toolUseId"].(string) == "" {
		t.Error("toolUseId should not be empty")
	}
}

func TestKiroPayloadEnvelope(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
		"temperature": 0.5,
		"top_p":     0.9,
	}

	out, err := buildKiroPayload("claude-sonnet-4.5", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	if _, ok := out["_kiroUpstreamModel"]; ok {
		t.Error("payload must NOT contain _kiroUpstreamModel key")
	}

	cs := out["conversationState"].(map[string]any)
	if cs["chatTriggerType"] != "MANUAL" {
		t.Errorf("chatTriggerType = %v", cs["chatTriggerType"])
	}
	if cs["conversationId"].(string) == "" {
		t.Error("conversationId should not be empty")
	}

	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	if uim["origin"] != "AI_EDITOR" {
		t.Errorf("origin = %v", uim["origin"])
	}
	if uim["modelId"] != "claude-sonnet-4.5" {
		t.Errorf("modelId = %v", uim["modelId"])
	}

	ic := out["inferenceConfig"].(map[string]any)
	if ic["maxTokens"] != 32000 {
		t.Errorf("maxTokens = %v", ic["maxTokens"])
	}
	if ic["temperature"] != 0.5 {
		t.Errorf("temperature = %v", ic["temperature"])
	}
	if ic["topP"] != 0.9 {
		t.Errorf("topP = %v", ic["topP"])
	}
}

func TestKiroThinkingPrefixOrder(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
	}

	out, err := buildKiroPayload("claude-sonnet-4.5-thinking-agentic", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	current := cs["currentMessage"].(map[string]any)
	content := current["userInputMessage"].(map[string]any)["content"].(string)

	// Order: thinking tag → context line → agentic prompt.
	thinkingIdx := strings.Index(content, "<thinking_mode>enabled</thinking_mode>")
	contextIdx := strings.Index(content, "[Context: Current time is")
	agenticIdx := strings.Index(content, "# CRITICAL: CHUNKED WRITE PROTOCOL")

	if thinkingIdx == -1 {
		t.Fatal("missing thinking_mode tag")
	}
	if contextIdx == -1 {
		t.Fatal("missing context line")
	}
	if agenticIdx == -1 {
		t.Fatal("missing agentic prompt")
	}
	if !(thinkingIdx < contextIdx && contextIdx < agenticIdx) {
		t.Errorf("wrong prefix order: thinking=%d context=%d agentic=%d", thinkingIdx, contextIdx, agenticIdx)
	}
}

func TestKiroEmptyCurrentMessageSynthesized(t *testing.T) {
	// Only assistant messages — no user messages at all.
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "assistant", "content": "hello"},
		},
	}

	out, err := buildKiroPayload("m", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}

	cs := out["conversationState"].(map[string]any)
	current := cs["currentMessage"].(map[string]any)
	uim := current["userInputMessage"].(map[string]any)
	if uim["content"].(string) == "" {
		t.Error("expected non-empty content in synthesized currentMessage")
	}
	if uim["modelId"].(string) != "m" {
		t.Errorf("modelId = %v", uim["modelId"])
	}
}
