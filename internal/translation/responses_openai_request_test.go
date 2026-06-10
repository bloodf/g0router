package translation

import (
	"reflect"
	"testing"
)

func TestResponsesOpenAIInstructions(t *testing.T) {
	body := map[string]any{
		"input":         []any{map[string]any{"type": "message", "role": "user", "content": "hi"}},
		"instructions":  "Be helpful",
		"model":         "gpt-4",
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %v", out["messages"])
	}
	sys := msgs[0].(map[string]any)
	if sys["role"] != "system" || sys["content"] != "Be helpful" {
		t.Errorf("expected system message, got %v", sys)
	}
}

func TestResponsesOpenAIStringInput(t *testing.T) {
	body := map[string]any{
		"input":  "hello",
		"model":  "gpt-4",
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %v", out["messages"])
	}
	msg := msgs[0].(map[string]any)
	if msg["role"] != "user" {
		t.Errorf("expected user role, got %v", msg["role"])
	}
}

func TestResponsesOpenAIEmptyInputPlaceholder(t *testing.T) {
	body := map[string]any{
		"input": []any{},
		"model": "gpt-4",
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected 1 placeholder message, got %v", out["messages"])
	}
	content := msgs[0].(map[string]any)["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if text != "..." {
		t.Errorf("expected '...' placeholder, got %q", text)
	}
}

func TestResponsesOpenAIItemGrouping(t *testing.T) {
	body := map[string]any{
		"input": []any{
			map[string]any{"type": "message", "role": "user", "content": "question"},
			map[string]any{"type": "function_call", "call_id": "c1", "name": "toolA", "arguments": "{}"},
			map[string]any{"type": "function_call_output", "call_id": "c1", "output": "result"},
		},
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d: %v", len(msgs), msgs)
	}
	// user → assistant with tool_calls → tool
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Errorf("msg0 role = %v", msgs[0].(map[string]any)["role"])
	}
	assistant := msgs[1].(map[string]any)
	if assistant["role"] != "assistant" {
		t.Errorf("msg1 role = %v", assistant["role"])
	}
	toolCalls, ok := assistant["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool_call, got %v", assistant["tool_calls"])
	}
	if msgs[2].(map[string]any)["role"] != "tool" {
		t.Errorf("msg2 role = %v", msgs[2].(map[string]any)["role"])
	}
}

func TestResponsesOpenAIRoleOnlyItems(t *testing.T) {
	body := map[string]any{
		"input": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %v", out["messages"])
	}
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Errorf("expected user role, got %v", msgs[0].(map[string]any)["role"])
	}
}

func TestResponsesOpenAIReasoningBuffering(t *testing.T) {
	body := map[string]any{
		"input": []any{
			map[string]any{"type": "reasoning", "summary": []any{map[string]any{"text": "thinking"}}},
			map[string]any{"type": "message", "role": "assistant", "content": "answer"},
		},
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := out["messages"].([]any)
	assistant := msgs[0].(map[string]any)
	if assistant["reasoning_content"] != "thinking" {
		t.Errorf("expected reasoning_content='thinking', got %v", assistant["reasoning_content"])
	}
}

func TestResponsesOpenAINamelessFunctionCallSkipped(t *testing.T) {
	body := map[string]any{
		"input": []any{
			map[string]any{"type": "function_call", "call_id": "c1", "name": "", "arguments": "{}"},
			map[string]any{"type": "function_call", "call_id": "c2", "arguments": "{}"},
			map[string]any{"type": "message", "role": "user", "content": "hi"},
		},
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := out["messages"].([]any)
	// Both nameless calls should be skipped; user message should be present
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (nameless skipped), got %d: %v", len(msgs), msgs)
	}
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Errorf("expected user message, got %v", msgs[0])
	}
}

func TestResponsesOpenAIHostedToolsDropped(t *testing.T) {
	body := map[string]any{
		"input": []any{map[string]any{"type": "message", "role": "user", "content": "hi"}},
		"tools": []any{
			map[string]any{"type": "function", "function": map[string]any{"name": "ok"}},
			map[string]any{"type": "request_user_input"},
			map[string]any{"type": "function", "name": "", "description": "empty"},
			map[string]any{"type": "function", "name": "flat", "parameters": map[string]any{"type": "object"}},
		},
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tools, ok := out["tools"].([]any)
	if !ok || len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %v", out["tools"])
	}
	// First is kept as-is (has function wrapper), last is converted
	t0 := tools[0].(map[string]any)
	if _, ok := t0["function"]; !ok {
		t.Errorf("expected wrapped tool, got %v", t0)
	}
	t1 := tools[1].(map[string]any)
	if t1["type"] != "function" {
		t.Errorf("expected converted tool, got %v", t1)
	}
	fn := t1["function"].(map[string]any)
	if fn["name"] != "flat" {
		t.Errorf("expected flat name, got %v", fn["name"])
	}
	// normalizeToolParameters adds properties
	if _, ok := fn["parameters"].(map[string]any)["properties"]; !ok {
		t.Errorf("expected properties added to parameters")
	}
}

func TestResponsesOpenAIFieldCleanup(t *testing.T) {
	body := map[string]any{
		"input":            []any{map[string]any{"type": "message", "role": "user", "content": "hi"}},
		"instructions":     "sys",
		"include":          []any{"usage"},
		"prompt_cache_key": "abc",
		"store":            true,
		"reasoning":        map[string]any{},
		"model":            "gpt-4",
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, key := range []string{"input", "instructions", "include", "prompt_cache_key", "store", "reasoning"} {
		if _, ok := out[key]; ok {
			t.Errorf("expected %s to be deleted", key)
		}
	}
}

func TestResponsesOpenAIImageInput(t *testing.T) {
	body := map[string]any{
		"input": []any{
			map[string]any{
				"type": "message", "role": "user",
				"content": []any{
					map[string]any{"type": "input_image", "image_url": "http://img", "detail": "high"},
					map[string]any{"type": "input_image", "file_id": "file_123"},
				},
			},
		},
	}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msgs := out["messages"].([]any)
	content := msgs[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(content))
	}
	img0 := content[0].(map[string]any)
	if img0["type"] != "image_url" {
		t.Errorf("expected image_url, got %v", img0["type"])
	}
	imgURL0 := img0["image_url"].(map[string]any)
	if imgURL0["url"] != "http://img" || imgURL0["detail"] != "high" {
		t.Errorf("unexpected image_url block: %v", imgURL0)
	}
	img1 := content[1].(map[string]any)
	imgURL1 := img1["image_url"].(map[string]any)
	if imgURL1["url"] != "file_123" || imgURL1["detail"] != "auto" {
		t.Errorf("unexpected default detail: %v", imgURL1)
	}
}

func TestResponsesOpenAINoInputReturnsBody(t *testing.T) {
	body := map[string]any{"model": "gpt-4", "messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(out, body) {
		t.Errorf("expected body unchanged when no input key")
	}
}

func TestResponsesOpenAINilNormalizeReturnsBody(t *testing.T) {
	body := map[string]any{"input": 123, "model": "gpt-4"}
	out, err := responsesToOpenAIRequest("gpt-4", body, false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(out, body) {
		t.Errorf("expected body unchanged when normalize returns nil")
	}
}
