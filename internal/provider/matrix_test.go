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
	for _, id := range []string{"cursor", "gitlab", "kiro"} {
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

func TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"alibaba", "azure", "cloudflare-ai-gateway", "github-copilot", "kimi", "litellm", "lm-studio", "opencode", "qianfan", "vllm", "xiaomi", "zhipu"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusSupported {
			t.Fatalf("%s status = %q, want supported", id, entry.PublicStatus)
		}
		if !entry.RegisteredAdapter {
			t.Fatalf("%s should mark registered adapter", id)
		}
		if !entry.Inference {
			t.Fatalf("%s should mark adapter inference capability", id)
		}
		if !entry.PublicInference || !entry.DirectDispatch {
			t.Fatalf("%s should mark provider-qualified public routing: %+v", id, entry)
		}
		if entry.ModelCatalog || entry.Quota {
			t.Fatalf("%s should not claim fake catalog or quota support: %+v", id, entry)
		}
		if !strings.Contains(strings.ToLower(entry.Notes), "provider-qualified") {
			t.Fatalf("%s notes = %q, want provider-qualified routing caveat", id, entry.Notes)
		}
	}
}

func TestProviderMatrixMarksBedrockPublicNonStreamingAfterConverseSupport(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("bedrock")
	if !ok {
		t.Fatal("provider matrix missing bedrock")
	}
	if entry.PublicStatus != ProviderStatusSupported {
		t.Fatalf("bedrock status = %q, want supported", entry.PublicStatus)
	}
	if !entry.RegisteredAdapter {
		t.Fatal("bedrock should mark registered adapter")
	}
	if !entry.PublicInference || !entry.DirectDispatch || !entry.ModelCatalog {
		t.Fatalf("bedrock should expose catalog-backed public direct dispatch: %+v", entry)
	}
	if entry.Streaming || entry.Quota {
		t.Fatalf("bedrock streaming/quota capabilities should stay false: %+v", entry)
	}
	if !entry.Inference {
		t.Fatalf("bedrock should expose adapter-only non-streaming Converse inference: %+v", entry)
	}
	if !entry.ListModels {
		t.Fatalf("bedrock should expose signed foundation model listing: %+v", entry)
	}
	note := strings.ToLower(entry.Notes)
	if !strings.Contains(note, "converse") || !strings.Contains(note, "catalog") || !strings.Contains(note, "non-streaming") {
		t.Fatalf("bedrock notes = %q, want explicit Converse catalog status", entry.Notes)
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
		"openai":                true,
		"anthropic":             true,
		"alibaba":               true,
		"azure":                 true,
		"bedrock":               true,
		"cerebras":              true,
		"cloudflare-ai-gateway": true,
		"cohere":                true,
		"deepseek":              true,
		"fireworks":             true,
		"gemini":                true,
		"github-copilot":        true,
		"groq":                  true,
		"huggingface":           true,
		"kimi":                  true,
		"litellm":               true,
		"lm-studio":             true,
		"mistral":               true,
		"minimax":               true,
		"nebius":                true,
		"nvidia":                true,
		"ollama":                true,
		"opencode":              true,
		"openrouter":            true,
		"perplexity":            true,
		"qianfan":               true,
		"qwen":                  true,
		"together":              true,
		"vercel-ai-gateway":     true,
		"vertex":                true,
		"vllm":                  true,
		"xai":                   true,
		"xiaomi":                true,
		"zhipu":                 true,
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

	if ids["cursor"] {
		t.Fatal("cursor is auth-only today and must not be advertised as an inference provider")
	}
}

func TestPublicProvidersDoNotClaimQuotaSupport(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"alibaba", "anthropic", "azure", "bedrock", "cerebras", "cloudflare-ai-gateway", "cohere", "deepseek", "fireworks", "github-copilot", "groq", "huggingface", "kimi", "litellm", "lm-studio", "mistral", "minimax", "nebius", "nvidia", "ollama", "opencode", "openai", "openrouter", "perplexity", "qianfan", "qwen", "together", "vercel-ai-gateway", "vllm", "xai", "xiaomi", "zhipu"} {
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
		if id == "alibaba" || id == "azure" || id == "cloudflare-ai-gateway" || id == "github-copilot" || id == "kimi" || id == "litellm" || id == "lm-studio" || id == "opencode" || id == "qianfan" || id == "vllm" || id == "xiaomi" || id == "zhipu" {
			if entry.ModelCatalog {
				t.Fatalf("%s should not claim static model catalog for deployment-defined routing: %+v", id, entry)
			}
			if id == "cloudflare-ai-gateway" || id == "opencode" || id == "xiaomi" {
				if entry.ListModels {
					t.Fatalf("%s should not claim model listing until provider-specific model-list support is implemented: %+v", id, entry)
				}
			} else if !entry.ListModels {
				t.Fatalf("%s should expose upstream list models/deployments: %+v", id, entry)
			}
		} else if !entry.ModelCatalog || !entry.ListModels {
			t.Fatalf("%s should expose catalog and model APIs: %+v", id, entry)
		}
		if id == "bedrock" {
			if entry.Streaming {
				t.Fatalf("%s should not claim streaming until event-stream support exists: %+v", id, entry)
			}
		} else if !entry.Streaming {
			t.Fatalf("%s should expose streaming: %+v", id, entry)
		}
		if entry.Quota {
			t.Fatalf("%s should not claim quota support until a real quota fetcher exists", id)
		}
	}
}

func TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"alibaba", "cloudflare-ai-gateway", "github-copilot", "kimi", "litellm", "lm-studio", "opencode", "qianfan", "vllm", "zhipu"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusSupported {
			t.Fatalf("%s status = %q, want supported", id, entry.PublicStatus)
		}
		if !entry.RegisteredAdapter || !entry.Inference || !entry.Streaming {
			t.Fatalf("%s should expose the OpenAI-compatible adapter surface: %+v", id, entry)
		}
		if id != "cloudflare-ai-gateway" && id != "opencode" && !entry.ListModels {
			t.Fatalf("%s should expose upstream list models/deployments: %+v", id, entry)
		}
		if !entry.PublicInference || !entry.DirectDispatch {
			t.Fatalf("%s should expose provider-qualified public routing: %+v", id, entry)
		}
		if entry.ModelCatalog || entry.Quota {
			t.Fatalf("%s should not claim fake catalog or quota: %+v", id, entry)
		}
	}
}

func TestReplicateRemainsAdapterOnlyUntilPublicSemanticsAreProven(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("replicate")
	if !ok {
		t.Fatal("provider matrix missing replicate")
	}
	if entry.PublicStatus != ProviderStatusAdapterOnly {
		t.Fatalf("replicate status = %q, want adapter_only", entry.PublicStatus)
	}
	if !entry.RegisteredAdapter || !entry.Inference || !entry.Streaming || !entry.ListModels {
		t.Fatalf("replicate should keep registered adapter capabilities: %+v", entry)
	}
	if entry.PublicInference || entry.DirectDispatch || entry.ModelCatalog || entry.Quota {
		t.Fatalf("replicate should not claim public routing, fake catalog, or quota yet: %+v", entry)
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
