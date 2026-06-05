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
		"gitlab-duo",
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

func TestProviderMatrixMarksOAuthOnlyProvidersExplicitly(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"cursor", "kiro"} {
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

func TestGitLabDuoPublicDynamicProvider(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("gitlab-duo")
	if !ok {
		t.Fatal("provider matrix missing gitlab-duo")
	}
	if entry.PublicStatus != ProviderStatusSupported {
		t.Fatalf("gitlab-duo status = %q, want supported", entry.PublicStatus)
	}
	if !entry.RegisteredAdapter || !entry.Inference || !entry.Streaming || !entry.PublicInference || !entry.DirectDispatch {
		t.Fatalf("gitlab-duo should expose public dynamic runtime support: %+v", entry)
	}
	if entry.ModelCatalog || !entry.ListModels || entry.Quota {
		t.Fatalf("gitlab-duo should claim alias listing, but no static catalog or quota: %+v", entry)
	}
	if entry.OAuthProvider != "gitlab-duo" || len(entry.AuthTypes) != 1 || entry.AuthTypes[0] != "oauth" {
		t.Fatalf("gitlab-duo auth metadata = %+v, want oauth through gitlab-duo", entry)
	}
}

func TestProviderMatrixMarksSearchCredentialsAuthOnly(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"kagi", "tavily"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusAuthOnly {
			t.Fatalf("%s status = %q, want auth_only", id, entry.PublicStatus)
		}
		if len(entry.AuthTypes) != 1 || entry.AuthTypes[0] != "api_key" {
			t.Fatalf("%s auth types = %+v, want api_key only", id, entry.AuthTypes)
		}
		if entry.RegisteredAdapter || entry.Inference || entry.PublicInference || entry.DirectDispatch || entry.Streaming || entry.ModelCatalog || entry.ListModels || entry.Quota {
			t.Fatalf("%s should be credential-only until web-search runtime is implemented: %+v", id, entry)
		}
		if !strings.Contains(strings.ToLower(entry.Notes), "search") || !strings.Contains(strings.ToLower(entry.Notes), "runtime") {
			t.Fatalf("%s notes = %q, want search runtime caveat", id, entry.Notes)
		}
	}
}

func TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"alibaba", "azure", "cloudflare-ai-gateway", "github-copilot", "gitlab-duo", "kilo", "kimi", "litellm", "lm-studio", "ollama-cloud", "opencode", "qianfan", "replicate", "vllm", "xiaomi", "zhipu"} {
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

func TestProviderMatrixKeepsKiroAuthOnlyAndKiloSupportedDistinct(t *testing.T) {
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
	if kilo.PublicStatus != ProviderStatusSupported {
		t.Fatalf("kilo status = %q, want supported", kilo.PublicStatus)
	}
	if !kilo.RegisteredAdapter || !kilo.PublicInference || !kilo.DirectDispatch || !kilo.Inference || !kilo.Streaming {
		t.Fatalf("kilo supported surface is incomplete: %+v", kilo)
	}
	if kilo.ModelCatalog || kilo.ListModels || kilo.Quota {
		t.Fatalf("kilo should not claim catalog, model listing, or quota support: %+v", kilo)
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
		"gitlab-duo":            true,
		"groq":                  true,
		"huggingface":           true,
		"kilo":                  true,
		"kimi":                  true,
		"litellm":               true,
		"lm-studio":             true,
		"mistral":               true,
		"minimax":               true,
		"nebius":                true,
		"nvidia":                true,
		"ollama":                true,
		"ollama-cloud":          true,
		"opencode":              true,
		"openrouter":            true,
		"perplexity":            true,
		"qianfan":               true,
		"qwen":                  true,
		"replicate":             true,
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
	for _, id := range []string{"alibaba", "anthropic", "azure", "bedrock", "cerebras", "cloudflare-ai-gateway", "cohere", "deepseek", "fireworks", "github-copilot", "groq", "huggingface", "kilo", "kimi", "litellm", "lm-studio", "mistral", "minimax", "nebius", "nvidia", "ollama", "ollama-cloud", "opencode", "openai", "openrouter", "perplexity", "qianfan", "qwen", "together", "vercel-ai-gateway", "vllm", "xai", "xiaomi", "zhipu"} {
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
		if id == "alibaba" || id == "azure" || id == "cloudflare-ai-gateway" || id == "github-copilot" || id == "kilo" || id == "kimi" || id == "litellm" || id == "lm-studio" || id == "ollama-cloud" || id == "opencode" || id == "qianfan" || id == "vllm" || id == "xiaomi" || id == "zhipu" {
			if entry.ModelCatalog {
				t.Fatalf("%s should not claim static model catalog for deployment-defined routing: %+v", id, entry)
			}
			if id == "cloudflare-ai-gateway" || id == "kilo" || id == "opencode" || id == "xiaomi" {
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

func TestOllamaCloudPublicNativeProvider(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("ollama-cloud")
	if !ok {
		t.Fatal("provider matrix missing ollama-cloud")
	}
	if entry.PublicStatus != ProviderStatusSupported {
		t.Fatalf("ollama-cloud status = %q, want supported", entry.PublicStatus)
	}
	if len(entry.AuthTypes) != 1 || entry.AuthTypes[0] != "api_key" {
		t.Fatalf("ollama-cloud auth types = %+v, want api_key only", entry.AuthTypes)
	}
	if !entry.RegisteredAdapter || !entry.Inference || !entry.PublicInference || !entry.DirectDispatch || !entry.Streaming || !entry.ListModels {
		t.Fatalf("ollama-cloud supported surface is incomplete: %+v", entry)
	}
	if entry.ModelCatalog || entry.Quota {
		t.Fatalf("ollama-cloud should not claim static catalog or quota: %+v", entry)
	}
	note := strings.ToLower(entry.Notes)
	if !strings.Contains(note, "native ollama") || !strings.Contains(note, "provider-qualified") {
		t.Fatalf("ollama-cloud notes = %q, want native/provider-qualified caveat", entry.Notes)
	}
}

func TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs(t *testing.T) {
	matrix := ProviderMatrix()
	for _, id := range []string{"alibaba", "cloudflare-ai-gateway", "github-copilot", "kilo", "kimi", "litellm", "lm-studio", "opencode", "qianfan", "vllm", "zhipu"} {
		entry, ok := matrix.Provider(id)
		if !ok {
			t.Fatalf("provider %q missing", id)
		}
		if entry.PublicStatus != ProviderStatusSupported {
			t.Fatalf("%s status = %q, want supported", id, entry.PublicStatus)
		}
		if !entry.RegisteredAdapter || !entry.Inference {
			t.Fatalf("%s should expose a registered inference adapter surface: %+v", id, entry)
		}
		if id != "replicate" && !entry.Streaming {
			t.Fatalf("%s should expose streaming: %+v", id, entry)
		}
		if id != "cloudflare-ai-gateway" && id != "kilo" && id != "opencode" && id != "replicate" && !entry.ListModels {
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

func TestReplicatePromotesToPredictionBackedInferenceProvider(t *testing.T) {
	entry, ok := ProviderMatrix().Provider("replicate")
	if !ok {
		t.Fatal("provider matrix missing replicate")
	}
	if entry.PublicStatus != ProviderStatusSupported {
		t.Fatalf("replicate status = %q, want supported", entry.PublicStatus)
	}
	if len(entry.AuthTypes) != 1 || entry.AuthTypes[0] != "api_key" {
		t.Fatalf("replicate auth types = %+v, want api_key only", entry.AuthTypes)
	}
	if !entry.RegisteredAdapter || !entry.Inference || !entry.PublicInference || !entry.DirectDispatch {
		t.Fatalf("replicate should claim prediction-backed provider-qualified runtime support: %+v", entry)
	}
	if entry.Streaming || entry.ListModels || entry.ModelCatalog || entry.Quota {
		t.Fatalf("replicate should not claim streaming, listing, catalog, or quota: %+v", entry)
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
