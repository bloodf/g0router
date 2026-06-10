package translation

import (
	"strings"
	"testing"
)

func TestWrapCloudCodeEnvelopeGeminiCLIHasSafetySettings(t *testing.T) {
	gemini := map[string]any{
		"contents":         []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
		"generationConfig": map[string]any{"temperature": 0.7},
		"safetySettings":   defaultSafetySettings(),
	}
	env := wrapInCloudCodeEnvelope("gemini-pro", gemini, nil, false)
	req, ok := env["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	if _, ok := req["safetySettings"]; !ok {
		t.Error("gemini-cli envelope must keep safetySettings")
	}
	if env["userAgent"] != "gemini-cli" {
		t.Errorf("userAgent = %v, want gemini-cli", env["userAgent"])
	}
}

func TestWrapCloudCodeEnvelopeAntigravityHasValidatedToolConfig(t *testing.T) {
	gemini := map[string]any{
		"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
		"tools":    []any{map[string]any{"functionDeclarations": []any{map[string]any{"name": "tool"}}}},
	}
	env := wrapInCloudCodeEnvelope("gemini-pro", gemini, nil, true)
	req, ok := env["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	if env["userAgent"] != "antigravity" {
		t.Errorf("userAgent = %v, want antigravity", env["userAgent"])
	}
	if env["requestType"] != "agent" {
		t.Errorf("requestType = %v, want agent", env["requestType"])
	}
	toolConfig, ok := req["toolConfig"].(map[string]any)
	if !ok {
		t.Fatal("toolConfig missing")
	}
	fcc, ok := toolConfig["functionCallingConfig"].(map[string]any)
	if !ok {
		t.Fatal("functionCallingConfig missing")
	}
	if fcc["mode"] != "VALIDATED" {
		t.Errorf("mode = %v, want VALIDATED", fcc["mode"])
	}
	if _, ok := req["safetySettings"]; ok {
		t.Error("antigravity envelope must not have safetySettings")
	}
}

func TestWrapCloudCodeEnvelopeUsesCredentialsProjectId(t *testing.T) {
	gemini := map[string]any{
		"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
	}
	creds := map[string]any{"projectId": "my-project-123"}
	env := wrapInCloudCodeEnvelope("gemini-pro", gemini, creds, false)
	if env["project"] != "my-project-123" {
		t.Errorf("project = %v, want my-project-123", env["project"])
	}
}

func TestWrapCloudCodeEnvelopeDerivesSessionFromConnectionId(t *testing.T) {
	gemini := map[string]any{
		"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
	}
	creds := map[string]any{"connectionId": "conn-456"}
	env := wrapInCloudCodeEnvelope("gemini-pro", gemini, creds, true)
	req, ok := env["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	sessionId, ok := req["sessionId"].(string)
	if !ok || sessionId == "" {
		t.Fatal("sessionId missing or empty")
	}
	// With connectionId, deriveSessionId should produce a stable id.
	env2 := wrapInCloudCodeEnvelope("gemini-pro", gemini, creds, true)
	req2 := env2["request"].(map[string]any)
	if req2["sessionId"] != sessionId {
		t.Error("sessionId should be stable for same connectionId")
	}
}

func TestWrapCloudCodeEnvelopeForClaudeToolBlocks(t *testing.T) {
	claudeReq := map[string]any{
		"model":      "claude-3-5-sonnet",
		"max_tokens": 4096,
		"messages": []any{
			map[string]any{
				"role": "assistant",
				"content": []any{
					map[string]any{"type": "tool_use", "id": "tu1", "name": "Read", "input": map[string]any{"path": "/tmp"}},
				},
			},
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "tool_result", "tool_use_id": "tu1", "content": "ok"},
				},
			},
		},
		"tools": []any{
			map[string]any{"name": "Read", "description": "read", "input_schema": map[string]any{"type": "object", "properties": map[string]any{}}},
		},
	}
	env := wrapInCloudCodeEnvelopeForClaude("claude-3-5-sonnet", claudeReq, nil)
	if env["userAgent"] != "antigravity" {
		t.Errorf("userAgent = %v, want antigravity", env["userAgent"])
	}
	req, ok := env["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	contents, ok := req["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("contents missing or empty: %v", req["contents"])
	}
	// First message is assistant with tool_use → model role with functionCall
	first := contents[0].(map[string]any)
	if first["role"] != "model" {
		t.Errorf("first role = %v, want model", first["role"])
	}
	parts := first["parts"].([]any)
	fc, ok := parts[0].(map[string]any)["functionCall"].(map[string]any)
	if !ok {
		t.Fatal("functionCall missing in first content")
	}
	if fc["name"] != "Read" {
		t.Errorf("functionCall.name = %v, want Read", fc["name"])
	}
	// Second message is tool_result → user role with functionResponse
	second := contents[1].(map[string]any)
	if second["role"] != "user" {
		t.Errorf("second role = %v, want user", second["role"])
	}
	sparts := second["parts"].([]any)
	fr, ok := sparts[0].(map[string]any)["functionResponse"].(map[string]any)
	if !ok {
		t.Fatal("functionResponse missing in second content")
	}
	if fr["name"] != "Read" {
		t.Errorf("functionResponse.name = %v, want Read", fr["name"])
	}
	// Tools converted to functionDeclarations
	tools, ok := req["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools missing: %v", req["tools"])
	}
	fds := tools[0].(map[string]any)["functionDeclarations"].([]any)
	if len(fds) != 1 {
		t.Fatalf("expected 1 functionDeclaration, got %d", len(fds))
	}
	// toolConfig present
	toolConfig, ok := req["toolConfig"].(map[string]any)
	if !ok {
		t.Fatal("toolConfig missing")
	}
	fcc := toolConfig["functionCallingConfig"].(map[string]any)
	if fcc["mode"] != "VALIDATED" {
		t.Errorf("mode = %v, want VALIDATED", fcc["mode"])
	}
	// System instruction has antigravity default + Claude system stripped
	sysInstr, ok := req["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatal("systemInstruction missing")
	}
	sysParts := sysInstr["parts"].([]any)
	foundDefault := false
	for _, p := range sysParts {
		text := p.(map[string]any)["text"].(string)
		if strings.Contains(text, "You are Antigravity") {
			foundDefault = true
			break
		}
	}
	if !foundDefault {
		t.Error("systemInstruction missing Antigravity default")
	}
}
