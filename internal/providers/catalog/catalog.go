package catalog

import "strings"

// ProviderConfig holds the static configuration for a single provider.
type ProviderConfig struct {
	Name       string
	BaseURL    string
	Format     string
	Headers    map[string]string
	AuthHeader string
	NoAuth     bool
	// Retry overrides the default per-status attempt count. The map key is the
	// HTTP status code and the value is the number of retry attempts for that
	// status. A value of zero disables retries for that status. Delays are taken
	// from the default retry configuration.
	Retry map[int]int
}

// RetryOverride returns the provider-specific retry attempt overrides.
func (p ProviderConfig) RetryOverride() map[int]int {
	return p.Retry
}

// Providers is the Go port of the reference PROVIDERS map
// (open-sse/config/providers.js:50-438). Only the 11 Stage-1 entries are
// included here; the rest are out of scope for this wave.
var Providers = map[string]ProviderConfig{
	"groq": {
		Name:    "groq",
		BaseURL: "https://api.groq.com/openai/v1/chat/completions",
		Format:  "openai",
	},
	"deepseek": {
		Name:    "deepseek",
		BaseURL: "https://api.deepseek.com/chat/completions",
		Format:  "openai",
	},
	"mistral": {
		Name:    "mistral",
		BaseURL: "https://api.mistral.ai/v1/chat/completions",
		Format:  "openai",
	},
	"cohere": {
		Name:    "cohere",
		BaseURL: "https://api.cohere.ai/v1/chat/completions",
		Format:  "openai",
	},
	"together": {
		Name:    "together",
		BaseURL: "https://api.together.xyz/v1/chat/completions",
		Format:  "openai",
	},
	"fireworks": {
		Name:    "fireworks",
		BaseURL: "https://api.fireworks.ai/inference/v1/chat/completions",
		Format:  "openai",
	},
	"openrouter": {
		Name:    "openrouter",
		BaseURL: "https://openrouter.ai/api/v1/chat/completions",
		Format:  "openai",
		Headers: map[string]string{
			"HTTP-Referer": "https://endpoint-proxy.local",
			"X-Title":      "Endpoint Proxy",
		},
	},
	"xai": {
		// providers.js:273-280 carries OAuth fields (clientId, tokenUrl, refreshUrl).
		// Stage-1 includes xai via its API-key (bearer) path only; OAuth is Wave-3.
		// Those fields are intentionally omitted from ProviderConfig here.
		Name:    "xai",
		BaseURL: "https://api.x.ai/v1/chat/completions",
		Format:  "openai",
	},
	"perplexity": {
		Name:    "perplexity",
		BaseURL: "https://api.perplexity.ai/chat/completions",
		Format:  "openai",
	},
	"kiro": {
		Name:    "kiro",
		BaseURL: "https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse",
		Format:  "kiro",
		Retry:   map[int]int{429: 2},
		Headers: map[string]string{
			"Content-Type":             "application/json",
			"Accept":                   "application/vnd.amazon.eventstream",
			"X-Amz-Target":             "AmazonCodeWhispererStreamingService.GenerateAssistantResponse",
			"User-Agent":               "AWS-SDK-JS/3.0.0 kiro-ide/1.0.0",
			"X-Amz-User-Agent":         "aws-sdk-js/3.0.0 kiro-ide/1.0.0",
		},
	},
	// --- w7-prov-special-b: antigravity multi-backend provider (PAR-PROV-020) ---
	// providers.js:105-113. The ref carries a baseUrls fallback list; the primary
	// host is stored here and the sandbox fallback is built by the executor. The
	// User-Agent platform/arch suffix is omitted (it is host-runtime metadata).
	"antigravity": {
		Name:    "antigravity",
		BaseURL: "https://daily-cloudcode-pa.googleapis.com",
		Format:  "antigravity",
		Headers: map[string]string{
			"User-Agent": "antigravity/1.107.0",
		},
	},
	// --- w7-prov-openai: Western openai-format providers ---
	"nvidia": {
		Name:    "nvidia",
		BaseURL: "https://integrate.api.nvidia.com/v1/chat/completions",
		Format:  "openai",
	},
	"cerebras": {
		Name:    "cerebras",
		BaseURL: "https://api.cerebras.ai/v1/chat/completions",
		Format:  "openai",
	},
	"nebius": {
		Name:    "nebius",
		BaseURL: "https://api.studio.nebius.ai/v1/chat/completions",
		Format:  "openai",
	},
	"siliconflow": {
		Name:    "siliconflow",
		BaseURL: "https://api.siliconflow.cn/v1/chat/completions",
		Format:  "openai",
	},
	"hyperbolic": {
		Name:    "hyperbolic",
		BaseURL: "https://api.hyperbolic.xyz/v1/chat/completions",
		Format:  "openai",
	},
	"blackbox": {
		Name:    "blackbox",
		BaseURL: "https://api.blackbox.ai/chat/completions",
		Format:  "openai",
	},
	"gitlab": {
		Name:    "gitlab",
		BaseURL: "https://gitlab.com/api/v4/chat/completions",
		Format:  "openai",
	},
	"codebuddy": {
		// Device-code OAuth acquisition is a w7-prov-oauth concern (ESC-3).
		// Catalog entry satisfies PAR-PROV-051 HAVE.
		Name:    "codebuddy",
		BaseURL: "https://copilot.tencent.com/v1/chat/completions",
		Format:  "openai",
	},
	"vercel-ai-gateway": {
		Name:    "vercel-ai-gateway",
		BaseURL: "https://ai-gateway.vercel.sh/v1/chat/completions",
		Format:  "openai",
	},
	"chutes": {
		Name:    "chutes",
		BaseURL: "https://llm.chutes.ai/v1/chat/completions",
		Format:  "openai",
	},
	// --- w7-prov-openai: Free-tier openai bundle (PAR-PROV-067, 28 providers) ---
	// agentrouter excluded: format:"claude" (ESC-4).
	"aimlapi": {
		Name:    "aimlapi",
		BaseURL: "https://api.aimlapi.com/v1/chat/completions",
		Format:  "openai",
	},
	"novita": {
		Name:    "novita",
		BaseURL: "https://api.novita.ai/v3/openai/chat/completions",
		Format:  "openai",
	},
	"modal": {
		Name:    "modal",
		BaseURL: "https://api.modal.com/v1/chat/completions",
		Format:  "openai",
	},
	"reka": {
		Name:    "reka",
		BaseURL: "https://api.reka.ai/v1/chat/completions",
		Format:  "openai",
	},
	"nlpcloud": {
		Name:    "nlpcloud",
		BaseURL: "https://api.nlpcloud.io/v1/gpu/chatbot",
		Format:  "openai",
	},
	"bazaarlink": {
		Name:    "bazaarlink",
		BaseURL: "https://bazaarlink.ai/api/v1/chat/completions",
		Format:  "openai",
	},
	"completions": {
		Name:    "completions",
		BaseURL: "https://completions.me/api/v1/chat/completions",
		Format:  "openai",
	},
	"enally": {
		// Uses X-API-Key header instead of Bearer (first AuthHeader use in catalog).
		Name:       "enally",
		BaseURL:    "https://ai.enally.in/v1/chat/completions",
		Format:     "openai",
		AuthHeader: "x-api-key",
	},
	"freetheai": {
		Name:    "freetheai",
		BaseURL: "https://api.freetheai.xyz/v1/chat/completions",
		Format:  "openai",
	},
	"llm7": {
		Name:    "llm7",
		BaseURL: "https://api.llm7.io/v1/chat/completions",
		Format:  "openai",
	},
	"lepton": {
		Name:    "lepton",
		BaseURL: "https://api.lepton.ai/api/v1/chat/completions",
		Format:  "openai",
	},
	"kluster": {
		Name:    "kluster",
		BaseURL: "https://api.kluster.ai/v1/chat/completions",
		Format:  "openai",
	},
	"ai21": {
		Name:    "ai21",
		BaseURL: "https://api.ai21.com/studio/v1/chat/completions",
		Format:  "openai",
	},
	"inference-net": {
		Name:    "inference-net",
		BaseURL: "https://api.inference.net/v1/chat/completions",
		Format:  "openai",
	},
	"predibase": {
		Name:    "predibase",
		BaseURL: "https://serving.app.predibase.com/v1/chat/completions",
		Format:  "openai",
	},
	"bytez": {
		Name:    "bytez",
		BaseURL: "https://api.bytez.com/models/v2",
		Format:  "openai",
	},
	"morph": {
		Name:    "morph",
		BaseURL: "https://api.morphllm.com/v1/chat/completions",
		Format:  "openai",
	},
	"longcat": {
		Name:    "longcat",
		BaseURL: "https://api.longcat.chat/openai/v1/chat/completions",
		Format:  "openai",
	},
	"puter": {
		Name:    "puter",
		BaseURL: "https://api.puter.com/puterai/openai/v1/chat/completions",
		Format:  "openai",
	},
	"uncloseai": {
		Name:    "uncloseai",
		BaseURL: "https://hermes.ai.unturf.com/v1/chat/completions",
		Format:  "openai",
		NoAuth:  true,
	},
	"scaleway": {
		Name:    "scaleway",
		BaseURL: "https://api.scaleway.ai/v1/chat/completions",
		Format:  "openai",
	},
	"deepinfra": {
		Name:    "deepinfra",
		BaseURL: "https://api.deepinfra.com/v1/openai/chat/completions",
		Format:  "openai",
	},
	"sambanova": {
		Name:    "sambanova",
		BaseURL: "https://api.sambanova.ai/v1/chat/completions",
		Format:  "openai",
	},
	"nscale": {
		Name:    "nscale",
		BaseURL: "https://inference.api.nscale.com/v1/chat/completions",
		Format:  "openai",
	},
	"baseten": {
		Name:    "baseten",
		BaseURL: "https://inference.baseten.co/v1/chat/completions",
		Format:  "openai",
	},
	"publicai": {
		Name:    "publicai",
		BaseURL: "https://api.publicai.co/v1/chat/completions",
		Format:  "openai",
	},
	"nous-research": {
		Name:    "nous-research",
		BaseURL: "https://inference-api.nousresearch.com/v1/chat/completions",
		Format:  "openai",
	},
	"glhf": {
		Name:    "glhf",
		BaseURL: "https://glhf.chat/api/openai/v1/chat/completions",
		Format:  "openai",
	},
	// --- w7-prov-openai: Chinese openai-format providers ---
	"glm-cn": {
		Name:    "glm-cn",
		BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions",
		Format:  "openai",
	},
	"alicode": {
		Name:    "alicode",
		BaseURL: "https://coding.dashscope.aliyuncs.com/v1/chat/completions",
		Format:  "openai",
	},
	"alicode-intl": {
		Name:    "alicode-intl",
		BaseURL: "https://coding-intl.dashscope.aliyuncs.com/v1/chat/completions",
		Format:  "openai",
	},
	"volcengine-ark": {
		Name:    "volcengine-ark",
		BaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions",
		Format:  "openai",
	},
	"byteplus": {
		Name:    "byteplus",
		BaseURL: "https://ark.ap-southeast.bytepluses.com/api/coding/v3/chat/completions",
		Format:  "openai",
	},
	"xiaomi-mimo": {
		Name:    "xiaomi-mimo",
		BaseURL: "https://api.xiaomimimo.com/v1/chat/completions",
		Format:  "openai",
	},
	"opencode-go": {
		Name:    "opencode-go",
		BaseURL: "https://opencode.ai/zen/go/v1/chat/completions",
		Format:  "openai",
	},
	"opencode": {
		Name:    "opencode",
		BaseURL: "https://opencode.ai",
		Format:  "openai",
		Headers: map[string]string{
			"x-opencode-client": "desktop",
		},
		NoAuth: true,
	},
	// --- w7-prov-special-a: claude-format providers (reuse anthropic path) ---
	// baseUrl is the FULL Anthropic-Messages endpoint (ref @827e5c3); the
	// anthropic adapter (NewForProvider) appends "?beta=true" and sets x-api-key
	// auth. Headers carry CLAUDE_API_HEADERS (providers.js:24-27).
	"glm": {
		Name:       "glm",
		BaseURL:    "https://api.z.ai/api/anthropic/v1/messages",
		Format:     "claude",
		AuthHeader: "x-api-key",
		Headers: map[string]string{
			"Anthropic-Version": "2023-06-01",
			"Anthropic-Beta":    "claude-code-20250219,interleaved-thinking-2025-05-14",
		},
	},
	"kimi": {
		Name:       "kimi",
		BaseURL:    "https://api.kimi.com/coding/v1/messages",
		Format:     "claude",
		AuthHeader: "x-api-key",
		Headers: map[string]string{
			"Anthropic-Version": "2023-06-01",
			"Anthropic-Beta":    "claude-code-20250219,interleaved-thinking-2025-05-14",
		},
	},
	"minimax": {
		Name:       "minimax",
		BaseURL:    "https://api.minimax.io/anthropic/v1/messages",
		Format:     "claude",
		AuthHeader: "x-api-key",
		Headers: map[string]string{
			"Anthropic-Version": "2023-06-01",
			"Anthropic-Beta":    "claude-code-20250219,interleaved-thinking-2025-05-14",
		},
	},
	"minimax-cn": {
		Name:       "minimax-cn",
		BaseURL:    "https://api.minimaxi.com/anthropic/v1/messages",
		Format:     "claude",
		AuthHeader: "x-api-key",
		Headers: map[string]string{
			"Anthropic-Version": "2023-06-01",
			"Anthropic-Beta":    "claude-code-20250219,interleaved-thinking-2025-05-14",
		},
	},
	// --- w7-prov-special-a: URL-template / runtime-URL-build openai providers ---
	// The endpoint URL is computed at request time by the urltemplate adapter
	// from schemas.Key.ProviderSpecificData. Bodies are plain OpenAI.
	"cloudflare-ai": {
		// {accountId} substituted at request time (default.js:64-68).
		Name:    "cloudflare-ai",
		BaseURL: "https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1/chat/completions",
		Format:  "openai",
	},
	"azure": {
		// baseUrl empty in ref; the resource URL is built from
		// providerSpecificData (azure.js:8-23). Auth via the api-key header.
		Name:       "azure",
		BaseURL:    "",
		Format:     "openai",
		AuthHeader: "api-key",
	},
	"xiaomi-tokenplan": {
		// region->baseURL resolved at request time (providers.js:447-457); the
		// sgp seed URL is kept for introspection.
		Name:    "xiaomi-tokenplan",
		BaseURL: "https://token-plan-sgp.xiaomimimo.com/v1/chat/completions",
		Format:  "openai",
	},
	"vertex": {
		// Partner-openai path (vertex.js:49-53): the endpoint is built at
		// request time from providerSpecificData.projectId. The native
		// gemini-on-vertex format is deferred (ESC-A1). BaseURL is the API host
		// seed kept for introspection.
		Name:    "vertex",
		BaseURL: "https://aiplatform.googleapis.com",
		Format:  "openai",
	},
	// --- w7-prov-special-a: commandcode (custom-JSON via existing converters) ---
	"commandcode": {
		Name:    "commandcode",
		BaseURL: "https://api.commandcode.ai/alpha/generate",
		Format:  "commandcode",
		Headers: map[string]string{
			"x-command-code-version": "0.25.7",
			"x-cli-environment":      "cli",
		},
	},
	"ollama": {
		Name:    "ollama",
		BaseURL: "https://ollama.com/api/chat",
		Format:  "ollama",
		NoAuth:  true,
	},
	"ollama-local": {
		Name:    "ollama-local",
		BaseURL: "http://localhost:11434/api/chat",
		Format:  "ollama",
		NoAuth:  true,
	},
}

// Lookup returns the ProviderConfig for the given provider name.
func Lookup(provider string) (ProviderConfig, bool) {
	cfg, ok := Providers[provider]
	return cfg, ok
}

// ResolveOllamaHost is the Go port of resolveOllamaLocalHost
// (providers.js:442-445). It returns the trimmed override or the default
// host, with any trailing slash removed.
func ResolveOllamaHost(baseURLOverride string) string {
	raw := strings.TrimSpace(baseURLOverride)
	if raw == "" {
		raw = "http://localhost:11434"
	}
	return strings.TrimRight(raw, "/")
}
