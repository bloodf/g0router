package provider

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/providers/bedrock"
	"github.com/bloodf/g0router/internal/providers/cloudflare"
	"github.com/bloodf/g0router/internal/providers/replicate"
)

// streamingAdapter is the minimal surface needed to check whether an adapter's
// advertised streaming capability matches its runtime behaviour.
type streamingAdapter interface {
	ChatCompletionStream(context.Context, providers.Key, *providers.ChatRequest) (<-chan providers.StreamChunk, error)
}

// TestProviderMatrixStreamingFlagMatchesAdapterBehaviour enforces honesty: an
// adapter whose matrix entry advertises Streaming=false must reject streaming
// with the ErrStreamingUnsupported sentinel, and an adapter advertising
// Streaming=true must not return that sentinel (cloudflare instead surfaces a
// clear configuration error when account_id is absent). This keeps the
// advertised capability and the actual adapter wiring from drifting apart.
func TestProviderMatrixStreamingFlagMatchesAdapterBehaviour(t *testing.T) {
	matrix := ProviderMatrix()
	req := &providers.ChatRequest{Model: "m", Messages: []providers.Message{{Role: "user", Content: "hi"}}}

	cases := []struct {
		id      string
		adapter streamingAdapter
	}{
		{id: "bedrock", adapter: bedrock.New("")},
		{id: "replicate", adapter: replicate.NewDefault()},
		{id: "cloudflare-ai-gateway", adapter: cloudflare.New("")},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			entry, ok := matrix.Provider(tc.id)
			if !ok {
				t.Fatalf("provider matrix missing %q", tc.id)
			}
			_, err := tc.adapter.ChatCompletionStream(context.Background(), providers.Key{}, req)
			if err == nil {
				t.Fatalf("%s ChatCompletionStream returned no error in offline test; cannot verify capability honesty", tc.id)
			}
			unsupported := errors.Is(err, providers.ErrStreamingUnsupported)
			if entry.Streaming && unsupported {
				t.Fatalf("%s advertises Streaming=true but adapter returns ErrStreamingUnsupported: %v", tc.id, err)
			}
			if !entry.Streaming && !unsupported {
				t.Fatalf("%s advertises Streaming=false but adapter does not return ErrStreamingUnsupported: %v", tc.id, err)
			}
		})
	}
}

// TestCloudflareRequiresAccountIDConfiguration verifies the cloudflare adapter
// fails fast with a clear configuration error rather than dispatching dead
// requests when the connection lacks an account_id.
func TestCloudflareRequiresAccountIDConfiguration(t *testing.T) {
	provider := cloudflare.New("")
	req := &providers.ChatRequest{Model: "m", Messages: []providers.Message{{Role: "user", Content: "hi"}}}
	_, err := provider.ChatCompletion(context.Background(), providers.Key{}, req)
	if err == nil {
		t.Fatal("cloudflare ChatCompletion without account_id should error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "account id") {
		t.Fatalf("cloudflare error = %v, want clear account id configuration message", err)
	}
}

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
			t.Fatalf("%s should stay auth-only outside built-in MCP search tools: %+v", id, entry)
		}
		notes := strings.ToLower(entry.Notes)
		if !strings.Contains(notes, "__search") || !strings.Contains(notes, "no inference adapter") {
			t.Fatalf("%s notes = %q, want built-in search tool plus no-inference caveat", id, entry.Notes)
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

func TestProviderMatrixMarksBedrockPublicStreamingAfterConverseSupport(t *testing.T) {
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
	if !entry.Streaming {
		t.Fatalf("bedrock should advertise native ConverseStream streaming: %+v", entry)
	}
	if entry.Quota {
		t.Fatalf("bedrock quota capability should stay false: %+v", entry)
	}
	if !entry.Inference {
		t.Fatalf("bedrock should expose Converse inference: %+v", entry)
	}
	if !entry.ListModels {
		t.Fatalf("bedrock should expose signed foundation model listing: %+v", entry)
	}
	note := strings.ToLower(entry.Notes)
	if !strings.Contains(note, "converse") || !strings.Contains(note, "catalog") || !strings.Contains(note, "streaming") {
		t.Fatalf("bedrock notes = %q, want explicit Converse catalog streaming status", entry.Notes)
	}
}

func TestProviderMatrixUnknownID(t *testing.T) {
	matrix := ProviderMatrix()
	_, ok := matrix.Provider("nonexistent-provider-xyz")
	if ok {
		t.Fatal("expected false for unknown provider id")
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

func TestPublicProvidersOnlyClaimImplementedQuotaSupport(t *testing.T) {
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
		if !entry.Streaming {
			t.Fatalf("%s should expose streaming: %+v", id, entry)
		}
		if id == "openrouter" {
			if !entry.Quota {
				t.Fatalf("%s should claim quota support after real fetcher implementation", id)
			}
		} else if entry.Quota {
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
	if !entry.Streaming {
		t.Fatalf("replicate should advertise native SSE token streaming: %+v", entry)
	}
	if entry.ListModels || entry.ModelCatalog || entry.Quota {
		t.Fatalf("replicate should not claim listing, catalog, or quota: %+v", entry)
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

func TestProviderDocsQuotaColumnMatchesMatrix(t *testing.T) {
	content, err := os.ReadFile("../../docs/PROVIDERS.md")
	if err != nil {
		t.Fatalf("read docs/PROVIDERS.md: %v", err)
	}
	text := string(content)
	header := "| g0router ID | Status | Auth | Refresh | Adapter | Public inference | Streaming | Catalog | List models | Quota | Notes |"
	if !strings.Contains(text, header) {
		t.Fatalf("provider docs matrix header is missing or changed; want:\n%s", header)
	}

	matrix := ProviderMatrix()
	for _, entry := range matrix.Entries() {
		want := "no"
		if entry.Quota {
			want = "yes"
		}
		rowPrefix := "| `" + entry.G0RouterID + "` |"
		row := lineWithPrefix(text, rowPrefix)
		if row == "" {
			t.Fatalf("provider docs missing row for %s", entry.G0RouterID)
		}
		cols := strings.Split(row, "|")
		// cols[0] = "", cols[1] = " `id` ", cols[2]=Status, cols[3]=Auth, cols[4]=Refresh,
		// cols[5]=Adapter, cols[6]=Public inference, cols[7]=Streaming, cols[8]=Catalog,
		// cols[9]=List models, cols[10]=Quota, cols[11]=Notes, cols[12]=""
		if len(cols) < 11 {
			t.Fatalf("provider docs row for %s has too few columns: %q", entry.G0RouterID, row)
		}
		got := strings.TrimSpace(cols[10])
		if got != want {
			t.Fatalf("provider docs quota for %s = %q, matrix says %v (want %q)", entry.G0RouterID, got, entry.Quota, want)
		}
	}
}

func lineWithPrefix(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, prefix) {
			return line
		}
	}
	return ""
}

func providerIDs(entries []ProviderMatrixEntry) map[string]bool {
	result := make(map[string]bool, len(entries))
	for _, entry := range entries {
		result[entry.G0RouterID] = true
	}
	return result
}
