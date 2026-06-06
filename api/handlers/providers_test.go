package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	providerinfo "github.com/bloodf/g0router/internal/provider"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type handlerModelSource struct {
	models []providers.Model
	err    error
}

func (s handlerModelSource) ListModels(ctx context.Context) ([]providers.Model, error) {
	return s.models, s.err
}

func TestProvidersListKnownProviders(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, handlerModelSource{}, "")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []struct {
			ID                string   `json:"id"`
			AuthTypes         []string `json:"auth_types"`
			PublicStatus      string   `json:"public_status"`
			RegisteredAdapter bool     `json:"registered_adapter"`
			PublicInference   bool     `json:"public_inference"`
			DirectDispatch    bool     `json:"direct_dispatch"`
			Inference         bool     `json:"inference"`
			Streaming         bool     `json:"streaming"`
			ModelCatalog      bool     `json:"model_catalog"`
			ListModels        bool     `json:"list_models"`
			Quota             bool     `json:"quota"`
			Notes             string   `json:"notes"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) == 0 {
		t.Fatal("providers list should not be empty")
	}
	if strings.Contains(string(body), `"auth_types":null`) {
		t.Fatalf("providers response serialized null auth_types: %s", body)
	}
	byID := make(map[string]struct {
		PublicStatus      string
		RegisteredAdapter bool
		PublicInference   bool
		DirectDispatch    bool
		Inference         bool
		Quota             bool
	}, len(decoded.Data))
	for _, provider := range decoded.Data {
		byID[provider.ID] = struct {
			PublicStatus      string
			RegisteredAdapter bool
			PublicInference   bool
			DirectDispatch    bool
			Inference         bool
			Quota             bool
		}{
			PublicStatus:      provider.PublicStatus,
			RegisteredAdapter: provider.RegisteredAdapter,
			PublicInference:   provider.PublicInference,
			DirectDispatch:    provider.DirectDispatch,
			Inference:         provider.Inference,
			Quota:             provider.Quota,
		}
	}
	if byID["openai"].PublicStatus != "supported" || !byID["openai"].PublicInference || !byID["openai"].DirectDispatch || !byID["openai"].Inference {
		t.Fatalf("openai provider = %+v, want supported inference provider", byID["openai"])
	}
	if byID["openai"].Quota {
		t.Fatalf("openai provider = %+v, should not claim quota support until a real fetcher exists", byID["openai"])
	}
	if byID["anthropic"].Quota {
		t.Fatalf("anthropic provider = %+v, should not claim quota support until a real fetcher exists", byID["anthropic"])
	}
	for _, id := range []string{"alibaba", "azure", "bedrock", "cerebras", "cloudflare-ai-gateway", "cohere", "deepseek", "fireworks", "gemini", "github-copilot", "gitlab-duo", "groq", "huggingface", "kilo", "kimi", "litellm", "lm-studio", "mistral", "minimax", "nebius", "nvidia", "ollama", "ollama-cloud", "opencode", "openrouter", "perplexity", "qianfan", "qwen", "replicate", "together", "vercel-ai-gateway", "vertex", "vllm", "xai", "zhipu"} {
		if byID[id].PublicStatus != "supported" || !byID[id].RegisteredAdapter || !byID[id].PublicInference || !byID[id].DirectDispatch || !byID[id].Inference {
			t.Fatalf("%s provider = %+v, want supported inference provider", id, byID[id])
		}
		if id == "openrouter" {
			if !byID[id].Quota {
				t.Fatalf("%s provider = %+v, should expose real quota support", id, byID[id])
			}
		} else if byID[id].Quota {
			t.Fatalf("%s provider = %+v, should not claim quota support", id, byID[id])
		}
	}
	for _, id := range []string{"kagi", "tavily"} {
		if byID[id].PublicStatus != "auth_only" || byID[id].RegisteredAdapter || byID[id].Inference || byID[id].PublicInference || byID[id].DirectDispatch {
			t.Fatalf("%s provider = %+v, want API-key auth-only search provider", id, byID[id])
		}
	}
	matrix := providerinfo.ProviderMatrix()
	for _, got := range decoded.Data {
		entry, ok := matrix.Provider(got.ID)
		if !ok {
			t.Fatalf("provider %q missing from matrix", got.ID)
		}
		if got.Inference != entry.Inference {
			t.Fatalf("%s inference = %v, want matrix value %v", got.ID, got.Inference, entry.Inference)
		}
	}
	var bedrock struct {
		PublicStatus      string
		RegisteredAdapter bool
		PublicInference   bool
		DirectDispatch    bool
		Inference         bool
		Streaming         bool
		ModelCatalog      bool
		ListModels        bool
		Quota             bool
		Notes             string
	}
	for _, provider := range decoded.Data {
		if provider.ID == "bedrock" {
			bedrock = struct {
				PublicStatus      string
				RegisteredAdapter bool
				PublicInference   bool
				DirectDispatch    bool
				Inference         bool
				Streaming         bool
				ModelCatalog      bool
				ListModels        bool
				Quota             bool
				Notes             string
			}{
				PublicStatus:      provider.PublicStatus,
				RegisteredAdapter: provider.RegisteredAdapter,
				PublicInference:   provider.PublicInference,
				DirectDispatch:    provider.DirectDispatch,
				Inference:         provider.Inference,
				Streaming:         provider.Streaming,
				ModelCatalog:      provider.ModelCatalog,
				ListModels:        provider.ListModels,
				Quota:             provider.Quota,
				Notes:             provider.Notes,
			}
			break
		}
	}
	if bedrock.PublicStatus != "supported" || !bedrock.RegisteredAdapter || !bedrock.PublicInference || !bedrock.DirectDispatch || !bedrock.Inference || !bedrock.Streaming || !bedrock.ModelCatalog || !bedrock.ListModels || bedrock.Quota {
		t.Fatalf("bedrock provider = %+v, want supported streaming Converse catalog provider without quota", bedrock)
	}
	if !strings.Contains(strings.ToLower(bedrock.Notes), "converse") || !strings.Contains(strings.ToLower(bedrock.Notes), "catalog") || !strings.Contains(strings.ToLower(bedrock.Notes), "streaming") {
		t.Fatalf("bedrock notes = %q, want explicit Converse catalog status", bedrock.Notes)
	}
}

func TestProvidersListModelsForProvider(t *testing.T) {
	source := handlerModelSource{models: []providers.Model{
		{ID: "gpt-4o", Object: "model", Created: 1, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
		{ID: "claude-sonnet-4", Object: "model", Created: 2, OwnedBy: "anthropic", Provider: providers.ProviderAnthropic},
	}}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, source, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []providers.Model `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 1 || decoded.Data[0].ID != "gpt-4o" {
		t.Fatalf("models = %+v, want only gpt-4o", decoded.Data)
	}
}

func TestProvidersListModelsCanonicalizesProviderAlias(t *testing.T) {
	source := handlerModelSource{models: []providers.Model{
		{ID: "gpt-4o", Object: "model", Created: 1, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
		{ID: "claude-sonnet-4", Object: "model", Created: 2, OwnedBy: "anthropic", Provider: providers.ProviderAnthropic},
	}}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, source, "codex")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []providers.Model `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 1 || decoded.Data[0].ID != "gpt-4o" {
		t.Fatalf("models = %+v, want codex alias to return OpenAI models", decoded.Data)
	}
}

func TestProvidersListModelsForDynamicProvider(t *testing.T) {
	source := handlerModelSource{models: []providers.Model{
		{ID: "kimi/kimi-k2.6", Object: "model", Created: 1, OwnedBy: "kimi", Provider: providers.ProviderKimi},
		{ID: "gpt-4o", Object: "model", Created: 2, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
	}}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, source, "kimi")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []providers.Model `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 1 || decoded.Data[0].ID != "kimi/kimi-k2.6" || decoded.Data[0].OwnedBy != "kimi" {
		t.Fatalf("models = %+v, want dynamic kimi model only", decoded.Data)
	}
}

func TestProvidersListModelsRejectsAuthOnlyProvider(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Providers(ctx, handlerModelSource{}, "cursor")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "provider inference unavailable") {
		t.Fatalf("body = %s, want provider inference unavailable", body)
	}
}

func TestProviderDetail(t *testing.T) {
	s := newHandlerStore(t)

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	source := handlerModelSource{models: []providers.Model{
		{ID: "gpt-4o", Object: "model", Created: 1, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
	}}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, s, source, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data providerDetailResponse `json:"data"`
	}
	decodeJSON(t, body, &decoded)

	if decoded.Data.ID != "openai" {
		t.Fatalf("id = %q, want openai", decoded.Data.ID)
	}
	if decoded.Data.Name != "openai" {
		t.Fatalf("name = %q, want openai", decoded.Data.Name)
	}
	if decoded.Data.ConnectionCount != 1 {
		t.Fatalf("connection_count = %d, want 1", decoded.Data.ConnectionCount)
	}
	if decoded.Data.HealthStatus != "healthy" {
		t.Fatalf("health_status = %q, want healthy", decoded.Data.HealthStatus)
	}
	if len(decoded.Data.Models) != 1 || decoded.Data.Models[0].ID != "gpt-4o" {
		t.Fatalf("models = %+v, want gpt-4o", decoded.Data.Models)
	}
	if decoded.Data.MatrixInfo.ID != "openai" {
		t.Fatalf("matrix_info.id = %q, want openai", decoded.Data.MatrixInfo.ID)
	}
}

func TestProviderDetailNotFound(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, s, handlerModelSource{}, "nonexistent")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "provider not found") {
		t.Fatalf("body = %s, want provider not found", body)
	}
}

func TestProviderDetailHealthStatus(t *testing.T) {
	s := newHandlerStore(t)

	// No connections → unknown
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, s, handlerModelSource{}, "openai")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "inactive",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: false,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	// All inactive → unhealthy
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderDetail(ctx, s, handlerModelSource{}, "openai")
	})
	var decoded struct {
		Data providerDetailResponse `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if decoded.Data.HealthStatus != "unhealthy" {
		t.Fatalf("health_status = %q, want unhealthy", decoded.Data.HealthStatus)
	}
	if decoded.Data.ConnectionCount != 1 {
		t.Fatalf("connection_count = %d, want 1", decoded.Data.ConnectionCount)
	}
}

func TestProviderConnections(t *testing.T) {
	s := newHandlerStore(t)

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "anthropic",
		Name:     "anthropic-primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderConnections(ctx, s, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []connectionResponse `json:"data"`
	}
	decodeJSON(t, body, &decoded)

	if len(decoded.Data) != 1 {
		t.Fatalf("connections = %d, want 1", len(decoded.Data))
	}
	if decoded.Data[0].Provider != "openai" {
		t.Fatalf("provider = %q, want openai", decoded.Data[0].Provider)
	}
}

func TestProviderConnectionsNotFound(t *testing.T) {
	s := newHandlerStore(t)
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderConnections(ctx, s, "nonexistent")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "provider not found") {
		t.Fatalf("body = %s, want provider not found", body)
	}
}

type fakeProviderAdapter struct {
	models []providers.Model
	err    error
}

func (f fakeProviderAdapter) Name() providers.ModelProvider {
	return "openai"
}

func (f fakeProviderAdapter) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, nil
}

func (f fakeProviderAdapter) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f fakeProviderAdapter) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return f.models, f.err
}

type fakeAdapterSource struct {
	provider providers.Provider
	ok       bool
}

func (f fakeAdapterSource) GetProvider(name providers.ModelProvider) (providers.Provider, bool) {
	return f.provider, f.ok
}

func TestProviderSuggestedModels(t *testing.T) {
	s := newHandlerStore(t)

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	adapter := fakeProviderAdapter{
		models: []providers.Model{
			{ID: "gpt-4o", Object: "model", Created: 1, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
			{ID: "gpt-4o-mini", Object: "model", Created: 2, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
		},
	}
	source := fakeAdapterSource{provider: adapter, ok: true}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, s, source, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []suggestedModelResponse `json:"data"`
	}
	decodeJSON(t, body, &decoded)

	if len(decoded.Data) != 2 {
		t.Fatalf("models = %d, want 2", len(decoded.Data))
	}
	if decoded.Data[0].ID != "gpt-4o" {
		t.Fatalf("model[0].id = %q, want gpt-4o", decoded.Data[0].ID)
	}
}

func TestProviderSuggestedModelsNoConnections(t *testing.T) {
	s := newHandlerStore(t)
	adapter := fakeProviderAdapter{}
	source := fakeAdapterSource{provider: adapter, ok: true}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, s, source, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "no active connections") {
		t.Fatalf("body = %s, want no active connections", body)
	}
}

func TestProviderSuggestedModelsUpstreamError(t *testing.T) {
	s := newHandlerStore(t)

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	adapter := fakeProviderAdapter{err: errors.New("upstream error")}
	source := fakeAdapterSource{provider: adapter, ok: true}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, s, source, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), body)
	}
	if !strings.Contains(string(body), "upstream model list failed") {
		t.Fatalf("body = %s, want upstream model list failed", body)
	}
}

func TestProviderSuggestedModelsUnsupported(t *testing.T) {
	s := newHandlerStore(t)

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	adapter := fakeProviderAdapter{err: providers.ErrListModelsUnsupported}
	source := fakeAdapterSource{provider: adapter, ok: true}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		ProviderSuggestedModels(ctx, s, source, "openai")
	})

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []suggestedModelResponse `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 0 {
		t.Fatalf("models = %d, want 0", len(decoded.Data))
	}
}

func newHandlerStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return s
}

func runHandler(t *testing.T, method, body string, handler func(*fasthttp.RequestCtx)) (*fasthttp.RequestCtx, []byte) {
	t.Helper()

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	if body != "" {
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBodyString(body)
	}
	handler(&ctx)
	return &ctx, ctx.Response.Body()
}

func decodeJSON(t *testing.T, body []byte, dest any) {
	t.Helper()

	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("unmarshal response: %v; body=%s", err, body)
	}
}
