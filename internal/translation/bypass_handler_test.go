package translation

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestBypassNonClaudeCliNil(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "{"}}}},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "some-other-agent", false, reg)
	if bypassed {
		t.Error("expected no bypass for non-claude-cli userAgent")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func TestBypassTitleExtraction(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "text", "text": "{"}}}},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if len(resp) == 0 {
		t.Fatal("expected non-empty response")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassWarmup(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Warmup"}}},
		},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassCount(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "count"}}},
		},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassSkipPatterns(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Please write a 5-10 word title for the following conversation:"}}},
		},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassNamingIsNewTopic(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Hello world from test"}}},
		},
		"system": []any{map[string]any{"type": "text", "text": "isNewTopic"}},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", true, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	// Body is Claude-shaped, so response is Claude-shaped; extract from text deltas.
	content := extractClaudeBypassContent(resp)
	if content == "" {
		// Fallback for OpenAI-shaped response (nil registry or openai source).
		content = extractBypassContent(resp)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("content not JSON: %v", content)
	}
	if parsed["isNewTopic"] != true {
		t.Errorf("isNewTopic = %v, want true", parsed["isNewTopic"])
	}
	if parsed["title"] != "Hello world from" {
		t.Errorf("title = %q, want 'Hello world from'", parsed["title"])
	}
}

func TestBypassNamingJSONEscapesTitle(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": `Hello "world" \ test`}}},
		},
		"system": []any{map[string]any{"type": "text", "text": "isNewTopic"}},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", true, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	content := extractClaudeBypassContent(resp)
	if content == "" {
		content = extractBypassContent(resp)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("content not valid JSON: %v\ncontent=%q", err, content)
	}
	if parsed["isNewTopic"] != true {
		t.Errorf("isNewTopic = %v, want true", parsed["isNewTopic"])
	}
	wantTitle := `Hello "world" \`
	if parsed["title"] != wantTitle {
		t.Errorf("title = %q, want %q", parsed["title"], wantTitle)
	}
}

func TestBypassTranslationErrorPropagated(t *testing.T) {
	errReg := NewRegistry()
	errReg.Register(FormatOpenAI, FormatClaude, nil, func(chunk map[string]any, state *StreamState) ([]map[string]any, error) {
		return nil, fmt.Errorf("stub translation error")
	})
	body := map[string]any{
		"model":  "claude-3-opus-20240229",
		"stream": true,
		"system": []any{map[string]any{"type": "text", "text": "You are helpful"}},
		"messages": []any{map[string]any{"role": "user", "content": "Warmup"}},
	}
	_, bypassed, err := HandleBypassRequest(body, "claude-3-opus-20240229", "claude-cli/1.0", false, errReg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if err == nil {
		t.Fatal("expected translation error to be propagated")
	}
	if !strings.Contains(err.Error(), "stub translation error") {
		t.Errorf("error = %q, want to contain 'stub translation error'", err.Error())
	}
}

func TestBypassNilRegistryClaudeNonStreamingError(t *testing.T) {
	body := map[string]any{
		"model":    "claude-3-opus-20240229",
		"stream":   false,
		"system":   []any{map[string]any{"type": "text", "text": "You are helpful"}},
		"messages": []any{map[string]any{"role": "user", "content": "Warmup"}},
	}
	_, bypassed, err := HandleBypassRequest(body, "claude-3-opus-20240229", "claude-cli/1.0", false, nil)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if err == nil {
		t.Fatal("expected error for nil registry with claude source format")
	}
	if !strings.Contains(err.Error(), "registry required") {
		t.Errorf("error = %q, want to contain 'registry required'", err.Error())
	}
}

func TestBypassNilRegistryClaudeStreamingError(t *testing.T) {
	body := map[string]any{
		"model":    "claude-3-opus-20240229",
		"stream":   true,
		"system":   []any{map[string]any{"type": "text", "text": "You are helpful"}},
		"messages": []any{map[string]any{"role": "user", "content": "Warmup"}},
	}
	_, bypassed, err := HandleBypassRequest(body, "claude-3-opus-20240229", "claude-cli/1.0", false, nil)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if err == nil {
		t.Fatal("expected error for nil registry with claude source format")
	}
	if !strings.Contains(err.Error(), "registry required") {
		t.Errorf("error = %q, want to contain 'registry required'", err.Error())
	}
}

func TestBypassNilRegistryOpenAIReturnsChunk(t *testing.T) {
	body := map[string]any{
		"model":    "gpt-4",
		"stream":   false,
		"messages": []any{map[string]any{"role": "user", "content": "Warmup"}},
	}
	resp, bypassed, err := HandleBypassRequest(body, "gpt-4", "claude-cli/1.0", false, nil)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(resp))
	}
	if _, ok := resp[0]["id"]; !ok {
		t.Error("expected OpenAI id field")
	}
}

func TestBypassNoMatchNil(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{
			map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "regular message"}}},
		},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3", "claude-cli/1.0", false, reg)
	if bypassed {
		t.Error("expected no bypass")
	}
	if resp != nil {
		t.Errorf("expected nil response, got %v", resp)
	}
}

func TestBypassClaudeSourceFormatResponse(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"model":    "claude-3-opus-20240229",
		"stream":   true,
		"system":   []any{map[string]any{"type": "text", "text": "You are helpful"}},
		"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Warmup"}}},
		},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3-opus-20240229", "claude-cli/1.0", false, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if len(resp) == 0 {
		t.Fatal("expected non-empty response")
	}
	// Claude-shaped response must include message_start as the first event.
	firstType, _ := resp[0]["type"].(string)
	if firstType != "message_start" {
		t.Fatalf("expected first chunk type=message_start for claude source, got %q: %v", firstType, resp[0])
	}
	// Concatenate text deltas to verify content.
	content := extractClaudeBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
	// Must see a message_delta or message_stop finish event.
	var sawFinish bool
	for _, chunk := range resp {
		if t, _ := chunk["type"].(string); t == "message_delta" || t == "message_stop" {
			sawFinish = true
			break
		}
	}
	if !sawFinish {
		t.Errorf("expected claude finish event (message_delta/message_stop) in response")
	}
}

func TestBypassOpenAISourceFormatResponse(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"model":    "gpt-4",
		"stream":   true,
		"messages": []any{map[string]any{"role": "user", "content": "Warmup"}},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "gpt-4", "claude-cli/1.0", false, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if len(resp) == 0 {
		t.Fatal("expected non-empty response")
	}
	// OpenAI-shaped response: first chunk must have id, object, choices.
	if _, ok := resp[0]["id"]; !ok {
		t.Fatalf("expected OpenAI id field, got %v", resp[0])
	}
	obj, _ := resp[0]["object"].(string)
	if obj != "chat.completion.chunk" {
		t.Fatalf("expected object=chat.completion.chunk for openai source, got %q", obj)
	}
	choices, ok := resp[0]["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatalf("expected OpenAI choices array, got %v", resp[0]["choices"])
	}
	// Must NOT contain claude-specific type fields.
	for _, chunk := range resp {
		if typ, _ := chunk["type"].(string); typ != "" {
			t.Errorf("expected no claude type field in openai source response, got %q", typ)
		}
	}
	content := extractBypassContent(resp)
	if content != defaultBypassText {
		t.Errorf("content = %q, want %q", content, defaultBypassText)
	}
}

func TestBypassNamingSourceFormat(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"model":    "claude-3-opus-20240229",
		"stream":   true,
		"system":   []any{map[string]any{"type": "text", "text": "isNewTopic"}},
		"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "text", "text": "Hello world from test"}}},
		},
	}
	resp, bypassed, _ := HandleBypassRequest(body, "claude-3-opus-20240229", "claude-cli/1.0", true, reg)
	if !bypassed {
		t.Fatal("expected bypass")
	}
	if len(resp) == 0 {
		t.Fatal("expected non-empty response")
	}
	// Claude source format → claude-shaped chunks.
	firstType, _ := resp[0]["type"].(string)
	if firstType != "message_start" {
		t.Fatalf("expected first chunk type=message_start for claude source naming bypass, got %q", firstType)
	}
	content := extractClaudeBypassContent(resp)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("naming content not JSON: %v", content)
	}
	if parsed["isNewTopic"] != true {
		t.Errorf("isNewTopic = %v, want true", parsed["isNewTopic"])
	}
	if parsed["title"] != "Hello world from" {
		t.Errorf("title = %q, want 'Hello world from'", parsed["title"])
	}
}

func extractBypassContent(resp []map[string]any) string {
	if len(resp) == 0 {
		return ""
	}
	// Non-streaming: single response with choices[0].message.content
	if len(resp) == 1 {
		choices, ok := resp[0]["choices"].([]any)
		if ok && len(choices) > 0 {
			choice := choices[0].(map[string]any)
			msg, ok := choice["message"].(map[string]any)
			if ok {
				return msg["content"].(string)
			}
			// Streaming final chunk
			delta, ok := choice["delta"].(map[string]any)
			if ok {
				if content, ok := delta["content"].(string); ok {
					return content
				}
			}
		}
	}
	// Streaming: concatenate content from chunks
	var parts []string
	for _, chunk := range resp {
		choices, ok := chunk["choices"].([]any)
		if !ok || len(choices) == 0 {
			continue
		}
		choice := choices[0].(map[string]any)
		delta, ok := choice["delta"].(map[string]any)
		if !ok {
			continue
		}
		if content, ok := delta["content"].(string); ok && content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "")
}

func extractClaudeBypassContent(resp []map[string]any) string {
	var parts []string
	for _, chunk := range resp {
		t, _ := chunk["type"].(string)
		if t != "content_block_delta" {
			continue
		}
		delta, ok := chunk["delta"].(map[string]any)
		if !ok {
			continue
		}
		if delta["type"] != "text_delta" {
			continue
		}
		if text, ok := delta["text"].(string); ok && text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "")
}
