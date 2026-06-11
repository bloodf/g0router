package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// TestModelsHandlerMarshalFailureFallsBackTo500 verifies that when the
// response marshal seam fails, the models handler eventually writes a 500
// status (AUD-011/012). The provider will fail with a network error
// in this test environment, which routes through writeError — writeError
// then exercises the same failing jsonMarshal seam and falls back to a
// plain-text 500 per the AUD-009–012 acceptance contract.
func TestModelsHandlerMarshalFailureFallsBackTo500(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusInternalServerError)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want %q", got, "internal error")
	}
}

func TestListModelsAggregatesCatalog(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	var resp struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Object != "list" {
		t.Errorf("object = %q, want list", resp.Object)
	}

	// Build map of model ID -> OwnedBy.
	ownedBy := make(map[string]string)
	for _, m := range resp.Data {
		ownedBy[m.ID] = m.OwnedBy
	}

	// Spot-check expected models from deepseek, groq, xai, ollama.
	checks := []struct {
		model     string
		wantOwner string
	}{
		{"deepseek-chat", "deepseek"},
		{"deepseek-reasoner", "deepseek"},
		{"llama-3.3-70b-versatile", "groq"},
		{"grok-4", "xai"},
		{"grok-3", "xai"},
		{"gpt-oss:120b", "ollama"},
		{"kimi-k2.5", "ollama"},
	}
	for _, c := range checks {
		got, ok := ownedBy[c.model]
		if !ok {
			t.Errorf("model %q not found in response", c.model)
			continue
		}
		if got != c.wantOwner {
			t.Errorf("model %q owned_by = %q, want %q", c.model, got, c.wantOwner)
		}
	}
}

func TestListModelsDeterministicOrder(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	ids := make([]string, len(resp.Data))
	for i, m := range resp.Data {
		ids[i] = m.ID
	}
	if !sort.StringsAreSorted(ids) {
		t.Errorf("model IDs are not sorted: %v", ids)
	}
}
