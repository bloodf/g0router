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
	for _, id := range []string{"github-copilot", "cursor", "gitlab", "kiro", "kimi", "alibaba", "zhipu"} {
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
	for _, id := range []string{"azure", "bedrock", "replicate"} {
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
		if !entry.Inference {
			t.Fatalf("%s should mark adapter inference capability even without public dispatch", id)
		}
		if entry.PublicInference || entry.DirectDispatch {
			t.Fatalf("%s should not be public/direct dispatch yet: %+v", id, entry)
		}
	}
}

func TestProviderMatrixKeepsBedrockAdapterOnlyAfterConverseSupport(t *testing.T) {
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
	if entry.PublicInference || entry.DirectDispatch || entry.Streaming || entry.ModelCatalog || entry.Quota {
		t.Fatalf("bedrock public/streaming/catalog/quota capabilities should stay false: %+v", entry)
	}
	if !entry.Inference {
		t.Fatalf("bedrock should expose adapter-only non-streaming Converse inference: %+v", entry)
	}
	if !entry.ListModels {
		t.Fatalf("bedrock should expose signed foundation model listing: %+v", entry)
	}
	note := strings.ToLower(entry.Notes)
	if !strings.Contains(note, "converse") || !strings.Contains(note, "list") || strings.Contains(note, "wave 7.f") {
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
	want := map[string]bool{
		"openai":            true,
		"anthropic":         true,
		"cerebras":          true,
		"cohere":            true,
		"deepseek":          true,
		"fireworks":         true,
		"gemini":            true,
		"groq":              true,
		"huggingface":       true,
		"mistral":           true,
		"minimax":           true,
		"nebius":            true,
		"nvidia":            true,
		"ollama":            true,
		"openrouter":        true,
		"perplexity":        true,
		"qwen":              true,
		"together":          true,
		"vercel-ai-gateway": true,
		"vertex":            true,
		"xai":               true,
	}
	if len(ids) != len(want) {
		t.Fatalf("public inference providers = %+v, want %+v", ids, want)
	}
	for id := range want {
		if !ids[id] {
			t.Fatalf("public inference providers = %+v, missing %s", ids, id)
		}
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
	for _, id := range []string{"litellm", "lm-studio", "replicate", "vllm"} {
		if ids[id] {
			t.Fatalf("%s remains adapter-only and must not be advertised as a public inference provider", id)
		}
	}
}

func TestPublicOpenAICompatibleProvidersDoNotClaimQuotaSupport(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"anthropic", "cerebras", "cohere", "deepseek", "fireworks", "groq", "huggingface", "mistral", "minimax", "nebius", "nvidia", "ollama", "openai", "openrouter", "perplexity", "qwen", "together", "vercel-ai-gateway", "xai"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusSupported {
			t.Fatalf("%s status = %q, want supported", id, entry.PublicStatus)
		}
		if !entry.PublicInference || !entry.DirectDispatch || !entry.RegisteredAdapter || !entry.Inference {
			t.Fatalf("%s supported surface is incomplete: %+v", id, entry)
		}
		if !entry.Streaming || !entry.ModelCatalog || !entry.ListModels {
			t.Fatalf("%s should expose shared OpenAI-compatible streaming and model APIs: %+v", id, entry)
		}
		if entry.Quota {
			t.Fatalf("%s should not claim quota support until a real quota fetcher exists", id)
		}
	}
}

func TestOpenAICompatibleGatewayProvidersAreRegisteredWithoutFakeCatalogs(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"litellm", "lm-studio", "vllm"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusAdapterOnly {
			t.Fatalf("%s status = %q, want adapter_only", id, entry.PublicStatus)
		}
		if !entry.RegisteredAdapter || !entry.Inference || !entry.Streaming || !entry.ListModels {
			t.Fatalf("%s should expose the OpenAI-compatible adapter surface: %+v", id, entry)
		}
		if entry.PublicInference || entry.DirectDispatch || entry.ModelCatalog || entry.Quota {
			t.Fatalf("%s should not claim public routing, fake catalog, or quota: %+v", id, entry)
		}
	}
}

func TestGeminiPublicNativeProviderStreams(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("gemini")
	if !ok {
		t.Fatal("provider matrix missing gemini")
	}
	if entry.PublicStatus != ProviderStatusSupported {
		t.Fatalf("gemini status = %q, want supported", entry.PublicStatus)
	}
	if !entry.PublicInference || !entry.DirectDispatch || !entry.RegisteredAdapter || !entry.Inference {
		t.Fatalf("gemini supported surface is incomplete: %+v", entry)
	}
	if !entry.Streaming || !entry.ModelCatalog || !entry.ListModels {
		t.Fatalf("gemini should expose streaming, catalog, and ListModels: %+v", entry)
	}
	if entry.Quota {
		t.Fatal("gemini should not claim quota support until a real quota fetcher exists")
	}
}

func TestVertexPublicNativeProviderStreams(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("vertex")
	if !ok {
		t.Fatal("provider matrix missing vertex")
	}
	if entry.PublicStatus != ProviderStatusSupported {
		t.Fatalf("vertex status = %q, want supported", entry.PublicStatus)
	}
	if !entry.PublicInference || !entry.DirectDispatch || !entry.RegisteredAdapter || !entry.Inference {
		t.Fatalf("vertex supported surface is incomplete: %+v", entry)
	}
	if !entry.Streaming || !entry.ModelCatalog || !entry.ListModels {
		t.Fatalf("vertex should expose streaming, catalog, and ListModels: %+v", entry)
	}
	if entry.Quota {
		t.Fatal("vertex should not claim quota support until a real quota fetcher exists")
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
