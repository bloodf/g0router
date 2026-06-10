package translation

import (
	"testing"
)

func TestNewRegistryWiresOpenAIVertexRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatOpenAI, FormatVertex) == nil {
		t.Error("NewRegistry must wire openai->vertex request translator")
	}
}

func TestOpenAIVertexReplacesThoughtSignature(t *testing.T) {
	body := map[string]any{
		"messages": []any{
			map[string]any{
				"role":              "assistant",
				"content":           "",
				"reasoning_content": "thinking...",
			},
		},
	}
	out, err := openaiToVertexRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	contents := out["contents"].([]any)
	turn := contents[0].(map[string]any)
	parts := turn["parts"].([]any)
	found := false
	for _, p := range parts {
		part := p.(map[string]any)
		if _, ok := part["thoughtSignature"]; ok {
			found = true
			if part["thoughtSignature"] != defaultThinkingVertexSignature {
				t.Errorf("thoughtSignature not replaced with vertex signature: %v", part["thoughtSignature"])
			}
		}
	}
	if !found {
		t.Fatal("expected at least one part with thoughtSignature for assistant reasoning_content")
	}
}

func TestOpenAIVertexStripsFunctionIDs(t *testing.T) {
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
							"name":      "get_weather",
							"arguments": `{"location":"NYC"}`,
						},
					},
				},
			},
			map[string]any{
				"role":         "tool",
				"tool_call_id": "call-1",
				"content":      `{"temp":72}`,
			},
		},
	}
	out, err := openaiToVertexRequest("gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	contents := out["contents"].([]any)
	for _, c := range contents {
		turn := c.(map[string]any)
		parts, ok := turn["parts"].([]any)
		if !ok {
			continue
		}
		for _, p := range parts {
			part := p.(map[string]any)
			if fc, ok := part["functionCall"].(map[string]any); ok {
				if _, ok := fc["id"]; ok {
					t.Error("functionCall.id should be stripped for Vertex")
				}
			}
			if fr, ok := part["functionResponse"].(map[string]any); ok {
				if _, ok := fr["id"]; ok {
					t.Error("functionResponse.id should be stripped for Vertex")
				}
			}
		}
	}
}
