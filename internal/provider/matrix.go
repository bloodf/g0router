package provider

import (
	"sort"
	"strings"
)

type ProviderStatus string

const (
	ProviderStatusSupported   ProviderStatus = "supported"
	ProviderStatusAdapterOnly ProviderStatus = "adapter_only"
	ProviderStatusAuthOnly    ProviderStatus = "auth_only"
	ProviderStatusUnsupported ProviderStatus = "unsupported"
)

type ProviderMatrixEntry struct {
	G0RouterID        string
	AuthTypes         []string
	OAuthProvider     string
	Refresh           bool
	RegisteredAdapter bool
	PublicInference   bool
	DirectDispatch    bool
	Inference         bool
	Streaming         bool
	ModelCatalog      bool
	ListModels        bool
	Quota             bool
	PublicStatus      ProviderStatus
	Notes             string
}

type ProviderMatrixTable struct {
	entries []ProviderMatrixEntry
}

func ProviderMatrix() ProviderMatrixTable {
	entries := []ProviderMatrixEntry{
		supportedProvider("openai", "codex", true, true, true, "api_key", "oauth"),
		supportedProvider("anthropic", "anthropic", true, true, true, "api_key", "oauth"),
		dynamicRoutableProvider("azure", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("bedrock", "", false, false, true, false, "api_key"),
		catalogRoutableProvider("cerebras", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("cohere", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("deepseek", "deepseek", true, true, true, false, "api_key", "oauth"),
		catalogRoutableProvider("fireworks", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("gemini", "gemini", true, true, true, false, "api_key", "oauth"),
		catalogRoutableProvider("groq", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("huggingface", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("mistral", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("nebius", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("nvidia", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("ollama", "", false, true, true, false, "noauth"),
		catalogRoutableProvider("openrouter", "", false, true, true, true, "api_key"),
		catalogRoutableProvider("perplexity", "", false, true, true, false, "api_key"),
		replicateProvider(),
		catalogRoutableProvider("together", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("vertex", "gemini", true, true, true, false, "oauth"),
		authOnlyProvider("antigravity", "antigravity", true, "Google OAuth credential flow; runtime dispatch is through Gemini/Vertex adapters."),
		dynamicRoutableProvider("github-copilot", "github-copilot", true, true, true, false, "oauth"),
		authOnlyProvider("cursor", "cursor", true, "loginDeepControl polling OAuth is implemented, but no Cursor inference adapter is wired."),
		gitLabDuoProvider(),
		dynamicRoutableProvider("kimi", "kimi", true, true, true, false, "api_key", "oauth"),
		authOnlyProvider("kiro", "kiro", true, "OAuth is implemented, but no Kiro inference adapter is wired."),
		catalogRoutableProvider("xai", "xai", true, true, true, false, "api_key", "oauth"),
		xiaomiProvider(),
		dynamicRoutableProvider("alibaba", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("minimax", "", false, true, true, false, "api_key"),
		dynamicRoutableProvider("zhipu", "", false, true, true, false, "api_key"),
		cloudflareGatewayProvider(),
		apiKeyAuthOnlyProvider("kagi", "Kagi API-key credentials can back the built-in kagi__search MCP tool; no inference adapter, public dispatch, catalog, streaming, pricing, or quota support is implemented."),
		kiloProvider(),
		dynamicRoutableProvider("litellm", "", false, true, true, false, "api_key"),
		dynamicRoutableProvider("lm-studio", "", false, true, true, false, "api_key"),
		ollamaCloudProvider(),
		opencodeProvider(),
		dynamicRoutableProvider("qianfan", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("qwen", "", false, true, true, false, "api_key"),
		apiKeyAuthOnlyProvider("tavily", "Tavily API-key credentials can back the built-in tavily__search MCP tool; no inference adapter, public dispatch, catalog, streaming, pricing, or quota support is implemented."),
		catalogRoutableProvider("vercel-ai-gateway", "", false, true, true, false, "api_key"),
		dynamicRoutableProvider("vllm", "", false, true, true, false, "api_key"),
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].G0RouterID < entries[j].G0RouterID
	})
	return ProviderMatrixTable{entries: entries}
}

func (m ProviderMatrixTable) Entries() []ProviderMatrixEntry {
	entries := make([]ProviderMatrixEntry, len(m.entries))
	copy(entries, m.entries)
	return entries
}

func (m ProviderMatrixTable) Provider(id string) (ProviderMatrixEntry, bool) {
	id = CanonicalProviderID(id)
	for _, entry := range m.entries {
		if entry.G0RouterID == id {
			return entry, true
		}
	}
	return ProviderMatrixEntry{}, false
}

func PublicInferenceProviders() []ProviderMatrixEntry {
	var entries []ProviderMatrixEntry
	for _, entry := range ProviderMatrix().Entries() {
		if entry.PublicStatus == ProviderStatusSupported && entry.PublicInference && entry.DirectDispatch {
			entries = append(entries, entry)
		}
	}
	return entries
}

func supportedProvider(id string, oauthProvider string, refresh, streaming, modelCatalog bool, authTypes ...string) ProviderMatrixEntry {
	entry := baseProvider(id)
	entry.AuthTypes = authTypes
	entry.OAuthProvider = oauthProvider
	entry.Refresh = refresh
	entry.RegisteredAdapter = true
	entry.PublicInference = true
	entry.DirectDispatch = true
	entry.Inference = true
	entry.Streaming = streaming
	entry.ModelCatalog = modelCatalog
	entry.ListModels = modelCatalog
	entry.Quota = false
	entry.PublicStatus = ProviderStatusSupported
	entry.Notes = "Public direct dispatch works through native routing; quota fetcher is not implemented yet."
	if id == "openai" {
}
	return entry
}

func catalogRoutableProvider(id string, oauthProvider string, refresh, streaming, modelCatalog, quota bool, authTypes ...string) ProviderMatrixEntry {
	entry := baseProvider(id)
	entry.AuthTypes = authTypes
	entry.OAuthProvider = oauthProvider
	entry.Refresh = refresh
	entry.RegisteredAdapter = true
	entry.PublicInference = true
	entry.DirectDispatch = true
	entry.Inference = true
	entry.Streaming = streaming
	entry.ModelCatalog = modelCatalog
	entry.ListModels = modelCatalog
	entry.Quota = quota
	entry.PublicStatus = ProviderStatusSupported
	if !quota {
		entry.Notes = "Public direct dispatch works through catalog routing; quota fetcher is not implemented yet."
	}
	if id == "bedrock" {
		entry.Notes = "Public direct dispatch works through catalog-backed non-streaming Bedrock Converse routing; streaming and quota are not implemented yet."
	}
	if id == "openrouter" {
		entry.Notes = "Public direct dispatch works through catalog routing; quota fetcher uses OpenRouter's current API key credits endpoint."
	}
	return entry
}

func adapterOnlyProvider(id string, oauthProvider string, refresh, streaming, listModels, quota bool, authTypes ...string) ProviderMatrixEntry {
	entry := baseProvider(id)
	entry.AuthTypes = authTypes
	entry.OAuthProvider = oauthProvider
	entry.Refresh = refresh
	entry.RegisteredAdapter = true
	entry.Inference = true
	entry.Streaming = streaming
	entry.ListModels = listModels
	entry.Quota = quota
	entry.PublicStatus = ProviderStatusAdapterOnly
	entry.Notes = "Adapter is registered in normal startup, but public routing remains limited by model catalog and capability coverage."
	return entry
}

func dynamicRoutableProvider(id string, oauthProvider string, refresh, streaming, listModels, quota bool, authTypes ...string) ProviderMatrixEntry {
	entry := adapterOnlyProvider(id, oauthProvider, refresh, streaming, listModels, quota, authTypes...)
	entry.PublicInference = true
	entry.DirectDispatch = true
	entry.PublicStatus = ProviderStatusSupported
	entry.Notes = "Public direct dispatch works through provider-qualified dynamic model IDs; no static model catalog or quota fetcher is implemented."
	return entry
}

func cloudflareGatewayProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("cloudflare-ai-gateway", "", false, true, false, false, "api_key")
	entry.Notes = "Public direct dispatch works through provider-qualified Cloudflare REST API model IDs when the stored connection includes account_id; no static model catalog or quota fetcher is implemented."
	return entry
}

func xiaomiProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("xiaomi", "xiaomi", true, true, false, false, "api_key", "oauth")
	entry.Notes = "Public direct dispatch works through provider-qualified Xiaomi Anthropic-compatible model IDs; token-plan keys use the token-plan endpoint, and no static model catalog or quota fetcher is implemented."
	return entry
}

func opencodeProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("opencode", "", false, true, false, false, "api_key")
	entry.Notes = "OpenCode Zen is supported through provider-qualified dynamic model IDs; OpenCode Go is not separately wired yet, and no static model catalog or quota fetcher is implemented."
	return entry
}

func kiloProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("kilo", "", false, true, false, false, "api_key")
	entry.Notes = "Kilo Gateway is supported through provider-qualified OpenAI-compatible dynamic model IDs; no static model catalog, model listing, or quota fetcher is implemented."
	return entry
}

func ollamaCloudProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("ollama-cloud", "", false, true, true, false, "api_key")
	entry.Notes = "Public direct dispatch works through provider-qualified native Ollama Cloud model IDs; native /api/tags model listing is supported, with no static catalog or quota fetcher."
	return entry
}

func gitLabDuoProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("gitlab-duo", "gitlab-duo", true, true, true, false, "oauth")
	entry.Notes = "Public direct dispatch works through provider-qualified GitLab Duo model IDs after exchanging the GitLab OAuth token for a Duo direct-access token; Duo alias model listing is supported, with no static priced catalog or quota fetcher."
	return entry
}

func replicateProvider() ProviderMatrixEntry {
	entry := dynamicRoutableProvider("replicate", "", false, false, false, false, "api_key")
	entry.Notes = "Public non-streaming direct dispatch works through provider-qualified Replicate prediction model IDs; streaming, model listing, static catalog, and quota fetcher are not implemented."
	return entry
}

func authOnlyProvider(id string, oauthProvider string, refresh bool, notes string) ProviderMatrixEntry {
	entry := baseProvider(id)
	entry.AuthTypes = []string{"oauth"}
	entry.OAuthProvider = oauthProvider
	entry.Refresh = refresh
	entry.PublicStatus = ProviderStatusAuthOnly
	entry.Notes = notes
	return entry
}

func apiKeyAuthOnlyProvider(id string, notes string) ProviderMatrixEntry {
	entry := baseProvider(id)
	entry.AuthTypes = []string{"api_key"}
	entry.PublicStatus = ProviderStatusAuthOnly
	entry.Notes = notes
	return entry
}

func unsupportedProvider(id string, notes string) ProviderMatrixEntry {
	entry := baseProvider(id)
	entry.PublicStatus = ProviderStatusUnsupported
	entry.Notes = notes
	return entry
}

func baseProvider(id string) ProviderMatrixEntry {
	id = strings.ToLower(strings.TrimSpace(id))
	return ProviderMatrixEntry{
		G0RouterID:   id,
		PublicStatus: ProviderStatusUnsupported,
	}
}
