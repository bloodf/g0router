package translation

import (
	"testing"
)

func TestNewRegistryWiresGeminiOpenAIResponse(t *testing.T) {
	reg := NewRegistry()
	if reg.ResponseTranslatorFor(FormatGemini, FormatOpenAI) == nil {
		t.Error("NewRegistry must wire gemini->openai response translator")
	}
}

func TestGeminiOpenAIFirstChunkRole(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"responseId":   "resp_123",
		"modelVersion": "gemini-1.5-pro",
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{map[string]any{"text": "hello"}},
				},
			},
		},
	}
	results, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	roleChunk := results[0]
	if roleChunk["id"] != "chatcmpl-resp_123" {
		t.Errorf("id = %v", roleChunk["id"])
	}
	if roleChunk["model"] != "gemini-1.5-pro" {
		t.Errorf("model = %v", roleChunk["model"])
	}
	choices := roleChunk["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	if delta["role"] != "assistant" {
		t.Errorf("delta.role = %v", delta["role"])
	}
	content := results[1]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["content"]
	if content != "hello" {
		t.Errorf("content = %v", content)
	}
}

func TestGeminiOpenAIThoughtParts(t *testing.T) {
	t.Run("thought true → reasoning_content", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"thought": true, "thoughtSignature": "sig", "text": "thinking..."},
						},
					},
				},
			},
		}
		results, err := geminiToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(results) < 2 {
			t.Fatalf("len(results) = %d", len(results))
		}
		thoughtChunk := results[1]
		delta := thoughtChunk["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if delta["reasoning_content"] != "thinking..." {
			t.Errorf("reasoning_content = %v", delta["reasoning_content"])
		}
	})

	t.Run("signature without thought → content", func(t *testing.T) {
		state := NewStreamState()
		chunk := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"thoughtSignature": "sig", "text": "plain text"},
						},
					},
				},
			},
		}
		results, err := geminiToOpenAIResponse(chunk, state)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if len(results) < 2 {
			t.Fatalf("len(results) = %d", len(results))
		}
		textChunk := results[1]
		delta := textChunk["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
		if delta["content"] != "plain text" {
			t.Errorf("content = %v", delta["content"])
		}
	})
}

func TestGeminiOpenAIFunctionCallDelta(t *testing.T) {
	state := NewStreamState()
	state.ToolNameMap = map[string]string{"cloaked": "original"}
	chunk := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{"functionCall": map[string]any{"name": "cloaked", "args": map[string]any{"loc": "NYC"}}},
					},
				},
			},
		},
	}
	results, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	tcChunk := results[1]
	delta := tcChunk["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	toolCalls := delta["tool_calls"].([]any)
	if len(toolCalls) != 1 {
		t.Fatalf("len(toolCalls) = %d", len(toolCalls))
	}
	tc := toolCalls[0].(map[string]any)
	if tc["index"] != 0 {
		t.Errorf("index = %v", tc["index"])
	}
	if tc["type"] != "function" {
		t.Errorf("type = %v", tc["type"])
	}
	fn := tc["function"].(map[string]any)
	if fn["name"] != "original" {
		t.Errorf("name = %v", fn["name"])
	}
	if fn["arguments"] != `{"loc":"NYC"}` {
		t.Errorf("arguments = %v", fn["arguments"])
	}

	// Second call should have sequential index
	results2, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(results2) < 1 {
		t.Fatalf("len(results2) = %d", len(results2))
	}
	tcChunk2 := results2[0]
	delta2 := tcChunk2["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	toolCalls2 := delta2["tool_calls"].([]any)
	tc2 := toolCalls2[0].(map[string]any)
	if tc2["index"] != 1 {
		t.Errorf("second index = %v", tc2["index"])
	}
}

func TestGeminiOpenAIInlineImage(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{"inlineData": map[string]any{"mimeType": "image/png", "data": "abc123"}},
					},
				},
			},
		},
	}
	results, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	imgChunk := results[1]
	delta := imgChunk["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)
	images := delta["images"].([]any)
	if len(images) != 1 {
		t.Fatalf("len(images) = %d", len(images))
	}
	img := images[0].(map[string]any)
	if img["type"] != "image_url" {
		t.Errorf("type = %v", img["type"])
	}
	url := img["image_url"].(map[string]any)["url"].(string)
	if url != "data:image/png;base64,abc123" {
		t.Errorf("url = %v", url)
	}
}

func TestGeminiOpenAIUsageMapping(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"candidates": []any{
			map[string]any{
				"content":      map[string]any{"parts": []any{}},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":        float64(10),
			"candidatesTokenCount":    float64(0),
			"thoughtsTokenCount":      float64(2),
			"totalTokenCount":         float64(20),
			"cachedContentTokenCount": float64(5),
		},
	}
	results, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	var finalChunk map[string]any
	for _, r := range results {
		c := r
		choices := c["choices"].([]any)
		if choices[0].(map[string]any)["finish_reason"] != nil {
			finalChunk = c
			break
		}
	}
	if finalChunk == nil {
		t.Fatal("no final chunk found")
	}
	usage := finalChunk["usage"].(map[string]any)
	if usage["prompt_tokens"] != float64(10) {
		t.Errorf("prompt_tokens = %v", usage["prompt_tokens"])
	}
	// candidates fallback: 20 - 10 - 2 = 8; completion = 8 + 2 = 10
	if usage["completion_tokens"] != float64(10) {
		t.Errorf("completion_tokens = %v", usage["completion_tokens"])
	}
	if usage["total_tokens"] != float64(20) {
		t.Errorf("total_tokens = %v", usage["total_tokens"])
	}
	promptDetails := usage["prompt_tokens_details"].(map[string]any)
	if promptDetails["cached_tokens"] != float64(5) {
		t.Errorf("cached_tokens = %v", promptDetails["cached_tokens"])
	}
	completionDetails := usage["completion_tokens_details"].(map[string]any)
	if completionDetails["reasoning_tokens"] != float64(2) {
		t.Errorf("reasoning_tokens = %v", completionDetails["reasoning_tokens"])
	}
}

func TestGeminiOpenAIFinishReasonToolCalls(t *testing.T) {
	state := NewStreamState()
	state.ToolCalls[0] = ToolCallInfo{ID: "tc1", Name: "tool"}

	chunk := map[string]any{
		"candidates": []any{
			map[string]any{
				"content":      map[string]any{"parts": []any{}},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     float64(5),
			"candidatesTokenCount": float64(3),
			"totalTokenCount":      float64(8),
		},
	}
	results, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	var finalChunk map[string]any
	for _, r := range results {
		c := r
		choices := c["choices"].([]any)
		if choices[0].(map[string]any)["finish_reason"] != nil {
			finalChunk = c
			break
		}
	}
	if finalChunk == nil {
		t.Fatal("no final chunk found")
	}
	choices := finalChunk["choices"].([]any)
	if choices[0].(map[string]any)["finish_reason"] != "tool_calls" {
		t.Errorf("finish_reason = %v", choices[0].(map[string]any)["finish_reason"])
	}
	if finalChunk["usage"] == nil {
		t.Fatal("usage missing on final chunk")
	}
}

func TestGeminiOpenAIResponseKeyUnwrap(t *testing.T) {
	state := NewStreamState()
	chunk := map[string]any{
		"response": map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "hello"}},
					},
				},
			},
		},
	}
	results, err := geminiToOpenAIResponse(chunk, state)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	content := results[1]["choices"].([]any)[0].(map[string]any)["delta"].(map[string]any)["content"]
	if content != "hello" {
		t.Errorf("content = %v", content)
	}
}
