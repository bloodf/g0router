package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeAdminModelsEngine struct {
	models []providers.Model
	err    error
}

func (f *fakeAdminModelsEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, nil
}

func (f *fakeAdminModelsEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f *fakeAdminModelsEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return f.models, f.err
}

type fakeAdminModelsStore struct {
	*store.Store
	disabledModels map[string]bool
	customModels   map[string]bool
	pricing        map[string]store.PricingOverride
}

func (f *fakeAdminModelsStore) IsModelDisabled(provider, model string) (bool, error) {
	return f.disabledModels[provider+"/"+model], nil
}

func (f *fakeAdminModelsStore) ListCustomModels() ([]store.CustomModel, error) {
	return f.Store.ListCustomModels()
}

func (f *fakeAdminModelsStore) GetPricingOverride(provider, model string) (store.PricingOverride, error) {
	po, ok := f.pricing[provider+"/"+model]
	if !ok {
		return store.PricingOverride{}, store.ErrNotFound
	}
	return po, nil
}

func TestAdminModelsReturnsMappedModels(t *testing.T) {
	s := newHandlerStore(t)

	engine := &fakeAdminModelsEngine{
		models: []providers.Model{
			{ID: "gpt-4o", Provider: providers.ProviderOpenAI, Object: "model", Created: 1234567890, OwnedBy: "openai"},
			{ID: "claude-sonnet-4", Provider: providers.ProviderAnthropic, Object: "model", Created: 1234567890, OwnedBy: "anthropic", IsCustom: true},
		},
	}

	fakeStore := &fakeAdminModelsStore{Store: s}
	fakeStore.pricing = map[string]store.PricingOverride{
		"openai/gpt-4o": {Provider: "openai", Model: "gpt-4o", InputCostPerToken: 2.5e-6, OutputCostPerToken: 10.0e-6},
	}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AdminModels(ctx, engine, fakeStore)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []struct {
			ID            string  `json:"id"`
			Provider      string  `json:"provider"`
			Name          string  `json:"name"`
			InputCost     float64 `json:"input_cost"`
			OutputCost    float64 `json:"output_cost"`
			ContextWindow int     `json:"context_window"`
			IsDisabled    bool    `json:"is_disabled"`
			IsCustom      bool    `json:"is_custom"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 2 {
		t.Fatalf("len = %d, want 2", len(decoded.Data))
	}

	first := decoded.Data[0]
	if first.ID != "openai/gpt-4o" {
		t.Errorf("id = %q, want openai/gpt-4o", first.ID)
	}
	if first.Provider != "openai" {
		t.Errorf("provider = %q, want openai", first.Provider)
	}
	if first.Name != "gpt-4o" {
		t.Errorf("name = %q, want gpt-4o", first.Name)
	}
	if first.InputCost != 2.5e-6 {
		t.Errorf("input_cost = %v, want 2.5e-6", first.InputCost)
	}
	if first.OutputCost != 10.0e-6 {
		t.Errorf("output_cost = %v, want 10.0e-6", first.OutputCost)
	}
	if first.IsCustom {
		t.Error("is_custom should be false")
	}
	if first.IsDisabled {
		t.Error("is_disabled should be false")
	}

	second := decoded.Data[1]
	if second.ID != "anthropic/claude-sonnet-4" {
		t.Errorf("id = %q, want anthropic/claude-sonnet-4", second.ID)
	}
	if !second.IsCustom {
		t.Error("is_custom should be true")
	}
}

func TestAdminModelsMarksDisabled(t *testing.T) {
	s := newHandlerStore(t)

	engine := &fakeAdminModelsEngine{
		models: []providers.Model{
			{ID: "gpt-4o", Provider: providers.ProviderOpenAI},
		},
	}

	fakeStore := &fakeAdminModelsStore{Store: s}
	fakeStore.disabledModels = map[string]bool{"openai/gpt-4o": true}

	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AdminModels(ctx, engine, fakeStore)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []struct {
			IsDisabled bool `json:"is_disabled"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 1 {
		t.Fatalf("len = %d, want 1", len(decoded.Data))
	}
	if !decoded.Data[0].IsDisabled {
		t.Error("is_disabled should be true")
	}
}

func TestAdminModelsNilEngineReturns503(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AdminModels(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestAdminModelsEngineErrorReturns500(t *testing.T) {
	engine := &fakeAdminModelsEngine{err: errors.New("boom")}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AdminModels(ctx, engine, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", ctx.Response.StatusCode())
	}
}

func TestAdminModelsEmptyList(t *testing.T) {
	engine := &fakeAdminModelsEngine{models: []providers.Model{}}
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		AdminModels(ctx, engine, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}
	var decoded struct {
		Data []struct{} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) != 0 {
		t.Fatalf("len = %d, want 0", len(decoded.Data))
	}
}
