package catalog

import (
	"testing"
)

func TestLookupKnownProviders(t *testing.T) {
	wantBaseURL := map[string]string{
		"groq":         "https://api.groq.com/openai/v1/chat/completions",
		"deepseek":     "https://api.deepseek.com/chat/completions",
		"mistral":      "https://api.mistral.ai/v1/chat/completions",
		"cohere":       "https://api.cohere.ai/v1/chat/completions",
		"together":     "https://api.together.xyz/v1/chat/completions",
		"fireworks":    "https://api.fireworks.ai/inference/v1/chat/completions",
		"openrouter":   "https://openrouter.ai/api/v1/chat/completions",
		"xai":          "https://api.x.ai/v1/chat/completions",
		"perplexity":   "https://api.perplexity.ai/chat/completions",
		"ollama":       "https://ollama.com/api/chat",
		"ollama-local": "http://localhost:11434/api/chat",
	}
	wantFormat := map[string]string{
		"groq":         "openai",
		"deepseek":     "openai",
		"mistral":      "openai",
		"cohere":       "openai",
		"together":     "openai",
		"fireworks":    "openai",
		"openrouter":   "openai",
		"xai":          "openai",
		"perplexity":   "openai",
		"ollama":       "ollama",
		"ollama-local": "ollama",
	}
	known := []string{
		"groq", "deepseek", "mistral", "cohere",
		"together", "fireworks", "openrouter", "xai",
		"perplexity", "ollama", "ollama-local",
	}
	for _, name := range known {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.Name != name {
			t.Errorf("Lookup(%q).Name = %q, want %q", name, cfg.Name, name)
		}
		if got, want := cfg.BaseURL, wantBaseURL[name]; got != want {
			t.Errorf("Lookup(%q).BaseURL = %q, want %q", name, got, want)
		}
		if got, want := cfg.Format, wantFormat[name]; got != want {
			t.Errorf("Lookup(%q).Format = %q, want %q", name, got, want)
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	_, ok := Lookup("nonexistent")
	if ok {
		t.Fatal("Lookup(\"nonexistent\") returned ok=true, want false")
	}
}

func TestOpenRouterHeaders(t *testing.T) {
	cfg, ok := Lookup("openrouter")
	if !ok {
		t.Fatal("Lookup(\"openrouter\") returned ok=false")
	}
	if got, want := cfg.Headers["HTTP-Referer"], "https://endpoint-proxy.local"; got != want {
		t.Errorf("openrouter HTTP-Referer = %q, want %q", got, want)
	}
	if got, want := cfg.Headers["X-Title"], "Endpoint Proxy"; got != want {
		t.Errorf("openrouter X-Title = %q, want %q", got, want)
	}
}

func TestOllamaConfig(t *testing.T) {
	for _, name := range []string{"ollama", "ollama-local"} {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.Format != "ollama" {
			t.Errorf("Lookup(%q).Format = %q, want %q", name, cfg.Format, "ollama")
		}
		if !cfg.NoAuth {
			t.Errorf("Lookup(%q).NoAuth = false, want true", name)
		}
	}
}

func TestProviderRetryOverride(t *testing.T) {
	cfg, ok := Lookup("kiro")
	if !ok {
		t.Fatal("kiro not in catalog")
	}
	got := cfg.RetryOverride()
	want429 := 2
	if got[429] != want429 {
		t.Errorf("kiro Retry[429] = %d, want %d", got[429], want429)
	}
}

func TestChineseOpenAIProviders(t *testing.T) {
	cases := map[string]string{
		"glm-cn":         "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions",
		"alicode":        "https://coding.dashscope.aliyuncs.com/v1/chat/completions",
		"alicode-intl":   "https://coding-intl.dashscope.aliyuncs.com/v1/chat/completions",
		"volcengine-ark": "https://ark.cn-beijing.volces.com/api/coding/v3/chat/completions",
		"byteplus":       "https://ark.ap-southeast.bytepluses.com/api/coding/v3/chat/completions",
		"xiaomi-mimo":    "https://api.xiaomimimo.com/v1/chat/completions",
		"opencode-go":    "https://opencode.ai/zen/go/v1/chat/completions",
	}
	for name, wantURL := range cases {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.Name != name {
			t.Errorf("Lookup(%q).Name = %q, want %q", name, cfg.Name, name)
		}
		if cfg.BaseURL != wantURL {
			t.Errorf("Lookup(%q).BaseURL = %q, want %q", name, cfg.BaseURL, wantURL)
		}
		if cfg.Format != "openai" {
			t.Errorf("Lookup(%q).Format = %q, want %q", name, cfg.Format, "openai")
		}
	}

	// opencode is openai-shaped but NoAuth with a custom client header.
	cfg, ok := Lookup("opencode")
	if !ok {
		t.Fatalf("Lookup(\"opencode\") returned ok=false")
	}
	if cfg.BaseURL != "https://opencode.ai" {
		t.Errorf("opencode BaseURL = %q, want %q", cfg.BaseURL, "https://opencode.ai")
	}
	if cfg.Format != "openai" {
		t.Errorf("opencode Format = %q, want %q", cfg.Format, "openai")
	}
	if !cfg.NoAuth {
		t.Errorf("opencode NoAuth = false, want true")
	}
	if got, want := cfg.Headers["x-opencode-client"], "desktop"; got != want {
		t.Errorf("opencode header x-opencode-client = %q, want %q", got, want)
	}
}

func TestWesternOpenAIProviders(t *testing.T) {
	cases := map[string]string{
		"nvidia":            "https://integrate.api.nvidia.com/v1/chat/completions",
		"cerebras":          "https://api.cerebras.ai/v1/chat/completions",
		"nebius":            "https://api.studio.nebius.ai/v1/chat/completions",
		"siliconflow":       "https://api.siliconflow.cn/v1/chat/completions",
		"hyperbolic":        "https://api.hyperbolic.xyz/v1/chat/completions",
		"blackbox":          "https://api.blackbox.ai/chat/completions",
		"gitlab":            "https://gitlab.com/api/v4/chat/completions",
		"codebuddy":         "https://copilot.tencent.com/v1/chat/completions",
		"vercel-ai-gateway": "https://ai-gateway.vercel.sh/v1/chat/completions",
		"chutes":            "https://llm.chutes.ai/v1/chat/completions",
	}
	for name, wantURL := range cases {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.BaseURL != wantURL {
			t.Errorf("Lookup(%q).BaseURL = %q, want %q", name, cfg.BaseURL, wantURL)
		}
		if cfg.Format != "openai" {
			t.Errorf("Lookup(%q).Format = %q, want %q", name, cfg.Format, "openai")
		}
	}
}

func TestWesternOpenAIModels(t *testing.T) {
	wantCount := map[string]int{
		"nvidia":      4,
		"cerebras":    6,
		"nebius":      2,
		"siliconflow": 10,
		"hyperbolic":  8,
		"blackbox":    17,
	}
	for p, n := range wantCount {
		if got := len(ModelsFor(p)); got != n {
			t.Errorf("ModelsFor(%q) len = %d, want %d", p, got, n)
		}
	}

	// No-catalog providers must have empty model list.
	for _, p := range []string{"gitlab", "codebuddy", "vercel-ai-gateway", "chutes"} {
		if got := len(ModelsFor(p)); got != 0 {
			t.Errorf("ModelsFor(%q) len = %d, want 0 (no-catalog provider)", p, got)
		}
	}

	// nvidia: assert embedding and stt typed entries with Params.
	var hasEmbedding, hasSTT bool
	for _, m := range ModelsFor("nvidia") {
		if m.Type == "embedding" {
			hasEmbedding = true
		}
		if m.Type == "stt" {
			hasSTT = true
			if len(m.Params) == 0 {
				t.Errorf("nvidia stt entry %q has empty Params", m.ID)
			}
		}
	}
	if !hasEmbedding {
		t.Error("nvidia: missing embedding-type model")
	}
	if !hasSTT {
		t.Error("nvidia: missing stt-type model")
	}

	// nebius: assert embedding entry present.
	var nebiusHasEmbedding bool
	for _, m := range ModelsFor("nebius") {
		if m.Type == "embedding" {
			nebiusHasEmbedding = true
		}
	}
	if !nebiusHasEmbedding {
		t.Error("nebius: missing embedding-type model")
	}
}

func TestFreeTierProviders(t *testing.T) {
	// 28 openai free-tier providers (agentrouter excluded — ESC-4, format:claude).
	cases := map[string]string{
		"aimlapi":       "https://api.aimlapi.com/v1/chat/completions",
		"novita":        "https://api.novita.ai/v3/openai/chat/completions",
		"modal":         "https://api.modal.com/v1/chat/completions",
		"reka":          "https://api.reka.ai/v1/chat/completions",
		"nlpcloud":      "https://api.nlpcloud.io/v1/gpu/chatbot",
		"bazaarlink":    "https://bazaarlink.ai/api/v1/chat/completions",
		"completions":   "https://completions.me/api/v1/chat/completions",
		"enally":        "https://ai.enally.in/v1/chat/completions",
		"freetheai":     "https://api.freetheai.xyz/v1/chat/completions",
		"llm7":          "https://api.llm7.io/v1/chat/completions",
		"lepton":        "https://api.lepton.ai/api/v1/chat/completions",
		"kluster":       "https://api.kluster.ai/v1/chat/completions",
		"ai21":          "https://api.ai21.com/studio/v1/chat/completions",
		"inference-net": "https://api.inference.net/v1/chat/completions",
		"predibase":     "https://serving.app.predibase.com/v1/chat/completions",
		"bytez":         "https://api.bytez.com/models/v2",
		"morph":         "https://api.morphllm.com/v1/chat/completions",
		"longcat":       "https://api.longcat.chat/openai/v1/chat/completions",
		"puter":         "https://api.puter.com/puterai/openai/v1/chat/completions",
		"uncloseai":     "https://hermes.ai.unturf.com/v1/chat/completions",
		"scaleway":      "https://api.scaleway.ai/v1/chat/completions",
		"deepinfra":     "https://api.deepinfra.com/v1/openai/chat/completions",
		"sambanova":     "https://api.sambanova.ai/v1/chat/completions",
		"nscale":        "https://inference.api.nscale.com/v1/chat/completions",
		"baseten":       "https://inference.baseten.co/v1/chat/completions",
		"publicai":      "https://api.publicai.co/v1/chat/completions",
		"nous-research": "https://inference-api.nousresearch.com/v1/chat/completions",
		"glhf":          "https://glhf.chat/api/openai/v1/chat/completions",
	}
	for name, wantURL := range cases {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.BaseURL != wantURL {
			t.Errorf("Lookup(%q).BaseURL = %q, want %q", name, cfg.BaseURL, wantURL)
		}
		if cfg.Format != "openai" {
			t.Errorf("Lookup(%q).Format = %q, want %q", name, cfg.Format, "openai")
		}
	}

	// enally uses x-api-key AuthHeader (first use of that field).
	cfg, ok := Lookup("enally")
	if !ok {
		t.Fatal("Lookup(\"enally\") returned ok=false")
	}
	if cfg.AuthHeader != "x-api-key" {
		t.Errorf("enally AuthHeader = %q, want %q", cfg.AuthHeader, "x-api-key")
	}

	// uncloseai is NoAuth.
	cfg, ok = Lookup("uncloseai")
	if !ok {
		t.Fatal("Lookup(\"uncloseai\") returned ok=false")
	}
	if !cfg.NoAuth {
		t.Error("uncloseai NoAuth = false, want true")
	}

	// agentrouter must NOT be present (ESC-4 — claude format, excluded).
	if _, ok := Lookup("agentrouter"); ok {
		t.Error("agentrouter must NOT be in catalog (ESC-4: claude format)")
	}
}

func TestFreeTierModels(t *testing.T) {
	wantCount := map[string]int{
		"aimlapi":       5,
		"novita":        4,
		"modal":         1,
		"reka":          2,
		"nlpcloud":      3,
		"bazaarlink":    2,
		"completions":   4,
		"enally":        3,
		"freetheai":     4,
		"llm7":          3,
		"lepton":        4,
		"kluster":       4,
		"ai21":          2,
		"inference-net": 3,
		"predibase":     3,
		"bytez":         3,
		"morph":         2,
		"longcat":       3,
		"puter":         5,
		"uncloseai":     2,
		"scaleway":      3,
		"deepinfra":     3,
		"sambanova":     3,
		"nscale":        2,
		"baseten":       2,
		"publicai":      1,
		"nous-research": 2,
		"glhf":          3,
	}
	for p, n := range wantCount {
		if got := len(ModelsFor(p)); got != n {
			t.Errorf("ModelsFor(%q) len = %d, want %d", p, got, n)
		}
	}

	// Spot-check aliases that must resolve.
	aliasCases := map[string]string{
		"aiml":  "aimlapi",
		"enly":  "enally",
		"unc":   "uncloseai",
		"nous":  "nous-research",
		"glhf":  "glhf",
	}
	for alias, want := range aliasCases {
		got, ok := ResolveProviderAlias(alias)
		if !ok {
			t.Errorf("ResolveProviderAlias(%q) ok=false, want true", alias)
			continue
		}
		if got != want {
			t.Errorf("ResolveProviderAlias(%q) = %q, want %q", alias, got, want)
		}
	}
}

// TestClaudeFormatProviders (w7-prov-special-a) verifies the claude-format
// catalog entries (glm/kimi/minimax/minimax-cn): Format:"claude", the ref base
// URLs (full .../v1/messages endpoints), AuthHeader:"x-api-key", and the
// CLAUDE_API_HEADERS (Anthropic-Version + Anthropic-Beta).
func TestClaudeFormatProviders(t *testing.T) {
	cases := map[string]string{
		"glm":        "https://api.z.ai/api/anthropic/v1/messages",
		"kimi":       "https://api.kimi.com/coding/v1/messages",
		"minimax":    "https://api.minimax.io/anthropic/v1/messages",
		"minimax-cn": "https://api.minimaxi.com/anthropic/v1/messages",
	}
	for name, wantURL := range cases {
		cfg, ok := Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) returned ok=false", name)
		}
		if cfg.Name != name {
			t.Errorf("Lookup(%q).Name = %q, want %q", name, cfg.Name, name)
		}
		if cfg.BaseURL != wantURL {
			t.Errorf("Lookup(%q).BaseURL = %q, want %q", name, cfg.BaseURL, wantURL)
		}
		if cfg.Format != "claude" {
			t.Errorf("Lookup(%q).Format = %q, want claude", name, cfg.Format)
		}
		if cfg.AuthHeader != "x-api-key" {
			t.Errorf("Lookup(%q).AuthHeader = %q, want x-api-key", name, cfg.AuthHeader)
		}
		if got, want := cfg.Headers["Anthropic-Version"], "2023-06-01"; got != want {
			t.Errorf("Lookup(%q).Headers[Anthropic-Version] = %q, want %q", name, got, want)
		}
		if got, want := cfg.Headers["Anthropic-Beta"], "claude-code-20250219,interleaved-thinking-2025-05-14"; got != want {
			t.Errorf("Lookup(%q).Headers[Anthropic-Beta] = %q, want %q", name, got, want)
		}
	}
}

// TestCommandCodeProvider (w7-prov-special-a) verifies the commandcode catalog
// entry: Format:"commandcode", ref base URL, and the custom CLI headers.
func TestCommandCodeProvider(t *testing.T) {
	cfg, ok := Lookup("commandcode")
	if !ok {
		t.Fatal("Lookup(commandcode) returned ok=false")
	}
	if cfg.BaseURL != "https://api.commandcode.ai/alpha/generate" {
		t.Errorf("commandcode BaseURL = %q, want https://api.commandcode.ai/alpha/generate", cfg.BaseURL)
	}
	if cfg.Format != "commandcode" {
		t.Errorf("commandcode Format = %q, want commandcode", cfg.Format)
	}
	if got, want := cfg.Headers["x-command-code-version"], "0.25.7"; got != want {
		t.Errorf("x-command-code-version = %q, want %q", got, want)
	}
	if got, want := cfg.Headers["x-cli-environment"], "cli"; got != want {
		t.Errorf("x-cli-environment = %q, want %q", got, want)
	}
}

// TestURLTemplateProviders (w7-prov-special-a) verifies the URL-template/build
// openai providers' catalog entries. The BaseURL is a template/seed; the actual
// endpoint is built at request time by the urltemplate adapter.
func TestURLTemplateProviders(t *testing.T) {
	// cloudflare-ai carries the {accountId} template.
	cf, ok := Lookup("cloudflare-ai")
	if !ok {
		t.Fatal("Lookup(cloudflare-ai) returned ok=false")
	}
	if cf.BaseURL != "https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1/chat/completions" {
		t.Errorf("cloudflare-ai BaseURL = %q", cf.BaseURL)
	}
	if cf.Format != "openai" {
		t.Errorf("cloudflare-ai Format = %q, want openai", cf.Format)
	}

	// azure: baseUrl empty in ref (resource URL built by the executor).
	az, ok := Lookup("azure")
	if !ok {
		t.Fatal("Lookup(azure) returned ok=false")
	}
	if az.BaseURL != "" {
		t.Errorf("azure BaseURL = %q, want empty", az.BaseURL)
	}
	if az.Format != "openai" {
		t.Errorf("azure Format = %q, want openai", az.Format)
	}

	// xiaomi-tokenplan: region-resolved; the seed sgp URL is the BaseURL.
	xm, ok := Lookup("xiaomi-tokenplan")
	if !ok {
		t.Fatal("Lookup(xiaomi-tokenplan) returned ok=false")
	}
	if xm.BaseURL != "https://token-plan-sgp.xiaomimimo.com/v1/chat/completions" {
		t.Errorf("xiaomi-tokenplan BaseURL = %q", xm.BaseURL)
	}
	if xm.Format != "openai" {
		t.Errorf("xiaomi-tokenplan Format = %q, want openai", xm.Format)
	}
}

// TestVertexProvider (w7-prov-special-a) verifies the vertex catalog entry. The
// partner-openai path is shipped (URL built at request time from
// providerSpecificData.projectId); the native gemini-on-vertex format is
// deferred (ESC-A1).
func TestVertexProvider(t *testing.T) {
	cfg, ok := Lookup("vertex")
	if !ok {
		t.Fatal("Lookup(vertex) returned ok=false")
	}
	if cfg.Format != "openai" {
		t.Errorf("vertex Format = %q, want openai (partner path)", cfg.Format)
	}
	// BaseURL is the API host seed; the partner endpoint is built per-request.
	if cfg.BaseURL != "https://aiplatform.googleapis.com" {
		t.Errorf("vertex BaseURL = %q, want https://aiplatform.googleapis.com", cfg.BaseURL)
	}
}

func TestResolveOllamaHost(t *testing.T) {
	// override trimmed
	if got := ResolveOllamaHost("  http://ollama.local:11434/  "); got != "http://ollama.local:11434" {
		t.Errorf("ResolveOllamaHost(trimmed) = %q", got)
	}
	// default
	if got := ResolveOllamaHost(""); got != "http://localhost:11434" {
		t.Errorf("ResolveOllamaHost(default) = %q, want %q", got, "http://localhost:11434")
	}
	// trailing slash stripped
	if got := ResolveOllamaHost("http://host:11434/"); got != "http://host:11434" {
		t.Errorf("ResolveOllamaHost(trailing slash) = %q, want %q", got, "http://host:11434")
	}
	// multiple trailing slashes stripped
	if got := ResolveOllamaHost("http://host:11434///"); got != "http://host:11434" {
		t.Errorf("ResolveOllamaHost(multiple slashes) = %q, want %q", got, "http://host:11434")
	}
}

// TestAntigravityProvider (w7-prov-special-b) verifies the antigravity catalog
// entry: Format "antigravity", the primary daily-cloudcode-pa base URL, and the
// antigravity/1.107.0 User-Agent header (providers.js:105-113).
func TestAntigravityProvider(t *testing.T) {
	cfg, ok := Lookup("antigravity")
	if !ok {
		t.Fatal("Lookup(antigravity) returned ok=false")
	}
	if cfg.Format != "antigravity" {
		t.Errorf("antigravity Format = %q, want antigravity", cfg.Format)
	}
	if cfg.BaseURL != "https://daily-cloudcode-pa.googleapis.com" {
		t.Errorf("antigravity BaseURL = %q, want the primary daily-cloudcode-pa host", cfg.BaseURL)
	}
	if got := cfg.Headers["User-Agent"]; got != "antigravity/1.107.0" {
		t.Errorf("antigravity User-Agent = %q, want antigravity/1.107.0", got)
	}
}
