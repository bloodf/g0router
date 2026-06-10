package translation

import "testing"

func TestFormatConstants(t *testing.T) {
	want := map[Format]string{
		FormatOpenAI:          "openai",
		FormatOpenAIResponses: "openai-responses",
		FormatOpenAIResponse:  "openai-response",
		FormatClaude:          "claude",
		FormatGemini:          "gemini",
		FormatGeminiCLI:       "gemini-cli",
		FormatVertex:          "vertex",
		FormatCodex:           "codex",
		FormatAntigravity:     "antigravity",
		FormatKiro:            "kiro",
		FormatCursor:          "cursor",
		FormatOllama:          "ollama",
		FormatCommandCode:     "commandcode",
	}
	for f, expected := range want {
		if string(f) != expected {
			t.Errorf("format %q = %q, want %q", f, string(f), expected)
		}
	}
}

func TestDetectFormatByEndpointResponses(t *testing.T) {
	if got := DetectFormatByEndpoint("/v1/responses", false); got != FormatOpenAIResponses {
		t.Errorf("responses = %q, want %q", got, FormatOpenAIResponses)
	}
}

func TestDetectFormatByEndpointMessages(t *testing.T) {
	if got := DetectFormatByEndpoint("/v1/messages", false); got != FormatClaude {
		t.Errorf("messages = %q, want %q", got, FormatClaude)
	}
}

func TestDetectFormatByEndpointChatWithInput(t *testing.T) {
	// Cursor CLI sends Responses-shaped body via chat endpoint; 9router
	// deliberately treats it as openai, not openai-responses.
	if got := DetectFormatByEndpoint("/v1/chat/completions", true); got != FormatOpenAI {
		t.Errorf("chat+input[] = %q, want %q", got, FormatOpenAI)
	}
}

func TestDetectFormatByEndpointChatWithoutInput(t *testing.T) {
	if got := DetectFormatByEndpoint("/v1/chat/completions", false); got != "" {
		t.Errorf("chat without input[] = %q, want empty", got)
	}
}

func TestDetectFormatByEndpointUnknown(t *testing.T) {
	if got := DetectFormatByEndpoint("/v1/unknown", false); got != "" {
		t.Errorf("unknown = %q, want empty", got)
	}
}
