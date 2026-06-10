package translation

import (
	"testing"
)

func TestNewRegistryWiresOpenAIAntigravityRequest(t *testing.T) {
	reg := NewRegistry()
	if reg.RequestTranslatorFor(FormatOpenAI, FormatAntigravity) == nil {
		t.Error("NewRegistry must wire openai->antigravity request translator")
	}
}

func TestOpenAIAntigravityGeminiModelUsesGeminiCLIEnvelope(t *testing.T) {
	reg := NewRegistry()
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	out, err := reg.TranslateRequest(FormatOpenAI, FormatAntigravity, "gemini-2.0-flash", body, false, nil)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if out["userAgent"] != "antigravity" {
		t.Errorf("userAgent = %v, want antigravity", out["userAgent"])
	}
	if out["requestType"] != "agent" {
		t.Errorf("requestType = %v, want agent", out["requestType"])
	}
	req, ok := out["request"].(map[string]any)
	if !ok {
		t.Fatal("request missing")
	}
	if req["contents"] == nil {
		t.Error("expected contents in request")
	}
}
