package handlers

import (
	"context"
	"encoding/json"
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
	for _, id := range []string{"deepseek", "groq", "mistral", "minimax", "openrouter", "perplexity"} {
		if byID[id].PublicStatus != "supported" || !byID[id].RegisteredAdapter || !byID[id].PublicInference || !byID[id].DirectDispatch || !byID[id].Inference {
			t.Fatalf("%s provider = %+v, want supported catalog-routable provider", id, byID[id])
		}
		if byID[id].Quota {
			t.Fatalf("%s provider = %+v, should not claim quota support", id, byID[id])
		}
	}
	if byID["github-copilot"].PublicStatus != "auth_only" || byID["github-copilot"].Inference {
		t.Fatalf("github-copilot provider = %+v, want auth_only without inference", byID["github-copilot"])
	}
	if byID["qwen"].PublicStatus != "unsupported" || byID["qwen"].Inference {
		t.Fatalf("qwen provider = %+v, want unsupported without inference", byID["qwen"])
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
	if bedrock.PublicStatus != "adapter_only" || !bedrock.RegisteredAdapter || bedrock.PublicInference || bedrock.DirectDispatch || bedrock.Inference || bedrock.Streaming || bedrock.ModelCatalog || bedrock.ListModels || bedrock.Quota {
		t.Fatalf("bedrock provider = %+v, want registered adapter without public capabilities", bedrock)
	}
	if !strings.Contains(strings.ToLower(bedrock.Notes), "converse") || strings.Contains(strings.ToLower(bedrock.Notes), "wave 7.f") {
		t.Fatalf("bedrock notes = %q, want explicit non-Converse status without Wave 7.F TODO", bedrock.Notes)
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
