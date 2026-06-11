package translation

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateBillingHeaderFormat(t *testing.T) {
	payload := map[string]any{"model": "claude-3", "messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	h, err := generateBillingHeader(payload)
	if err != nil {
		t.Fatalf("generateBillingHeader: %v", err)
	}
	wantPrefix := "x-anthropic-billing-header:"
	if !strings.HasPrefix(h, wantPrefix) {
		t.Errorf("header = %q, want prefix %q", h, wantPrefix)
	}
	// cc_version=2.1.92.<3hex>
	if !strings.Contains(h, "cc_version=2.1.92.") {
		t.Errorf("header missing cc_version=2.1.92.")
	}
	if !strings.Contains(h, "cc_entrypoint=sdk-cli") {
		t.Errorf("header missing cc_entrypoint=sdk-cli")
	}
	// cch=<5hex>;
	re := regexp.MustCompile(`cch=[a-f0-9]{5};`)
	if !re.MatchString(h) {
		t.Errorf("header missing cch=5hex")
	}
}

func TestCloakToolsSuffixAndDecoys(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{"name": "myTool", "description": "d", "input_schema": map[string]any{"type": "object"}},
		},
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "tool_use", "id": "tu1", "name": "myTool", "input": map[string]any{}},
				},
			},
		},
		"tool_choice": map[string]any{"type": "tool", "name": "myTool"},
	}
	cloaked, toolMap, err := cloakClaudeTools(body)
	if err != nil {
		t.Fatalf("cloakClaudeTools: %v", err)
	}
	if toolMap == nil {
		t.Fatal("toolMap is nil")
	}
	if toolMap["myTool_ide"] != "myTool" {
		t.Errorf("toolMap[myTool_ide] = %q, want myTool", toolMap["myTool_ide"])
	}

	tools := cloaked["tools"].([]any)
	if len(tools) != 1+len(ccDecoyTools) {
		t.Fatalf("tools len = %d, want %d", len(tools), 1+len(ccDecoyTools))
	}
	first := tools[0].(map[string]any)
	if first["name"] != "myTool_ide" {
		t.Errorf("first tool name = %q, want myTool_ide", first["name"])
	}

	msgs := cloaked["messages"].([]any)
	content := msgs[0].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["name"] != "myTool_ide" {
		t.Errorf("tool_use name = %q, want myTool_ide", content[0].(map[string]any)["name"])
	}

	tc := cloaked["tool_choice"].(map[string]any)
	if tc["name"] != "myTool_ide" {
		t.Errorf("tool_choice.name = %q, want myTool_ide", tc["name"])
	}
}

func TestDecloakToolNames(t *testing.T) {
	toolMap := map[string]string{"myTool_ide": "myTool"}
	body := map[string]any{
		"content": []any{
			map[string]any{"type": "tool_use", "id": "tu1", "name": "myTool_ide", "input": map[string]any{}},
		},
	}
	result := decloakToolNames(body, toolMap)
	content := result["content"].([]any)
	if content[0].(map[string]any)["name"] != "myTool" {
		t.Errorf("decloaked name = %q, want myTool", content[0].(map[string]any)["name"])
	}
}

func TestApplyCloakingOAuthOnly(t *testing.T) {
	body := map[string]any{"messages": []any{map[string]any{"role": "user", "content": "hi"}}}
	result := applyCloaking(body, "sk-ant-api", "")
	if _, ok := result["system"]; ok {
		t.Error("non-oauth key should not inject system")
	}
}

func TestApplyCloakingBillingHeaderShapes(t *testing.T) {
	apiKey := "sk-ant-oat-test"

	t.Run("array system", func(t *testing.T) {
		body := map[string]any{
			"system": []any{map[string]any{"type": "text", "text": "sys1"}},
		}
		result := applyCloaking(body, apiKey, "")
		sys := result["system"].([]any)
		if len(sys) != 2 {
			t.Fatalf("system len = %d, want 2", len(sys))
		}
		first := sys[0].(map[string]any)
		if !strings.HasPrefix(first["text"].(string), "x-anthropic-billing-header:") {
			t.Errorf("first system block = %v", first)
		}
	})

	t.Run("string system", func(t *testing.T) {
		body := map[string]any{"system": "sys text"}
		result := applyCloaking(body, apiKey, "")
		sys := result["system"].([]any)
		if len(sys) != 2 {
			t.Fatalf("system len = %d, want 2", len(sys))
		}
		if sys[1].(map[string]any)["text"] != "sys text" {
			t.Errorf("second block = %v", sys[1])
		}
	})

	t.Run("absent system", func(t *testing.T) {
		body := map[string]any{"messages": []any{}}
		result := applyCloaking(body, apiKey, "")
		sys := result["system"].([]any)
		if len(sys) != 1 {
			t.Fatalf("system len = %d, want 1", len(sys))
		}
	})

	t.Run("already injected", func(t *testing.T) {
		body := map[string]any{
			"system": []any{map[string]any{"type": "text", "text": "x-anthropic-billing-header: old"}},
		}
		result := applyCloaking(body, apiKey, "")
		sys := result["system"].([]any)
		if len(sys) != 1 {
			t.Fatalf("system len = %d, want 1 (skip duplicate)", len(sys))
		}
	})
}

func TestApplyCloakingFakeUserIdPreserved(t *testing.T) {
	apiKey := "sk-ant-oat-test"
	body := map[string]any{
		"metadata": map[string]any{"user_id": "existing"},
	}
	result := applyCloaking(body, apiKey, "")
	meta := result["metadata"].(map[string]any)
	if meta["user_id"] != "existing" {
		t.Errorf("user_id = %q, want existing", meta["user_id"])
	}
}

func TestApplyCloakingFakeUserIdInjected(t *testing.T) {
	apiKey := "sk-ant-oat-test"
	body := map[string]any{}
	result := applyCloaking(body, apiKey, "sess-123")
	meta, ok := result["metadata"].(map[string]any)
	if !ok {
		t.Fatal("metadata missing")
	}
	uid, ok := meta["user_id"].(string)
	if !ok || uid == "" {
		t.Fatal("user_id missing or empty")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(uid), &parsed); err != nil {
		t.Fatalf("user_id not valid JSON: %v", err)
	}
	if parsed["session_id"] != "sess-123" {
		t.Errorf("session_id = %v, want sess-123", parsed["session_id"])
	}
}
