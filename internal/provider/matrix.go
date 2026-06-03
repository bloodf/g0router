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
	OMPID             string
	Router9ID         string
	BifrostID         string
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
		adapterOnlyProvider("azure", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("bedrock", "", false, false, false, false, "api_key"),
		adapterOnlyProvider("cerebras", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("cohere", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("deepseek", "deepseek", true, true, true, false, "api_key", "oauth"),
		adapterOnlyProvider("fireworks", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("gemini", "gemini", true, false, true, false, "api_key", "oauth"),
		adapterOnlyProvider("groq", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("huggingface", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("mistral", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("nebius", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("nvidia", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("ollama", "", false, true, true, false, "noauth"),
		adapterOnlyProvider("openrouter", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("perplexity", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("replicate", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("together", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("vertex", "gemini", true, false, true, false, "oauth"),
		authOnlyProvider("antigravity", "antigravity", true, "Google OAuth credential flow; runtime dispatch is through Gemini/Vertex adapters."),
		authOnlyProvider("github-copilot", "github-copilot", true, "OAuth is implemented, but no GitHub Copilot inference adapter is wired."),
		authOnlyProvider("cursor", "cursor", true, "OAuth is implemented, but no Cursor inference adapter is wired."),
		authOnlyProvider("gitlab", "gitlab", true, "OAuth is implemented for GitLab-style identity, but no GitLab inference adapter is wired."),
		authOnlyProvider("kimi", "kimi", true, "Device-code OAuth is implemented, but no Moonshot/Kimi inference adapter is wired."),
		authOnlyProvider("kiro", "kiro", true, "OAuth is implemented, but no Kiro inference adapter is wired."),
		authOnlyProvider("xai", "xai", true, "OAuth is implemented, but no xAI inference adapter is wired."),
		authOnlyProvider("xiaomi", "xiaomi", true, "OAuth is implemented, but no Xiaomi inference adapter is wired."),
		apiKeyAuthOnlyProvider("alibaba", "Direct API-key capture is implemented, but no Alibaba inference adapter is wired."),
		apiKeyAuthOnlyProvider("minimax", "API-key capture is implemented, but no MiniMax inference adapter is wired."),
		apiKeyAuthOnlyProvider("zhipu", "Direct API-key capture is implemented, but no ZAI/Zhipu inference adapter is wired."),
		unsupportedProvider("cloudflare-ai-gateway", "No gateway adapter is implemented."),
		unsupportedProvider("kagi", "No Kagi tool/search provider integration is implemented."),
		unsupportedProvider("kilo", "No Kilo provider integration is implemented."),
		unsupportedProvider("litellm", "No self-hosted gateway adapter is implemented."),
		unsupportedProvider("lm-studio", "No local LM Studio adapter is implemented."),
		unsupportedProvider("ollama-cloud", "Only local Ollama is implemented."),
		unsupportedProvider("opencode", "No OpenCode provider integration is implemented."),
		unsupportedProvider("qianfan", "No Baidu Qianfan auth or inference adapter is implemented."),
		unsupportedProvider("qwen", "No OAuth, inference adapter, or model catalog is implemented."),
		unsupportedProvider("tavily", "No Tavily tool/search provider integration is implemented."),
		unsupportedProvider("vercel-ai-gateway", "No Vercel AI Gateway adapter is implemented."),
		unsupportedProvider("vllm", "No configurable OpenAI-compatible self-hosted adapter is implemented."),
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
	entry.Quota = true
	entry.PublicStatus = ProviderStatusSupported
	if id == "openai" {
		entry.OMPID = "openai/codex"
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
	if id == "bedrock" {
		entry.Inference = false
		entry.Notes = "Adapter is registered, but it does not implement Bedrock Converse, streaming, model catalog/ListModels, quota, or public direct dispatch."
	}
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
		OMPID:        id,
		Router9ID:    id,
		BifrostID:    id,
		G0RouterID:   id,
		PublicStatus: ProviderStatusUnsupported,
	}
}
