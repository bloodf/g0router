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
		catalogRoutableProvider("openrouter", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("perplexity", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("replicate", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("together", "", false, true, true, false, "api_key"),
		catalogRoutableProvider("vertex", "gemini", true, false, true, false, "oauth"),
		authOnlyProvider("antigravity", "antigravity", true, "Google OAuth credential flow; runtime dispatch is through Gemini/Vertex adapters."),
		authOnlyProvider("github-copilot", "github-copilot", true, "OAuth is implemented, but no GitHub Copilot inference adapter is wired."),
		authOnlyProvider("cursor", "cursor", true, "OAuth is implemented, but no Cursor inference adapter is wired."),
		authOnlyProvider("gitlab", "gitlab", true, "OAuth is implemented for GitLab-style identity, but no GitLab inference adapter is wired."),
		authOnlyProvider("kimi", "kimi", true, "Device-code OAuth is implemented, but no Moonshot/Kimi inference adapter is wired."),
		authOnlyProvider("kiro", "kiro", true, "OAuth is implemented, but no Kiro inference adapter is wired."),
		catalogRoutableProvider("xai", "xai", true, true, true, false, "api_key", "oauth"),
		authOnlyProvider("xiaomi", "xiaomi", true, "OAuth is implemented, but no Xiaomi inference adapter is wired."),
		apiKeyAuthOnlyProvider("alibaba", "Direct API-key capture is implemented, but no Alibaba inference adapter is wired."),
		catalogRoutableProvider("minimax", "", false, true, true, false, "api_key"),
		apiKeyAuthOnlyProvider("zhipu", "Direct API-key capture is implemented, but no ZAI/Zhipu inference adapter is wired."),
		unsupportedProvider("cloudflare-ai-gateway", "No gateway adapter is implemented."),
		unsupportedProvider("kagi", "No Kagi tool/search provider integration is implemented."),
		unsupportedProvider("kilo", "No Kilo provider integration is implemented."),
		adapterOnlyProvider("litellm", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("lm-studio", "", false, true, true, false, "api_key"),
		unsupportedProvider("ollama-cloud", "Only local Ollama is implemented."),
		unsupportedProvider("opencode", "No OpenCode provider integration is implemented."),
		unsupportedProvider("qianfan", "No Baidu Qianfan auth or inference adapter is implemented."),
		catalogRoutableProvider("qwen", "", false, true, true, false, "api_key"),
		unsupportedProvider("tavily", "No Tavily tool/search provider integration is implemented."),
		catalogRoutableProvider("vercel-ai-gateway", "", false, true, true, false, "api_key"),
		adapterOnlyProvider("vllm", "", false, true, true, false, "api_key"),
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
		entry.OMPID = "openai/codex"
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
