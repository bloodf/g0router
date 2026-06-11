package translation

import (
	"testing"
)

func TestPrepareSystemCacheControl(t *testing.T) {
	body := map[string]any{
		"system": []any{
			map[string]any{"type": "text", "text": "sys1", "cache_control": map[string]any{"type": "ephemeral"}},
			map[string]any{"type": "text", "text": "sys2", "cache_control": map[string]any{"type": "ephemeral"}},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	sys := result["system"].([]any)
	if len(sys) != 2 {
		t.Fatalf("system len = %d, want 2", len(sys))
	}
	first := sys[0].(map[string]any)
	if _, ok := first["cache_control"]; ok {
		t.Error("first system block should have cache_control removed")
	}
	last := sys[1].(map[string]any)
	cc, ok := last["cache_control"].(map[string]any)
	if !ok {
		t.Fatal("last system block missing cache_control")
	}
	if cc["type"] != "ephemeral" || cc["ttl"] != "1h" {
		t.Errorf("last cache_control = %v, want ephemeral+1h", cc)
	}
}

func TestPrepareFiltersEmptyKeepsFinalAssistant(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{"role": "assistant", "content": ""},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("messages len = %d, want 2", len(msgs))
	}
	last := msgs[1].(map[string]any)
	if last["role"] != "assistant" {
		t.Errorf("last role = %q, want assistant", last["role"])
	}
}

func TestPrepareToolUseOrdering(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "before"},
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read"},
					map[string]any{"type": "text", "text": "after"},
				},
			},
			map[string]any{"role": "user", "content": "next"},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	// Find the assistant message
	var assistant map[string]any
	for _, m := range msgs {
		msg := m.(map[string]any)
		if msg["role"] == "assistant" {
			assistant = msg
			break
		}
	}
	if assistant == nil {
		t.Fatal("assistant message not found")
	}
	content := assistant["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if content[0].(map[string]any)["type"] != "text" {
		t.Errorf("first block type = %q, want text", content[0].(map[string]any)["type"])
	}
	if content[1].(map[string]any)["type"] != "tool_use" {
		t.Errorf("second block type = %q, want tool_use", content[1].(map[string]any)["type"])
	}
}

func TestPrepareLastAssistantCacheControl(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "hello"},
				},
			},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	content := last["content"].([]any)
	block := content[0].(map[string]any)
	cc, ok := block["cache_control"].(map[string]any)
	if !ok {
		t.Fatal("last assistant block missing cache_control")
	}
	if cc["type"] != "ephemeral" {
		t.Errorf("cache_control type = %v, want ephemeral", cc["type"])
	}
}

func TestPrepareThinkingInjection(t *testing.T) {
	body := map[string]any{
		"thinking": map[string]any{"type": "enabled", "budget_tokens": 1024},
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read"},
				},
			},
			map[string]any{"role": "user", "content": "next"},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	var assistant map[string]any
	for _, m := range msgs {
		msg := m.(map[string]any)
		if msg["role"] == "assistant" {
			assistant = msg
			break
		}
	}
	if assistant == nil {
		t.Fatal("assistant message not found")
	}
	content := assistant["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if content[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("first block type = %q, want thinking", content[0].(map[string]any)["type"])
	}
	if content[1].(map[string]any)["type"] != "tool_use" {
		t.Errorf("second block type = %q, want tool_use", content[1].(map[string]any)["type"])
	}
}

func TestPrepareToolsBuiltinFilterNonClaude(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{"name": "myTool", "type": "function"},
			map[string]any{"name": "web_search", "type": "builtin"},
		},
	}
	result := PrepareClaudeRequest(body, "minimax", "", "")
	tools := result["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	if tools[0].(map[string]any)["name"] != "myTool" {
		t.Errorf("tool name = %q, want myTool", tools[0].(map[string]any)["name"])
	}
}

func TestPrepareMinimaxOutputConfigDropped(t *testing.T) {
	body := map[string]any{
		"output_config": map[string]any{"type": "json"},
	}
	result := PrepareClaudeRequest(body, "minimax", "", "")
	if _, ok := result["output_config"]; ok {
		t.Error("output_config should be dropped for minimax")
	}
}

func TestPrepareCloakingForOAuth(t *testing.T) {
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	result := PrepareClaudeRequest(body, "claude", "sk-ant-oat-test", "conn-123")
	if _, ok := result["system"]; !ok {
		t.Fatal("cloaking should inject system billing header")
	}
	meta, ok := result["metadata"].(map[string]any)
	if !ok {
		t.Fatal("cloaking should inject metadata")
	}
	if _, ok := meta["user_id"]; !ok {
		t.Error("metadata.user_id should be injected")
	}
}

func TestPrepareThinkingSignatureReplaced(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "hmm", "signature": "old"},
				},
			},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	content := last["content"].([]any)
	block := content[0].(map[string]any)
	if block["signature"] != defaultThinkingClaudeSignature {
		t.Error("thinking signature should be replaced with default")
	}
}

func TestPrepareDropEmptyToolsAndToolChoice(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{"name": "web_search", "type": "builtin"},
		},
		"tool_choice": map[string]any{"type": "auto"},
	}
	result := PrepareClaudeRequest(body, "minimax", "", "")
	if _, ok := result["tools"]; ok {
		t.Error("tools should be dropped when empty after filtering")
	}
	if _, ok := result["tool_choice"]; ok {
		t.Error("tool_choice should be dropped when tools are empty")
	}
}

func TestPrepareCacheControlRemovedFromMessages(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "hi", "cache_control": map[string]any{"type": "ephemeral"}},
				},
			},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	content := msgs[0].(map[string]any)["content"].([]any)
	block := content[0].(map[string]any)
	if _, ok := block["cache_control"]; ok {
		t.Error("cache_control should be removed from message content blocks")
	}
}

func TestPrepareThinkingNotInjectedWhenAlreadyPresent(t *testing.T) {
	body := map[string]any{
		"thinking": map[string]any{"type": "enabled"},
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": ".", "signature": "sig"},
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read"},
				},
			},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	content := last["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if content[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("first block = %q, want thinking", content[0].(map[string]any)["type"])
	}
}

func TestPrepareSystemStringNotArray(t *testing.T) {
	body := map[string]any{
		"system": "plain text",
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	// system string should remain unchanged (cache_control only applies to array system)
	if result["system"] != "plain text" {
		t.Errorf("system = %v, want plain text", result["system"])
	}
}

func TestPrepareLastToolCacheControl(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{"name": "t1", "type": "function"},
			map[string]any{"name": "t2", "type": "function"},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(tools))
	}
	first := tools[0].(map[string]any)
	if _, ok := first["cache_control"]; ok {
		t.Error("first tool should not have cache_control")
	}
	last := tools[1].(map[string]any)
	cc, ok := last["cache_control"].(map[string]any)
	if !ok {
		t.Fatal("last tool missing cache_control")
	}
	if cc["type"] != "ephemeral" || cc["ttl"] != "1h" {
		t.Errorf("last tool cache_control = %v", cc)
	}
}

func TestPrepareConsecutiveSameRoleMerged(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "a"}}},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "b"}}},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages len = %d, want 1 (merged)", len(msgs))
	}
	content := msgs[0].(map[string]any)["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
}

func TestPrepareCloakingNotAppliedForNonOAuth(t *testing.T) {
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	result := PrepareClaudeRequest(body, "claude", "sk-ant-api-test", "conn-123")
	if _, ok := result["system"]; ok {
		t.Error("system should not be injected for non-oauth key")
	}
}

func TestPrepareCloakingNotAppliedForNonClaude(t *testing.T) {
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	result := PrepareClaudeRequest(body, "openai", "sk-ant-oat-test", "conn-123")
	if _, ok := result["system"]; ok {
		t.Error("system should not be injected for non-claude provider")
	}
}

func TestPrepareThinkingInjectionOnlyForAnthropicCompatible(t *testing.T) {
	body := map[string]any{
		"thinking": map[string]any{"type": "enabled"},
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read"},
				},
			},
			map[string]any{"role": "user", "content": "next"},
		},
	}
	result := PrepareClaudeRequest(body, "anthropic-compatible-test", "", "")
	msgs := result["messages"].([]any)
	var assistant map[string]any
	for _, m := range msgs {
		msg := m.(map[string]any)
		if msg["role"] == "assistant" {
			assistant = msg
			break
		}
	}
	if assistant == nil {
		t.Fatal("assistant message not found")
	}
	content := assistant["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if content[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("first block type = %q, want thinking", content[0].(map[string]any)["type"])
	}
}

func TestPrepareMinimaxCnOutputConfigDropped(t *testing.T) {
	body := map[string]any{
		"output_config": map[string]any{"type": "json"},
	}
	result := PrepareClaudeRequest(body, "minimax-cn", "", "")
	if _, ok := result["output_config"]; ok {
		t.Error("output_config should be dropped for minimax-cn")
	}
}

func TestPrepareToolsNotFilteredForClaude(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{"name": "myTool", "type": "function"},
			map[string]any{"name": "web_search", "type": "builtin"},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(tools))
	}
}

func TestPrepareLastAssistantCacheControlSkipsThinking(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "thinking", "thinking": "hmm"},
					map[string]any{"type": "text", "text": "hello"},
				},
			},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	content := last["content"].([]any)
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	// cache_control should be on the text block, not the thinking block
	block := content[1].(map[string]any)
	if block["type"] != "text" {
		t.Fatalf("expected text block")
	}
	cc, ok := block["cache_control"].(map[string]any)
	if !ok {
		t.Fatal("text block missing cache_control")
	}
	if cc["type"] != "ephemeral" {
		t.Errorf("cache_control type = %v", cc["type"])
	}
}

func TestPrepareEmptyMessagesFiltered(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": ""},
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": ""}}},
			map[string]any{"role": "user", "content": "valid"},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages len = %d, want 1", len(msgs))
	}
	if msgs[0].(map[string]any)["content"] != "valid" {
		t.Errorf("content = %v, want valid", msgs[0].(map[string]any)["content"])
	}
}

func TestPrepareNonMapContentElementNoPanic(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "text", "text": "hello"},
					42, // non-map element should be skipped, not panic
				},
			},
		},
	}
	// Must not panic on non-map content element.
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages len = %d, want 1", len(msgs))
	}
}

func TestPrepareMergeToolResultsFirst(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "a"},
					map[string]any{"type": "tool_result", "tool_use_id": "tr1"},
				},
			},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "text", "text": "b"},
					map[string]any{"type": "tool_result", "tool_use_id": "tr2"},
				},
			},
		},
	}
	result := PrepareClaudeRequest(body, "claude", "", "")
	msgs := result["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages len = %d, want 1", len(msgs))
	}
	content := msgs[0].(map[string]any)["content"].([]any)
	if len(content) != 4 {
		t.Fatalf("content len = %d, want 4", len(content))
	}
	// tool_results should be first
	if content[0].(map[string]any)["type"] != "tool_result" {
		t.Errorf("first block = %q, want tool_result", content[0].(map[string]any)["type"])
	}
	if content[1].(map[string]any)["type"] != "tool_result" {
		t.Errorf("second block = %q, want tool_result", content[1].(map[string]any)["type"])
	}
}
