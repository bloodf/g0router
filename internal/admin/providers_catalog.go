package admin

import (
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// providerCatalogDTO is the provider-shaped read overlay the dashboard consumes
// (plan §1.6, PAR-UI-087/088). It composes the existing providers/connections
// store data with static known-provider metadata; it does NOT mutate either CRUD
// contract or introduce new tables.
type providerCatalogDTO struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	DisplayName     string   `json:"display_name"`
	Description     string   `json:"description"`
	AuthTypes       []string `json:"auth_types"`
	Capabilities    []string `json:"capabilities"`
	ConnectionCount int      `json:"connection_count"`
	Status          string   `json:"status"`
}

// catalogConnectionDTO is the UI-shaped connection (plan §8 ESCALATION-2):
// provider/auth_type/is_active/needs_reauth, with NO secret material. The
// existing connectionDTO (provider_id/kind/secret_set) is left untouched.
type catalogConnectionDTO struct {
	ID          string   `json:"id"`
	Provider    string   `json:"provider"`
	Name        string   `json:"name"`
	AuthType    string   `json:"auth_type"`
	IsActive    bool     `json:"is_active"`
	Models      []string `json:"models"`
	Priority    int      `json:"priority"`
	NeedsReauth bool     `json:"needs_reauth"`
}

type catalogModelDTO struct {
	ID            string  `json:"id"`
	Provider      string  `json:"provider"`
	Name          string  `json:"name"`
	InputCost     float64 `json:"input_cost"`
	OutputCost    float64 `json:"output_cost"`
	ContextWindow int     `json:"context_window"`
	IsDisabled    bool    `json:"is_disabled"`
	IsCustom      bool    `json:"is_custom"`
}

type suggestedModelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type batchResultDTO struct {
	Provider  string `json:"provider"`
	OK        bool   `json:"ok"`
	LatencyMS int    `json:"latency_ms"`
}

// catalogMeta is the static known-provider metadata, keyed by provider type. It
// mirrors the dashboard's mock catalog (providerinfo.ProviderMatrix surface).
type catalogMeta struct {
	displayName  string
	description  string
	authTypes    []string
	capabilities []string
}

var knownProviderMeta = map[string]catalogMeta{
	"openai":     {"OpenAI", "GPT-4, GPT-3.5, DALL-E, Whisper", []string{"api_key"}, []string{"chat", "images", "audio", "embeddings"}},
	"anthropic":  {"Anthropic", "Claude 3.5 Sonnet, Opus, Haiku", []string{"api_key", "oauth"}, []string{"chat", "vision"}},
	"gemini":     {"Google AI", "Gemini Pro, Flash, Ultra", []string{"api_key", "oauth"}, []string{"chat", "vision", "embeddings"}},
	"google":     {"Google AI", "Gemini Pro, Flash, Ultra", []string{"api_key", "oauth"}, []string{"chat", "vision", "embeddings"}},
	"azure":      {"Azure OpenAI", "Enterprise GPT-4 via Azure", []string{"api_key"}, []string{"chat", "embeddings"}},
	"bedrock":    {"AWS Bedrock", "Amazon Claude, Llama, Titan", []string{"api_key", "custom"}, []string{"chat", "embeddings"}},
	"cerebras":   {"Cerebras", "Fast inference on Cerebras hardware", []string{"api_key"}, []string{"chat"}},
	"cohere":     {"Cohere", "Command, Embed, Rerank", []string{"api_key"}, []string{"chat", "embeddings"}},
	"deepseek":   {"DeepSeek", "DeepSeek V3, Coder", []string{"api_key"}, []string{"chat"}},
	"fireworks":  {"Fireworks AI", "Fast open-source inference", []string{"api_key"}, []string{"chat", "embeddings"}},
	"groq":       {"Groq", "Ultra-fast LLM inference", []string{"api_key"}, []string{"chat"}},
	"huggingface": {"Hugging Face", "Inference API and endpoints", []string{"api_key"}, []string{"chat", "embeddings"}},
	"minimax":    {"MiniMax", "MiniMax M3 and multi-modal models", []string{"api_key"}, []string{"chat"}},
	"mistral":    {"Mistral AI", "Mistral Large, Medium, Small", []string{"api_key"}, []string{"chat"}},
	"nebius":     {"Nebius", "Nebius AI inference", []string{"api_key"}, []string{"chat"}},
	"nvidia":     {"NVIDIA", "NVIDIA NIM inference", []string{"api_key"}, []string{"chat"}},
	"ollama":     {"Ollama", "Local open-source models", []string{"noauth"}, []string{"chat", "embeddings"}},
	"openrouter": {"OpenRouter", "Unified API for 100+ models", []string{"api_key"}, []string{"chat", "images"}},
	"perplexity": {"Perplexity", "Search-augmented LLMs", []string{"api_key"}, []string{"chat"}},
	"qwen":       {"Qwen", "Alibaba Qwen models", []string{"api_key"}, []string{"chat"}},
	"together":   {"Together AI", "Open-source model hub", []string{"api_key"}, []string{"chat", "images"}},
	"vertex":     {"Google Vertex", "Gemini on GCP", []string{"oauth"}, []string{"chat", "vision"}},
	"xai":        {"xAI", "Grok models", []string{"api_key"}, []string{"chat", "vision"}},
	"replicate":  {"Replicate", "Run any open-source model", []string{"api_key"}, []string{"chat", "images", "audio"}},
	"moonshot":   {"Moonshot AI", "Kimi long-context models", []string{"api_key"}, []string{"chat"}},
	"ai21":       {"AI21 Labs", "Jamba, Jurassic models", []string{"api_key"}, []string{"chat"}},
}

// catalogModelMeta is the static per-type default model catalog (mirrors the
// dashboard mock catalog). Costs are USD per 1M tokens.
var catalogModelMeta = map[string][]catalogModelDTO{
	"openai": {
		{ID: "gpt-4o", Name: "gpt-4o", InputCost: 2.5, OutputCost: 10.0, ContextWindow: 128000},
		{ID: "gpt-4o-mini", Name: "gpt-4o-mini", InputCost: 0.15, OutputCost: 0.6, ContextWindow: 128000},
	},
	"anthropic": {
		{ID: "claude-sonnet-4", Name: "claude-sonnet-4", InputCost: 3.0, OutputCost: 15.0, ContextWindow: 200000},
		{ID: "claude-opus-4", Name: "claude-opus-4", InputCost: 15.0, OutputCost: 75.0, ContextWindow: 200000},
		{ID: "claude-3-5-haiku-20241022", Name: "claude-3-5-haiku-20241022", InputCost: 0.8, OutputCost: 4.0, ContextWindow: 200000},
	},
	"gemini": {
		{ID: "gemini-2.5-flash", Name: "gemini-2.5-flash", InputCost: 0.3, OutputCost: 2.5, ContextWindow: 1000000},
		{ID: "gemini-2.5-flash-lite", Name: "gemini-2.5-flash-lite", InputCost: 0.1, OutputCost: 0.4, ContextWindow: 1000000},
	},
	"google": {
		{ID: "gemini-2.5-flash", Name: "gemini-2.5-flash", InputCost: 0.3, OutputCost: 2.5, ContextWindow: 1000000},
		{ID: "gemini-2.5-flash-lite", Name: "gemini-2.5-flash-lite", InputCost: 0.1, OutputCost: 0.4, ContextWindow: 1000000},
	},
	"groq": {
		{ID: "llama-3.3-70b-versatile", Name: "llama-3.3-70b-versatile", InputCost: 0.59, OutputCost: 0.79, ContextWindow: 128000},
		{ID: "llama-3.1-8b-instant", Name: "llama-3.1-8b-instant", InputCost: 0.05, OutputCost: 0.08, ContextWindow: 128000},
	},
	"mistral": {
		{ID: "mistral-large-latest", Name: "mistral-large-latest", InputCost: 2.0, OutputCost: 6.0, ContextWindow: 128000},
		{ID: "mistral-small-latest", Name: "mistral-small-latest", InputCost: 0.1, OutputCost: 0.3, ContextWindow: 128000},
	},
	"deepseek": {
		{ID: "deepseek-chat", Name: "deepseek-chat", InputCost: 0.27, OutputCost: 1.1, ContextWindow: 64000},
		{ID: "deepseek-reasoner", Name: "deepseek-reasoner", InputCost: 0.55, OutputCost: 2.19, ContextWindow: 64000},
	},
	"openrouter": {
		{ID: "openai/gpt-4o", Name: "openai/gpt-4o", InputCost: 2.5, OutputCost: 10.0, ContextWindow: 128000},
		{ID: "openai/gpt-4o-mini", Name: "openai/gpt-4o-mini", InputCost: 0.15, OutputCost: 0.6, ContextWindow: 128000},
	},
	"xai": {
		{ID: "grok-4.3", Name: "grok-4.3", InputCost: 1.25, OutputCost: 2.5, ContextWindow: 128000},
	},
	"perplexity": {
		{ID: "sonar", Name: "sonar", InputCost: 1.0, OutputCost: 1.0, ContextWindow: 128000},
		{ID: "sonar-pro", Name: "sonar-pro", InputCost: 3.0, OutputCost: 15.0, ContextWindow: 128000},
	},
}

// metaForType returns the static metadata for a provider type, falling back to a
// neutral api_key entry (plan §1.6 fallback) when the type is unknown.
func metaForType(typ, name string) catalogMeta {
	if m, ok := knownProviderMeta[typ]; ok {
		return m
	}
	display := name
	if display == "" {
		display = typ
	}
	return catalogMeta{displayName: display, authTypes: []string{"api_key"}, capabilities: []string{}}
}

// buildCatalogEntry composes one provider's catalog DTO from its store record and
// the live connection set.
func buildCatalogEntry(p *store.ProviderRecord, connections []*store.Connection) providerCatalogDTO {
	meta := metaForType(p.Type, p.Name)
	count := 0
	active := false
	for _, c := range connections {
		if c.ProviderID == p.ID {
			count++
			active = true
		}
	}
	status := "inactive"
	if active {
		status = "active"
	}
	return providerCatalogDTO{
		ID:              p.ID,
		Name:            p.Name,
		Type:            p.Type,
		DisplayName:     meta.displayName,
		Description:     meta.description,
		AuthTypes:       meta.authTypes,
		Capabilities:    meta.capabilities,
		ConnectionCount: count,
		Status:          status,
	}
}

// ListProviderCatalog handles GET /api/providers/catalog (PAR-UI-087).
func (h *Handlers) ListProviderCatalog(ctx *fasthttp.RequestCtx) {
	providers, err := h.store.ListProviders()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list providers")
		return
	}
	connections, err := h.store.ListConnections()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list connections")
		return
	}
	out := make([]providerCatalogDTO, 0, len(providers))
	for _, p := range providers {
		out = append(out, buildCatalogEntry(p, connections))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// GetProviderCatalog handles GET /api/providers/{id}/catalog (PAR-UI-088).
func (h *Handlers) GetProviderCatalog(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	p, err := h.store.GetProvider(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}
	connections, err := h.store.ListConnections()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list connections")
		return
	}
	writeData(ctx, fasthttp.StatusOK, buildCatalogEntry(p, connections))
}

// GetProviderConnections handles GET /api/providers/{id}/connections (PAR-UI-088).
// It emits UI-shaped connections (ESCALATION-2) and masks all secret material.
func (h *Handlers) GetProviderConnections(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	connections, err := h.store.ListConnections()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list connections")
		return
	}
	out := make([]catalogConnectionDTO, 0)
	for _, c := range connections {
		if c.ProviderID != id {
			continue
		}
		out = append(out, toCatalogConnectionDTO(c))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// toCatalogConnectionDTO maps a store connection to the UI shape, deriving
// is_active (a connection row is active once it exists) and needs_reauth (an
// oauth connection whose token has an expiry in the past).
func toCatalogConnectionDTO(c *store.Connection) catalogConnectionDTO {
	needsReauth := c.Kind == "oauth" && c.ExpiresAt != 0 && c.ExpiresAt < time.Now().Unix()
	return catalogConnectionDTO{
		ID:          c.ID,
		Provider:    c.ProviderID,
		Name:        c.Name,
		AuthType:    c.Kind,
		IsActive:    true,
		Models:      []string{},
		Priority:    0,
		NeedsReauth: needsReauth,
	}
}

// GetProviderModels handles GET /api/providers/{id}/models (PAR-UI-089).
func (h *Handlers) GetProviderModels(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	p, err := h.store.GetProvider(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}
	writeData(ctx, fasthttp.StatusOK, modelsForProvider(p))
}

// modelsForProvider returns the static catalog models for a provider, stamped
// with the provider's id. Returns an empty (non-nil) slice when none are known.
func modelsForProvider(p *store.ProviderRecord) []catalogModelDTO {
	defs := catalogModelMeta[p.Type]
	out := make([]catalogModelDTO, 0, len(defs))
	for _, m := range defs {
		m.Provider = p.ID
		out = append(out, m)
	}
	return out
}

// GetProviderSuggestedModels handles GET /api/providers/{id}/suggested-models
// (PAR-UI-089): the top N (≤5) models as a trimmed {id,name} list.
func (h *Handlers) GetProviderSuggestedModels(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	p, err := h.store.GetProvider(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "provider not found")
		return
	}
	models := modelsForProvider(p)
	limit := len(models)
	if limit > 5 {
		limit = 5
	}
	out := make([]suggestedModelDTO, 0, limit)
	for _, m := range models[:limit] {
		out = append(out, suggestedModelDTO{ID: m.ID, Name: m.Name})
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// TestProvidersBatch handles POST /api/providers/test-batch (PAR-UI-090). Under
// test it is deterministic: ok = the provider has at least one connection. No
// real outbound provider network is performed.
func (h *Handlers) TestProvidersBatch(ctx *fasthttp.RequestCtx) {
	providers, err := h.store.ListProviders()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list providers")
		return
	}
	connections, err := h.store.ListConnections()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list connections")
		return
	}
	counts := map[string]int{}
	for _, c := range connections {
		counts[c.ProviderID]++
	}
	results := make([]batchResultDTO, 0, len(providers))
	for _, p := range providers {
		ok := counts[p.ID] > 0
		results = append(results, batchResultDTO{Provider: p.ID, OK: ok, LatencyMS: 0})
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"results": results})
}
