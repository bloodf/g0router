package translation

import (
	"testing"
)

func TestNewRegistryWiresOpenAIGeminiRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatOpenAI, FormatGemini) == nil {
		t.Error("NewRegistry must wire openai->gemini request translator")
	}
}

func TestOpenAIGeminiGenerationConfig(t *testing.T) {
	cases := []struct {
		name string
		body map[string]any
		want map[string]any
	}{
		{
			name: "all four params",
			body: map[string]any{
				"temperature": float64(0.7),
				"top_p":       float64(0.9),
				"top_k":       float64(40),
				"max_tokens":  float64(100),
			},
			want: map[string]any{
				"temperature":     float64(0.7),
				"topP":            float64(0.9),
				"topK":            float64(40),
				"maxOutputTokens": float64(100),
			},
		},
		{
			name: "absent param omission",
			body: map[string]any{},
			want: map[string]any{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			body["messages"] = []any{map[string]any{"role": "user", "content": "hi"}}
			out, err := openaiToGeminiRequest("gemini-pro", body, false)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			genConfig, ok := out["generationConfig"].(map[string]any)
			if !ok {
				t.Fatalf("generationConfig missing")
			}
			for k, v := range tc.want {
				if genConfig[k] != v {
					t.Errorf("generationConfig[%q] = %v, want %v", k, genConfig[k], v)
				}
			}
			for k := range genConfig {
				if _, expected := tc.want[k]; !expected {
					t.Errorf("unexpected generationConfig key %q = %v", k, genConfig[k])
				}
			}
		})
	}
}

func TestOpenAIGeminiSystemInstruction(t *testing.T) {
	t.Run("multi-message system", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{"role": "system", "content": "You are helpful."},
				map[string]any{"role": "user", "content": "hi"},
			},
		}
		out, err := openaiToGeminiRequest("gemini-pro", body, false)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		sysInstr, ok := out["systemInstruction"].(map[string]any)
		if !ok {
			t.Fatal("systemInstruction missing")
		}
		if sysInstr["role"] != "user" {
			t.Errorf("role = %v", sysInstr["role"])
		}
		parts := sysInstr["parts"].([]any)
		if len(parts) != 1 || parts[0].(map[string]any)["text"] != "You are helpful." {
			t.Errorf("parts = %v", parts)
		}
	})
	t.Run("lone system message becomes user", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{"role": "system", "content": "You are helpful."},
			},
		}
		out, err := openaiToGeminiRequest("gemini-pro", body, false)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if _, ok := out["systemInstruction"]; ok {
			t.Error("systemInstruction should not exist for lone system message")
		}
		contents := out["contents"].([]any)
		if len(contents) != 1 {
			t.Fatalf("len(contents) = %d", len(contents))
		}
		content := contents[0].(map[string]any)
		if content["role"] != "user" {
			t.Errorf("role = %v", content["role"])
		}
	})
}

func TestOpenAIGeminiReasoningContentThoughtParts(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role":              "assistant",
				"content":           "answer",
				"reasoning_content": "thinking...",
			},
		},
	}
	out, err := openaiToGeminiRequest("gemini-pro", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	contents := out["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("len(contents) = %d", len(contents))
	}
	parts := contents[0].(map[string]any)["parts"].([]any)
	if len(parts) != 3 {
		t.Fatalf("len(parts) = %d, want 3", len(parts))
	}
	if parts[0].(map[string]any)["thought"] != true || parts[0].(map[string]any)["text"] != "thinking..." {
		t.Errorf("part[0] = %v", parts[0])
	}
	if parts[1].(map[string]any)["thoughtSignature"] != defaultThinkingAGSignature || parts[1].(map[string]any)["text"] != "" {
		t.Errorf("part[1] = %v", parts[1])
	}
	if parts[2].(map[string]any)["text"] != "answer" {
		t.Errorf("part[2] = %v", parts[2])
	}
}

func TestOpenAIGeminiToolCallPairing(t *testing.T) {
	t.Run("basic pairing", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []any{
						map[string]any{
							"id":   "call-abc-123",
							"type": "function",
							"function": map[string]any{
								"name":      "get_weather",
								"arguments": `{"location":"NYC"}`,
							},
						},
					},
				},
				map[string]any{
					"role":         "tool",
					"tool_call_id": "call-abc-123",
					"content":      `{"temp":72}`,
				},
			},
		}
		out, err := openaiToGeminiRequest("gemini-pro", body, false)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		contents := out["contents"].([]any)
		if len(contents) != 2 {
			t.Fatalf("len(contents) = %d, want 2", len(contents))
		}
		modelContent := contents[0].(map[string]any)
		if modelContent["role"] != "model" {
			t.Errorf("role = %v", modelContent["role"])
		}
		modelParts := modelContent["parts"].([]any)
		if len(modelParts) != 1 {
			t.Fatalf("len(modelParts) = %d", len(modelParts))
		}
		fc := modelParts[0].(map[string]any)["functionCall"].(map[string]any)
		if fc["name"] != "get_weather" {
			t.Errorf("functionCall.name = %v", fc["name"])
		}
		if fc["id"] != "call-abc-123" {
			t.Errorf("functionCall.id = %v", fc["id"])
		}

		userContent := contents[1].(map[string]any)
		if userContent["role"] != "user" {
			t.Errorf("role = %v", userContent["role"])
		}
		userParts := userContent["parts"].([]any)
		if len(userParts) != 1 {
			t.Fatalf("len(userParts) = %d", len(userParts))
		}
		fr := userParts[0].(map[string]any)["functionResponse"].(map[string]any)
		if fr["name"] != "get_weather" {
			t.Errorf("functionResponse.name = %v", fr["name"])
		}
		response := fr["response"].(map[string]any)
		result := response["result"].(map[string]any)
		if result["temp"] != float64(72) {
			t.Errorf("result = %v", result)
		}
	})

	t.Run("id-split name fallback", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []any{
						map[string]any{
							"id":   "call-foo-bar-123-456",
							"type": "function",
							"function": map[string]any{
								"name":      "original_name",
								"arguments": `{}`,
							},
						},
					},
				},
				map[string]any{
					"role":         "tool",
					"tool_call_id": "call-foo-bar-123-456",
					"content":      "ok",
				},
			},
		}
		out, err := openaiToGeminiRequest("gemini-pro", body, false)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		contents := out["contents"].([]any)
		userContent := contents[1].(map[string]any)
		userParts := userContent["parts"].([]any)
		fr := userParts[0].(map[string]any)["functionResponse"].(map[string]any)
		if fr["name"] != "original_name" {
			t.Errorf("functionResponse.name = %v, want original_name", fr["name"])
		}
	})

	t.Run("non-JSON tool response wrapping", func(t *testing.T) {
		body := map[string]any{
			"messages": []any{
				map[string]any{
					"role":    "assistant",
					"content": "",
					"tool_calls": []any{
						map[string]any{
							"id":   "call-1",
							"type": "function",
							"function": map[string]any{
								"name":      "tool",
								"arguments": `{}`,
							},
						},
					},
				},
				map[string]any{
					"role":         "tool",
					"tool_call_id": "call-1",
					"content":      "plain text response",
				},
			},
		}
		out, err := openaiToGeminiRequest("gemini-pro", body, false)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		contents := out["contents"].([]any)
		userContent := contents[1].(map[string]any)
		userParts := userContent["parts"].([]any)
		fr := userParts[0].(map[string]any)["functionResponse"].(map[string]any)
		response := fr["response"].(map[string]any)
		result := response["result"].(map[string]any)
		if result["result"] != "plain text response" {
			t.Errorf("result = %v", result)
		}
	})
}

func TestOpenAIGeminiToolsCleaned(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{
				"type": "function",
				"function": map[string]any{
					"name":        "bad@name",
					"description": "does stuff",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"foo": map[string]any{"type": "string", "minLength": float64(1)},
						},
					},
				},
			},
			map[string]any{
				"name":        "claude_tool",
				"description": "claude tool",
				"input_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"bar": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
	out, err := openaiToGeminiRequest("gemini-pro", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tools, ok := out["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %v", out["tools"])
	}
	fdList := tools[0].(map[string]any)["functionDeclarations"].([]any)
	if len(fdList) != 2 {
		t.Fatalf("len(functionDeclarations) = %d", len(fdList))
	}
	fd0 := fdList[0].(map[string]any)
	if fd0["name"] != "bad_name" {
		t.Errorf("fd0.name = %v", fd0["name"])
	}
	params0 := fd0["parameters"].(map[string]any)
	if _, ok := params0["minLength"]; ok {
		t.Error("minLength should be removed")
	}
	fd1 := fdList[1].(map[string]any)
	if fd1["name"] != "claude_tool" {
		t.Errorf("fd1.name = %v", fd1["name"])
	}
}

func TestOpenAIGeminiNoToolConfig(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": "hi"},
		},
		"tool_choice": "auto",
	}
	out, err := openaiToGeminiRequest("gemini-pro", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := out["toolConfig"]; ok {
		t.Error("toolConfig should not be present")
	}

	body["tool_choice"] = map[string]any{"type": "function", "function": map[string]any{"name": "foo"}}
	out, err = openaiToGeminiRequest("gemini-pro", body, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if _, ok := out["toolConfig"]; ok {
		t.Error("toolConfig should not be present for object tool_choice")
	}
}
