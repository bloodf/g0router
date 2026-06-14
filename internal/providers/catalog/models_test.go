package catalog

import (
	"testing"
)

func TestModelsForDeepSeek(t *testing.T) {
	models := ModelsFor("deepseek")
	if len(models) != 6 {
		t.Fatalf("ModelsFor(\"deepseek\") len = %d, want 6", len(models))
	}

	// Verify -max and -none aliases map to the correct upstream model.
	for _, id := range []string{"deepseek-v4-pro-max", "deepseek-v4-pro-none"} {
		m, ok := ResolveModel("deepseek", id)
		if !ok {
			t.Fatalf("ResolveModel(\"deepseek\", %q) not found", id)
		}
		if m.UpstreamModelID != "deepseek-v4-pro" {
			t.Errorf("%q UpstreamModelID = %q, want %q", id, m.UpstreamModelID, "deepseek-v4-pro")
		}
	}
}

func TestModelTypeVerbatim(t *testing.T) {
	// A model without a type field should have Type == "" (NOT defaulted to "llm").
	m, ok := ResolveModel("deepseek", "deepseek-v4-pro")
	if !ok {
		t.Fatal("ResolveModel(\"deepseek\", \"deepseek-v4-pro\") not found")
	}
	if m.Type != "" {
		t.Errorf("no-type entry Type = %q, want zero value", m.Type)
	}

	// An STT entry should have Type == "stt".
	m, ok = ResolveModel("groq", "whisper-large-v3")
	if !ok {
		t.Fatal("ResolveModel(\"groq\", \"whisper-large-v3\") not found")
	}
	if m.Type != "stt" {
		t.Errorf("stt entry Type = %q, want %q", m.Type, "stt")
	}
}

func TestGroqSTTModels(t *testing.T) {
	models := ModelsFor("groq")
	var sttCount int
	for _, m := range models {
		if m.Type == "stt" {
			sttCount++
		}
	}
	if sttCount != 3 {
		t.Fatalf("groq STT models = %d, want 3", sttCount)
	}
}

func TestResolveModelUpstream(t *testing.T) {
	m, ok := ResolveModel("deepseek", "deepseek-v4-pro-max")
	if !ok {
		t.Fatal("ResolveModel not found")
	}
	if m.ID != "deepseek-v4-pro-max" {
		t.Errorf("ID = %q, want %q", m.ID, "deepseek-v4-pro-max")
	}
	if m.UpstreamModelID != "deepseek-v4-pro" {
		t.Errorf("UpstreamModelID = %q, want %q", m.UpstreamModelID, "deepseek-v4-pro")
	}
}

func TestOpenRouterCatalogTypes(t *testing.T) {
	models := ModelsFor("openrouter")
	if len(models) == 0 {
		t.Fatal("ModelsFor(\"openrouter\") is empty")
	}

	var hasEmbedding, hasTTS, hasImage bool
	for _, m := range models {
		switch m.Type {
		case "embedding":
			hasEmbedding = true
		case "tts":
			hasTTS = true
		case "image":
			hasImage = true
		}
		if m.Type == "image" && len(m.Params) == 0 {
			t.Errorf("image entry %q has empty Params, want non-empty", m.ID)
		}
	}
	if !hasEmbedding {
		t.Error("openrouter missing embedding type")
	}
	if !hasTTS {
		t.Error("openrouter missing tts type")
	}
	if !hasImage {
		t.Error("openrouter missing image type")
	}
}

func TestModelsForOllama(t *testing.T) {
	models := ModelsFor("ollama")
	if len(models) == 0 {
		t.Fatal("ModelsFor(\"ollama\") is empty")
	}
	// Verify the static block has the expected 6 entries.
	if len(models) != 6 {
		t.Errorf("ModelsFor(\"ollama\") len = %d, want 6", len(models))
	}
	want := []string{
		"gpt-oss:120b",
		"kimi-k2.5",
		"glm-5",
		"minimax-m2.5",
		"glm-4.7-flash",
		"qwen3.5",
	}
	for i, id := range want {
		if models[i].ID != id {
			t.Errorf("ModelsFor(\"ollama\")[%d].ID = %q, want %q", i, models[i].ID, id)
		}
	}
}

func TestChineseOpenAIModels(t *testing.T) {
	wantCount := map[string]int{
		"glm-cn":         5,
		"alicode":        8,
		"alicode-intl":   7,
		"volcengine-ark": 9,
		"byteplus":       7,
		"xiaomi-mimo":    4,
		"opencode-go":    10,
	}
	for p, n := range wantCount {
		if got := len(ModelsFor(p)); got != n {
			t.Errorf("ModelsFor(%q) len = %d, want %d", p, got, n)
		}
	}

	wantFirst := map[string]string{
		"glm-cn":         "glm-5.1",
		"alicode":        "qwen3.5-plus",
		"alicode-intl":   "qwen3.5-plus",
		"volcengine-ark": "Doubao-Seed-2.0-Code",
		"byteplus":       "seed-2-0-pro-260328",
		"xiaomi-mimo":    "mimo-v2.5-pro",
		"opencode-go":    "kimi-k2.6",
	}
	for p, first := range wantFirst {
		models := ModelsFor(p)
		if len(models) == 0 {
			t.Errorf("ModelsFor(%q) is empty", p)
			continue
		}
		if models[0].ID != first {
			t.Errorf("ModelsFor(%q)[0].ID = %q, want %q", p, models[0].ID, first)
		}
	}

	// opencode has an empty ref model block.
	if got := len(ModelsFor("opencode")); got != 0 {
		t.Errorf("ModelsFor(\"opencode\") len = %d, want 0", got)
	}
}

// TestClaudeFormatModels (w7-prov-special-a) verifies the model blocks for the
// claude-format providers transcribed verbatim from providerModels.js @827e5c3.
func TestClaudeFormatModels(t *testing.T) {
	wantCount := map[string]int{
		"glm":     4,
		"kimi":    4,
		"minimax": 5,
	}
	for p, n := range wantCount {
		if got := len(ModelsFor(p)); got != n {
			t.Errorf("ModelsFor(%q) len = %d, want %d", p, got, n)
		}
	}
	wantFirst := map[string]string{
		"glm":     "glm-5.1",
		"kimi":    "kimi-k2.6",
		"minimax": "MiniMax-M3",
	}
	for p, first := range wantFirst {
		models := ModelsFor(p)
		if len(models) == 0 {
			t.Errorf("ModelsFor(%q) is empty", p)
			continue
		}
		if models[0].ID != first {
			t.Errorf("ModelsFor(%q)[0].ID = %q, want %q", p, models[0].ID, first)
		}
	}
}

func TestModelsForUnknown(t *testing.T) {
	models := ModelsFor("nonexistent")
	if len(models) != 0 {
		t.Fatalf("ModelsFor(\"nonexistent\") len = %d, want 0", len(models))
	}
}
