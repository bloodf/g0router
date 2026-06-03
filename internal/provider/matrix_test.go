package provider

import (
	"strings"
	"testing"
)

func TestProviderMatrixCoversRemediationParityTiers(t *testing.T) {
	required := []string{
		"openai",
		"anthropic",
		"gemini",
		"antigravity",
		"github-copilot",
		"cursor",
		"deepseek",
		"kimi",
		"qwen",
		"perplexity",
		"openrouter",
		"groq",
		"mistral",
		"cohere",
		"replicate",
		"cerebras",
		"fireworks",
		"together",
		"nvidia",
		"huggingface",
		"nebius",
		"xai",
		"azure",
		"vertex",
		"bedrock",
		"ollama",
		"vercel-ai-gateway",
		"cloudflare-ai-gateway",
		"litellm",
		"vllm",
		"lm-studio",
		"ollama-cloud",
		"kilo",
		"opencode",
		"gitlab",
		"kiro",
		"zhipu",
		"xiaomi",
		"minimax",
		"alibaba",
		"qianfan",
		"tavily",
		"kagi",
	}

	matrix := ProviderMatrix()
	for _, id := range required {
		if _, ok := matrix.Provider(id); !ok {
			t.Fatalf("provider matrix missing %q", id)
		}
	}
}

func TestProviderMatrixMarksAuthOnlyProvidersExplicitly(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"github-copilot", "cursor", "gitlab", "kiro", "kimi", "minimax", "alibaba", "zhipu"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusAuthOnly {
			t.Fatalf("%s status = %q, want auth_only", id, entry.PublicStatus)
		}
		if entry.PublicInference {
			t.Fatalf("%s marked public-inference capable despite auth-only status", id)
		}
	}
}

func TestProviderMatrixMarksRegisteredButUnroutableAdaptersAsAdapterOnly(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"gemini", "groq", "azure", "vertex", "bedrock", "ollama", "cohere", "replicate", "nebius"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusAdapterOnly {
			t.Fatalf("%s status = %q, want adapter_only", id, entry.PublicStatus)
		}
		if !entry.RegisteredAdapter {
			t.Fatalf("%s should mark registered adapter", id)
		}
		if id != "bedrock" && !entry.Inference {
			t.Fatalf("%s should mark adapter inference capability even without public dispatch", id)
		}
		if entry.PublicInference || entry.DirectDispatch {
			t.Fatalf("%s should not be public/direct dispatch yet: %+v", id, entry)
		}
	}
}

func TestProviderMatrixLocksBedrockBehindIncompleteConverseStatus(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("bedrock")
	if !ok {
		t.Fatal("provider matrix missing bedrock")
	}
	if entry.PublicStatus != ProviderStatusAdapterOnly {
		t.Fatalf("bedrock status = %q, want adapter_only", entry.PublicStatus)
	}
	if !entry.RegisteredAdapter {
		t.Fatal("bedrock should mark registered adapter")
	}
	if entry.PublicInference || entry.DirectDispatch || entry.Inference || entry.Streaming || entry.ModelCatalog || entry.ListModels || entry.Quota {
		t.Fatalf("bedrock capabilities should all be false except registered adapter: %+v", entry)
	}
	note := strings.ToLower(entry.Notes)
	if !strings.Contains(note, "converse") || strings.Contains(note, "wave 7.f") {
		t.Fatalf("bedrock notes = %q, want explicit non-Converse status without Wave 7.F TODO", entry.Notes)
	}
}

func TestProviderMatrixKeepsKiroAndKiloDistinct(t *testing.T) {
	matrix := ProviderMatrix()

	kiro, ok := matrix.Provider("kiro")
	if !ok {
		t.Fatal("provider matrix missing kiro")
	}
	if kiro.PublicStatus != ProviderStatusAuthOnly {
		t.Fatalf("kiro status = %q, want auth_only", kiro.PublicStatus)
	}

	kilo, ok := matrix.Provider("kilo")
	if !ok {
		t.Fatal("provider matrix missing kilo")
	}
	if kilo.PublicStatus != ProviderStatusUnsupported {
		t.Fatalf("kilo status = %q, want unsupported", kilo.PublicStatus)
	}
}

func TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries(t *testing.T) {
	public := PublicInferenceProviders()
	ids := providerIDs(public)
	if len(ids) != 2 || !ids["openai"] || !ids["anthropic"] {
		t.Fatalf("public inference providers = %+v, want only openai and anthropic until catalog routing lands", ids)
	}
	for _, entry := range public {
		if entry.PublicStatus != ProviderStatusSupported {
			t.Fatalf("public inference provider %s status = %q, want supported", entry.G0RouterID, entry.PublicStatus)
		}
		if !entry.PublicInference || !entry.DirectDispatch {
			t.Fatalf("public inference provider %s is not direct-dispatch capable: %+v", entry.G0RouterID, entry)
		}
	}

	if ids["github-copilot"] {
		t.Fatal("github-copilot is auth-only today and must not be advertised as an inference provider")
	}
	if ids["cursor"] {
		t.Fatal("cursor is auth-only today and must not be advertised as an inference provider")
	}
	if ids["qwen"] {
		t.Fatal("qwen is unsupported today and must not be advertised as an inference provider")
	}
	if ids["groq"] {
		t.Fatal("groq has an adapter but public dispatch cannot route it until Wave 7.E")
	}
}

func TestProviderMatrixSupportedEntriesHaveUsableSurface(t *testing.T) {
	for _, entry := range ProviderMatrix().Entries() {
		if entry.PublicStatus != ProviderStatusSupported {
			continue
		}
		if !entry.PublicInference || !entry.DirectDispatch {
			t.Fatalf("%s marked supported without public direct dispatch", entry.G0RouterID)
		}
		if len(entry.AuthTypes) == 0 {
			t.Fatalf("%s marked supported without auth type", entry.G0RouterID)
		}
	}
}

func providerIDs(entries []ProviderMatrixEntry) map[string]bool {
	result := make(map[string]bool, len(entries))
	for _, entry := range entries {
		result[entry.G0RouterID] = true
	}
	return result
}
