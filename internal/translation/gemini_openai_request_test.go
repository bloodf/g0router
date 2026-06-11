package translation

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGeminiOpenAIGenerationConfig(t *testing.T) {
	t.Run("maxOutputTokens temperature topP", func(t *testing.T) {
		body := map[string]any{
			"generationConfig": map[string]any{
				"maxOutputTokens": 100,
				"temperature":     0.7,
				"topP":            0.9,
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if result["temperature"] != 0.7 {
			t.Errorf("temperature = %v, want 0.7", result["temperature"])
		}
		if result["top_p"] != 0.9 {
			t.Errorf("top_p = %v, want 0.9", result["top_p"])
		}
		// max_tokens goes through AdjustMaxTokens.
		// With no tools and maxOutputTokens=100, AdjustMaxTokens returns 100.
		if result["max_tokens"] != 100 {
			t.Errorf("max_tokens = %v, want 100", result["max_tokens"])
		}
	})

	t.Run("AdjustMaxTokens with tools bumps min", func(t *testing.T) {
		body := map[string]any{
			"generationConfig": map[string]any{
				"maxOutputTokens": 100,
			},
			"tools": []any{
				map[string]any{"functionDeclarations": []any{map[string]any{"name": "x"}}},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		// AdjustMaxTokens sees tools and value < defaultMinTokens (32000)
		if result["max_tokens"] != defaultMinTokens {
			t.Errorf("max_tokens = %v, want %d", result["max_tokens"], defaultMinTokens)
		}
	})

	t.Run("no generationConfig", func(t *testing.T) {
		body := map[string]any{}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if _, ok := result["max_tokens"]; ok {
			t.Error("expected no max_tokens when generationConfig absent")
		}
		if _, ok := result["temperature"]; ok {
			t.Error("expected no temperature when generationConfig absent")
		}
		if _, ok := result["top_p"]; ok {
			t.Error("expected no top_p when generationConfig absent")
		}
	})
}

func TestGeminiOpenAISystemInstruction(t *testing.T) {
	t.Run("string passthrough", func(t *testing.T) {
		body := map[string]any{
			"systemInstruction": "You are helpful.",
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs, ok := result["messages"].([]any)
		if !ok || len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %v", result["messages"])
		}
		msg := msgs[0].(map[string]any)
		if msg["role"] != "system" {
			t.Errorf("role = %v, want system", msg["role"])
		}
		if msg["content"] != "You are helpful." {
			t.Errorf("content = %v", msg["content"])
		}
	})

	t.Run("parts text join", func(t *testing.T) {
		body := map[string]any{
			"systemInstruction": map[string]any{
				"parts": []any{
					map[string]any{"text": "Hello "},
					map[string]any{"text": "world"},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		msg := msgs[0].(map[string]any)
		if msg["content"] != "Hello world" {
			t.Errorf("content = %v, want 'Hello world'", msg["content"])
		}
	})

	t.Run("empty systemInstruction omitted", func(t *testing.T) {
		body := map[string]any{
			"systemInstruction": map[string]any{
				"parts": []any{},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		if len(msgs) != 0 {
			t.Errorf("expected 0 messages, got %d", len(msgs))
		}
	})
}

func TestGeminiOpenAIContentTextAndImage(t *testing.T) {
	t.Run("text only", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{"text": "hi"},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		msg := msgs[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("role = %v, want user", msg["role"])
		}
		if msg["content"] != "hi" {
			t.Errorf("content = %v, want 'hi'", msg["content"])
		}
	})

	t.Run("inlineData image", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"inlineData": map[string]any{
								"mimeType": "image/png",
								"data":     "abc123",
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		content := msg["content"].([]any)
		if len(content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(content))
		}
		part := content[0].(map[string]any)
		if part["type"] != "image_url" {
			t.Errorf("type = %v, want image_url", part["type"])
		}
		url := part["image_url"].(map[string]any)["url"].(string)
		if url != "data:image/png;base64,abc123" {
			t.Errorf("url = %v", url)
		}
	})

	t.Run("text and image mixed", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{"text": "look at this"},
						map[string]any{
							"inlineData": map[string]any{
								"mimeType": "image/jpeg",
								"data":     "xyz",
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		content := msg["content"].([]any)
		if len(content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(content))
		}
		if content[0].(map[string]any)["type"] != "text" {
			t.Errorf("content[0].type = %v", content[0].(map[string]any)["type"])
		}
		if content[1].(map[string]any)["type"] != "image_url" {
			t.Errorf("content[1].type = %v", content[1].(map[string]any)["type"])
		}
	})

	t.Run("non-user role becomes assistant", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{"text": "ok"},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		if msg["role"] != "assistant" {
			t.Errorf("role = %v, want assistant", msg["role"])
		}
	})
}

func TestGeminiOpenAIFunctionCall(t *testing.T) {
	body := map[string]any{
		"contents": []any{
			map[string]any{
				"role": "model",
				"parts": []any{
					map[string]any{
						"functionCall": map[string]any{
							"name": "get_weather",
							"args": map[string]any{"location": "NYC"},
						},
					},
					map[string]any{
						"functionCall": map[string]any{
							"name": "get_time",
							"args": map[string]any{"timezone": "EST"},
						},
					},
				},
			},
		},
	}
	result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	msgs := result["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	msg := msgs[0].(map[string]any)
	if msg["role"] != "assistant" {
		t.Errorf("role = %v, want assistant", msg["role"])
	}
	toolCalls, ok := msg["tool_calls"].([]any)
	if !ok || len(toolCalls) != 2 {
		t.Fatalf("expected 2 tool_calls, got %v", msg["tool_calls"])
	}

	ids := make(map[string]bool)
	for i, tcRaw := range toolCalls {
		tc := tcRaw.(map[string]any)
		id, _ := tc["id"].(string)
		if !strings.HasPrefix(id, "call_") {
			t.Errorf("tool_calls[%d].id = %q, expected prefix 'call_'", i, id)
		}
		if ids[id] {
			t.Errorf("tool_calls[%d].id = %q is not unique within the message", i, id)
		}
		ids[id] = true
		if tc["type"] != "function" {
			t.Errorf("tool_calls[%d].type = %v, want function", i, tc["type"])
		}
	}

	tc0 := toolCalls[0].(map[string]any)
	fn0 := tc0["function"].(map[string]any)
	if fn0["name"] != "get_weather" {
		t.Errorf("tool_calls[0].name = %v", fn0["name"])
	}
	if fn0["arguments"] != `{"location":"NYC"}` {
		t.Errorf("tool_calls[0].arguments = %v", fn0["arguments"])
	}

	tc1 := toolCalls[1].(map[string]any)
	fn1 := tc1["function"].(map[string]any)
	if fn1["name"] != "get_time" {
		t.Errorf("tool_calls[1].name = %v", fn1["name"])
	}
	if fn1["arguments"] != `{"timezone":"EST"}` {
		t.Errorf("tool_calls[1].arguments = %v", fn1["arguments"])
	}
}

func TestGeminiOpenAIFunctionResponse(t *testing.T) {
	t.Run("id fallback", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_1",
								"name":     "get_weather",
								"response": map[string]any{"result": map[string]any{"temp": 72}},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		if len(msgs) != 1 {
			t.Fatalf("expected 1 message, got %d", len(msgs))
		}
		msg := msgs[0].(map[string]any)
		if msg["role"] != "tool" {
			t.Errorf("role = %v, want tool", msg["role"])
		}
		if msg["tool_call_id"] != "call_1" {
			t.Errorf("tool_call_id = %v, want call_1", msg["tool_call_id"])
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["temp"] != float64(72) {
			t.Errorf("content temp = %v", parsed["temp"])
		}
	})

	t.Run("name fallback when id absent", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"name":     "get_weather",
								"response": map[string]any{"humidity": 50},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		if msg["tool_call_id"] != "get_weather" {
			t.Errorf("tool_call_id = %v, want get_weather", msg["tool_call_id"])
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["humidity"] != float64(50) {
			t.Errorf("content humidity = %v", parsed["humidity"])
		}
	})

	t.Run("response without result field", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_2",
								"response": map[string]any{"raw": "data"},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["raw"] != "data" {
			t.Errorf("content raw = %v", parsed["raw"])
		}
	})

	t.Run("empty response defaults to {}", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id": "call_3",
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		if msg["content"] != "{}" {
			t.Errorf("content = %v, want '{}'", msg["content"])
		}
	})

	t.Run("result 0 falls back to response object", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_4",
								"response": map[string]any{"result": 0, "extra": "data"},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["result"] != float64(0) {
			t.Errorf("content result = %v, want 0", parsed["result"])
		}
		if parsed["extra"] != "data" {
			t.Errorf("content extra = %v, want data", parsed["extra"])
		}
	})

	t.Run("result false falls back to response object", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_5",
								"response": map[string]any{"result": false, "extra": "data"},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["result"] != false {
			t.Errorf("content result = %v, want false", parsed["result"])
		}
		if parsed["extra"] != "data" {
			t.Errorf("content extra = %v, want data", parsed["extra"])
		}
	})

	t.Run("result empty string falls back to response object", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_6",
								"response": map[string]any{"result": "", "extra": "data"},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["result"] != "" {
			t.Errorf("content result = %v, want empty string", parsed["result"])
		}
		if parsed["extra"] != "data" {
			t.Errorf("content extra = %v, want data", parsed["extra"])
		}
	})

	t.Run("result object passes through", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_7",
								"response": map[string]any{"result": map[string]any{"x": 1}},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["x"] != float64(1) {
			t.Errorf("content x = %v, want 1", parsed["x"])
		}
	})

	t.Run("absent result falls back to response object", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{
							"functionResponse": map[string]any{
								"id":       "call_8",
								"response": map[string]any{"raw": "value"},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msgs := result["messages"].([]any)
		msg := msgs[0].(map[string]any)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(msg["content"].(string)), &parsed); err != nil {
			t.Fatalf("content not valid JSON: %v", err)
		}
		if parsed["raw"] != "value" {
			t.Errorf("content raw = %v, want value", parsed["raw"])
		}
	})
}

func TestGeminiOpenAITools(t *testing.T) {
	body := map[string]any{
		"tools": []any{
			map[string]any{
				"functionDeclarations": []any{
					map[string]any{
						"name":        "Read",
						"description": "read file",
						"parameters": map[string]any{
							"type":       "object",
							"properties": map[string]any{},
						},
					},
				},
			},
		},
	}
	result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %v", result["tools"])
	}
	tool := tools[0].(map[string]any)
	if tool["type"] != "function" {
		t.Errorf("type = %v, want function", tool["type"])
	}
	fn := tool["function"].(map[string]any)
	if fn["name"] != "Read" {
		t.Errorf("name = %v", fn["name"])
	}
	if fn["description"] != "read file" {
		t.Errorf("description = %v", fn["description"])
	}
	params := fn["parameters"].(map[string]any)
	if params["type"] != "object" {
		t.Errorf("parameters.type = %v", params["type"])
	}

	// Default parameters when absent.
	body2 := map[string]any{
		"tools": []any{
			map[string]any{
				"functionDeclarations": []any{
					map[string]any{
						"name": "Write",
					},
				},
			},
		},
	}
	result2, err := geminiToOpenAIRequest("gemini-pro", body2, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	tools2 := result2["tools"].([]any)
	fn2 := tools2[0].(map[string]any)["function"].(map[string]any)
	params2 := fn2["parameters"].(map[string]any)
	if params2["type"] != "object" {
		t.Errorf("default parameters.type = %v", params2["type"])
	}
	props, ok := params2["properties"].(map[string]any)
	if !ok || len(props) != 0 {
		t.Errorf("default properties = %v", params2["properties"])
	}
}

func TestGeminiOpenAISingleTextCollapse(t *testing.T) {
	t.Run("user single text → string", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role":  "user",
					"parts": []any{map[string]any{"text": "hello"}},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msg := result["messages"].([]any)[0].(map[string]any)
		if msg["content"] != "hello" {
			t.Errorf("content = %v, want 'hello'", msg["content"])
		}
	})

	t.Run("user multiple text → array", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "user",
					"parts": []any{
						map[string]any{"text": "hello"},
						map[string]any{"text": "world"},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msg := result["messages"].([]any)[0].(map[string]any)
		arr, ok := msg["content"].([]any)
		if !ok || len(arr) != 2 {
			t.Fatalf("expected 2 content parts, got %v", msg["content"])
		}
	})

	t.Run("assistant single text → string", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role":  "model",
					"parts": []any{map[string]any{"text": "ok"}},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msg := result["messages"].([]any)[0].(map[string]any)
		if msg["content"] != "ok" {
			t.Errorf("content = %v, want 'ok'", msg["content"])
		}
	})

	t.Run("assistant with tool_calls keeps text as string", func(t *testing.T) {
		body := map[string]any{
			"contents": []any{
				map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{"text": "Let me check"},
						map[string]any{
							"functionCall": map[string]any{
								"name": "get_weather",
								"args": map[string]any{},
							},
						},
					},
				},
			},
		}
		result, err := geminiToOpenAIRequest("gemini-pro", body, false, nil)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		msg := result["messages"].([]any)[0].(map[string]any)
		if msg["content"] != "Let me check" {
			t.Errorf("content = %v, want 'Let me check'", msg["content"])
		}
		if _, ok := msg["tool_calls"]; !ok {
			t.Error("expected tool_calls")
		}
	})
}

