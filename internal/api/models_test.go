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

// TestModelsGetByID verifies GET /v1/models/:id returns a single model entry
// matching the requested ID.
func TestModelsGetByID(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models/deepseek-chat")
	ctx.SetUserValue("id", "deepseek-chat")
	h.Get(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}

	var resp struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID != "deepseek-chat" {
		t.Errorf("id = %q, want deepseek-chat", resp.ID)
	}
	if resp.Object != "model" {
		t.Errorf("object = %q, want model", resp.Object)
	}
	if resp.OwnedBy != "deepseek" {
		t.Errorf("owned_by = %q, want deepseek", resp.OwnedBy)
	}
}

// TestModelsGetUnknown404 verifies GET /v1/models/:id returns a 404 JSON
// envelope when the requested model does not exist.
func TestModelsGetUnknown404(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models/nonexistent-model")
	ctx.SetUserValue("id", "nonexistent-model")
	h.Get(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Errorf("status = %d, want 404", ctx.Response.StatusCode())
	}

	var resp struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Error == nil || resp.Error.Message == "" {
		t.Errorf("expected error envelope, got %s", string(ctx.Response.Body()))
	}
}

// fakeDisabledChecker implements DisabledChecker for testing.
type fakeDisabledChecker struct {
	disabled map[string]map[string]bool // providerAlias → set of modelIDs
}

func (f *fakeDisabledChecker) IsDisabled(providerAlias, modelID string) (bool, error) {
	if models, ok := f.disabled[providerAlias]; ok {
		return models[modelID], nil
	}
	return false, nil
}

func TestModelsListExcludesDisabled(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetDisabledChecker(&fakeDisabledChecker{
		disabled: map[string]map[string]bool{
			"deepseek": {"deepseek-chat": true},
		},
	})

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
		t.Fatalf("unmarshal: %v", err)
	}

	for _, m := range resp.Data {
		if m.ID == "deepseek-chat" {
			t.Fatal("deepseek-chat should be excluded (disabled)")
		}
	}

	// Other deepseek models must still appear.
	found := false
	for _, m := range resp.Data {
		if m.ID == "deepseek-reasoner" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("deepseek-reasoner should still appear")
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
