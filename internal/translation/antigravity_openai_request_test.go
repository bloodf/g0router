package translation

import (
	"testing"
)

func TestNewRegistryWiresAntigravityOpenAIRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatAntigravity, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire antigravity->openai request translator")
	}
}

func TestAntigravityOpenAIUnwrapsEnvelope(t *testing.T) {
	body := map[string]any{
		"project":   "p",
		"model":     "gemini-pro",
		"userAgent": "antigravity",
		"request": map[string]any{
			"contents": []any{
				map[string]any{
					"role":  "user",
					"parts": []any{map[string]any{"text": "hi"}},
				},
			},
		},
	}
	out, err := antigravityToOpenAIRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) == 0 {
		t.Fatalf("messages missing or empty: %v", out["messages"])
	}
	first := msgs[0].(map[string]any)
	if first["role"] != "user" || first["content"] != "hi" {
		t.Errorf("first message = %v", first)
	}
}

func TestAntigravityOpenAIThinkingConfigToReasoningEffort(t *testing.T) {
	cases := []struct {
		budget int
		want   string
	}{
		{1024, "low"},
		{8192, "medium"},
		{32768, "high"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			body := map[string]any{
				"request": map[string]any{
					"generationConfig": map[string]any{
						"thinkingConfig": map[string]any{
							"thinkingBudget": tc.budget,
						},
					},
					"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
				},
			}
			out, err := antigravityToOpenAIRequest("gemini-pro", body, false, nil)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if out["reasoning_effort"] != tc.want {
				t.Errorf("reasoning_effort = %v, want %s", out["reasoning_effort"], tc.want)
			}
		})
	}
}

func TestAntigravityOpenAIConvertContentToolResults(t *testing.T) {
	body := map[string]any{
		"request": map[string]any{
			"contents": []any{
				map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{"functionCall": map[string]any{"id": "fc1", "name": "tool", "args": map[string]any{}}},
					},
				},
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{"functionResponse": map[string]any{"id": "fc1", "name": "tool", "response": map[string]any{"result": "ok"}}},
					},
				},
			},
		},
	}
	out, err := antigravityToOpenAIRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := out["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(msgs))
	}
	// First: assistant with tool_calls
	assistant := msgs[0].(map[string]any)
	if assistant["role"] != "assistant" {
		t.Errorf("msg[0] role = %v", assistant["role"])
	}
	toolCalls := assistant["tool_calls"].([]any)
	if len(toolCalls) != 1 {
		t.Fatalf("len(tool_calls) = %d", len(toolCalls))
	}
	// Second: tool message
	toolMsg := msgs[1].(map[string]any)
	if toolMsg["role"] != "tool" {
		t.Errorf("msg[1] role = %v", toolMsg["role"])
	}
	if toolMsg["tool_call_id"] != "fc1" {
		t.Errorf("tool_call_id = %v", toolMsg["tool_call_id"])
	}
}

func TestAntigravityOpenAINormalizeSchemaTypes(t *testing.T) {
	body := map[string]any{
		"request": map[string]any{
			"tools": []any{
				map[string]any{
					"functionDeclarations": []any{
						map[string]any{
							"name":        "test",
							"description": "desc",
							"parameters": map[string]any{
								"type":            "OBJECT",
								"enumDescriptions": []any{"a", "b"},
								"properties": map[string]any{
									"foo": map[string]any{"type": "STRING"},
								},
							},
						},
					},
				},
			},
			"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
		},
	}
	out, err := antigravityToOpenAIRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tools := out["tools"].([]any)
	fn := tools[0].(map[string]any)["function"].(map[string]any)
	params := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("type = %v, want object", params["type"])
	}
	if _, ok := params["enumDescriptions"]; ok {
		t.Error("enumDescriptions should be stripped")
	}
	foo := params["properties"].(map[string]any)["foo"].(map[string]any)
	if foo["type"] != "string" {
		t.Errorf("foo.type = %v, want string", foo["type"])
	}
}
