package translation

import (
	"testing"
)

func TestNewRegistryWiresOpenAIGeminiCLIRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatOpenAI, FormatGeminiCLI) == nil {
		t.Error("NewRegistry must wire openai->gemini-cli request translator")
	}
}

func TestOpenAIGeminiCLIThinkingConfigFromReasoningEffort(t *testing.T) {
	cases := []struct {
		effort string
		want   int
	}{
		{"low", 1024},
		{"medium", 8192},
		{"high", 32768},
	}
	for _, tc := range cases {
		t.Run(tc.effort, func(t *testing.T) {
			body := map[string]any{
				"messages":         []any{map[string]any{"role": "user", "content": "hi"}},
				"reasoning_effort": tc.effort,
			}
			out, err := openaiToGeminiCLIRequest("gemini-pro", body, false, nil)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			genConfig := out["generationConfig"].(map[string]any)
			thinkingConfig := genConfig["thinkingConfig"].(map[string]any)
			if thinkingConfig["thinkingBudget"] != float64(tc.want) {
				t.Errorf("thinkingBudget = %v, want %d", thinkingConfig["thinkingBudget"], tc.want)
			}
			if thinkingConfig["include_thoughts"] != true {
				t.Errorf("include_thoughts = %v, want true", thinkingConfig["include_thoughts"])
			}
		})
	}
}

func TestOpenAIGeminiCLIEnvelopeShape(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatGeminiCLI, "gemini-pro", body, false, nil)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if out["project"] == nil {
		t.Error("expected project in envelope")
	}
	if out["userAgent"] != "gemini-cli" {
		t.Errorf("userAgent = %v, want gemini-cli", out["userAgent"])
	}
	req, ok := out["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	if req["contents"] == nil {
		t.Error("expected contents in request")
	}
}
