package handlers

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

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
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) == 0 {
		t.Fatal("providers list should not be empty")
	}
	if decoded.Data[0].ID != "anthropic" {
		t.Fatalf("first provider = %q, want anthropic", decoded.Data[0].ID)
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
