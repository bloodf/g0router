package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
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
		{"deepseek-v4-pro", "deepseek"},
		{"deepseek-reasoner", "deepseek"},
		{"llama-3.3-70b-versatile", "groq"},
		{"grok-4-fast-reasoning", "xai"},
		{"grok-3", "xai"},
		{"gpt-oss:120b", "ollama"},
		{"glm-4.7-flash", "ollama"},
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
	ctx.Request.SetRequestURI("/v1/models/deepseek-v4-pro")
	ctx.SetUserValue("id", "deepseek-v4-pro")
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
	if resp.ID != "deepseek-v4-pro" {
		t.Errorf("id = %q, want deepseek-v4-pro", resp.ID)
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
			"deepseek": {"deepseek-v4-pro": true},
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
		if m.ID == "deepseek-v4-pro" {
			t.Fatal("deepseek-v4-pro should be excluded (disabled)")
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

// fakeComboLister implements ComboLister for testing.
type fakeComboLister struct {
	names []string
}

func (f *fakeComboLister) ListComboNames() ([]string, error) {
	return f.names, nil
}

// fakeCustomModelLister implements CustomModelLister for testing.
type fakeCustomModelLister struct {
	models []CustomModel
	err    error
}

func (f *fakeCustomModelLister) ListCustomModels() ([]CustomModel, error) {
	return f.models, f.err
}

// fakeAliasModelLister implements AliasModelLister for testing.
type fakeAliasModelLister struct {
	names []string
	err   error
}

func (f *fakeAliasModelLister) ListAliasNames() ([]string, error) {
	return f.names, f.err
}

// fakeSubConfigReader implements SubConfigModelReader for testing.
type fakeSubConfigReader struct {
	models []SubConfigModel
	err    error
}

func (f *fakeSubConfigReader) ListSubConfigModels() ([]SubConfigModel, error) {
	return f.models, f.err
}

func TestModelsListCombosFirst(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetComboLister(&fakeComboLister{
		names: []string{"fast-combo", "smart-combo"},
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
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) < 2 {
		t.Fatalf("expected at least 2 entries, got %d", len(resp.Data))
	}
	// First two entries must be combo entries in list order.
	if resp.Data[0].ID != "fast-combo" || resp.Data[0].OwnedBy != "combo" {
		t.Errorf("data[0] = {%q, %q}, want {fast-combo, combo}", resp.Data[0].ID, resp.Data[0].OwnedBy)
	}
	if resp.Data[1].ID != "smart-combo" || resp.Data[1].OwnedBy != "combo" {
		t.Errorf("data[1] = {%q, %q}, want {smart-combo, combo}", resp.Data[1].ID, resp.Data[1].OwnedBy)
	}
	// Remaining entries must be provider models (not combo-owned).
	for i, m := range resp.Data[2:] {
		if m.OwnedBy == "combo" {
			t.Errorf("data[%d] = {%q, combo}, expected provider model", i+2, m.ID)
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

func TestModelsByKind(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)

	tests := []struct {
		kind        string
		wantStatus  int
		wantNonEmpty bool // if true, at least one entry expected
		wantEmpty   bool  // valid kind but no catalog entries
	}{
		{kind: "image", wantStatus: 200, wantNonEmpty: true},
		{kind: "tts", wantStatus: 200, wantNonEmpty: true},
		{kind: "stt", wantStatus: 200, wantNonEmpty: true},
		{kind: "embedding", wantStatus: 200, wantNonEmpty: true},
		// image-to-text and web are valid kinds but have no catalog entries → empty list, not 404.
		{kind: "image-to-text", wantStatus: 200, wantEmpty: true},
		{kind: "web", wantStatus: 200, wantEmpty: true},
		// Unknown kind → 404.
		{kind: "unknown-kind", wantStatus: 404},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.Request.Header.SetMethod(http.MethodGet)
			ctx.Request.SetRequestURI("/v1/models/" + tt.kind)
			ctx.SetUserValue("kind", tt.kind)
			h.GetByKind(&ctx)

			if ctx.Response.StatusCode() != tt.wantStatus {
				t.Fatalf("status = %d, want %d", ctx.Response.StatusCode(), tt.wantStatus)
			}
			if tt.wantStatus != 200 {
				return
			}

			var resp struct {
				Object string `json:"object"`
				Data   []struct {
					ID      string `json:"id"`
					OwnedBy string `json:"owned_by"`
				} `json:"data"`
			}
			if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if resp.Object != "list" {
				t.Errorf("object = %q, want list", resp.Object)
			}
			if tt.wantNonEmpty && len(resp.Data) == 0 {
				t.Errorf("expected non-empty list for kind %q", tt.kind)
			}
			if tt.wantEmpty && len(resp.Data) != 0 {
				t.Errorf("expected empty list for kind %q, got %d entries", tt.kind, len(resp.Data))
			}
			// Each returned entry must have a non-empty provider and a catalog Type
			// that matches the requested kind's type set (PAR-ROUTE-037).
			wantTypes := kindSlugMap[tt.kind]
			typeSet := make(map[string]bool, len(wantTypes))
			for _, typ := range wantTypes {
				typeSet[typ] = true
			}
			for _, m := range resp.Data {
				if m.OwnedBy == "" {
					t.Errorf("entry %q has empty owned_by", m.ID)
				}
				if len(wantTypes) > 0 {
					// Look up this model in the catalog to verify its type.
					found := false
					for _, cm := range catalog.ModelsFor(m.OwnedBy) {
						if cm.ID == m.ID {
							actualType := cm.Type
							if actualType == "" {
								actualType = "llm"
							}
							if !typeSet[actualType] {
								t.Errorf("model %q (provider=%q) has type %q, not in %v", m.ID, m.OwnedBy, actualType, wantTypes)
							}
							found = true
							break
						}
					}
					if !found {
						t.Errorf("model %q (provider=%q) not found in catalog", m.ID, m.OwnedBy)
					}
				}
			}
		})
	}
}

// TestModelTestRoutesByKind verifies PAR-ROUTE-038: GetTestByKind returns the correct
// API endpoint for each model kind, and 404 for unknown kinds.
func TestModelTestRoutesByKind(t *testing.T) {
	h := &ModelsHandler{}

	tests := []struct {
		kind         string
		wantEndpoint string
		want404      bool
	}{
		{"embedding", "/v1/embeddings", false},
		{"image", "/v1/images/generations", false},
		{"image-to-text", "/v1/images/generations", false},
		{"tts", "/v1/audio/speech", false},
		{"stt", "/v1/audio/transcriptions", false},
		{"web", "/v1/chat/completions", false},
		{"llm", "/v1/chat/completions", false},
		{"unknown-kind", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.SetUserValue("kind", tt.kind)
			h.GetTestByKind(&ctx)

			if tt.want404 {
				if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
					t.Errorf("status = %d, want 404", ctx.Response.StatusCode())
				}
				return
			}
			if ctx.Response.StatusCode() != fasthttp.StatusOK {
				t.Errorf("status = %d, want 200", ctx.Response.StatusCode())
			}
			var route ModelTestRoute
			if err := json.Unmarshal(ctx.Response.Body(), &route); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if route.Endpoint != tt.wantEndpoint {
				t.Errorf("endpoint = %q, want %q", route.Endpoint, tt.wantEndpoint)
			}
			if route.Method != "POST" {
				t.Errorf("method = %q, want POST", route.Method)
			}
			if route.Kind != tt.kind {
				t.Errorf("kind = %q, want %q", route.Kind, tt.kind)
			}
		})
	}
}

// TestModelsList_MergesCustomModels verifies PAR-ROUTE-057: a custom model is
// appended after the catalog section with owned_by set to its provider field.
// Ref order follows route.js:358 (catalog IDs first, then custom IDs).
func TestModelsList_MergesCustomModels(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetCustomModelLister(&fakeCustomModelLister{
		models: []CustomModel{{ID: "my-custom", Provider: "openai", Type: "llm"}},
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
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	found := false
	catalogIdx := -1
	customIdx := -1
	for i, m := range resp.Data {
		if m.ID == "my-custom" {
			found = true
			customIdx = i
			if m.OwnedBy != "openai" {
				t.Errorf("my-custom owned_by = %q, want openai (route.js:318-321)", m.OwnedBy)
			}
		}
		if m.ID == "gpt-4o" && catalogIdx < 0 {
			catalogIdx = i
		}
	}
	if !found {
		t.Fatal("my-custom not found in response")
	}
	if catalogIdx >= 0 && customIdx <= catalogIdx {
		t.Errorf("custom position %d not after catalog %d (route.js:358)", customIdx, catalogIdx)
	}
}

// TestModelsList_MergesAliasModels verifies PAR-ROUTE-057: an alias name is
// exposed as a model entry with owned_by "alias", positioned after custom entries.
func TestModelsList_MergesAliasModels(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetAliasModelLister(&fakeAliasModelLister{
		names: []string{"fast"},
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
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	found := false
	for _, m := range resp.Data {
		if m.ID == "fast" {
			found = true
			if m.OwnedBy != "alias" {
				t.Errorf("fast owned_by = %q, want alias", m.OwnedBy)
			}
		}
	}
	if !found {
		t.Fatal("alias 'fast' not found")
	}
}

// TestModelsList_DedupCustomVsCatalog verifies PAR-ROUTE-057 dedup direction:
// a custom ID that collides with a catalog ID is skipped because the seen set is
// seeded with catalog IDs first (route.js:358 Set([...modelIds, ...customModelIds, ...aliasModelIds])).
func TestModelsList_DedupCustomVsCatalog(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetCustomModelLister(&fakeCustomModelLister{
		models: []CustomModel{{ID: "deepseek-v4-pro", Provider: "openai", Type: "llm"}},
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	var resp struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	count := 0
	var survivor string
	for _, m := range resp.Data {
		if m.ID == "deepseek-v4-pro" {
			count++
			survivor = m.OwnedBy
		}
	}
	if count != 1 {
		t.Fatalf("deepseek-v4-pro appears %d times, want 1", count)
	}
	if survivor != "deepseek" {
		t.Errorf("survivor owned_by = %q, want deepseek (catalog wins per route.js:358)", survivor)
	}
}

// TestModelsList_CustomListerError verifies that a failing custom lister returns
// a 500 server_error, matching the combo-lister error path.
func TestModelsList_CustomListerError(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetCustomModelLister(&fakeCustomModelLister{
		err: errors.New("custom lister failed"),
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", ctx.Response.StatusCode())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "server_error") {
		t.Errorf("body = %q, want server_error envelope", body)
	}
}

// TestModelsList_NilListersUnchanged verifies that when no custom/alias listers
// are wired the response contains only combos and catalog models.
func TestModelsList_NilListersUnchanged(t *testing.T) {
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
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, m := range resp.Data {
		if m.OwnedBy == "alias" {
			t.Errorf("unexpected alias entry %q with no alias lister", m.ID)
		}
	}

	// Spot-check that a known catalog model is still present.
	found := false
	for _, m := range resp.Data {
		if m.ID == "deepseek-chat" {
			found = true
			break
		}
	}
	if !found {
		t.Error("deepseek-chat missing from catalog-only response")
	}
}

// TestModelsList_IncludesSubConfigModels verifies PAR-ROUTE-058: TTS and embedding
// models from connection metadata are appended after alias entries with owned_by set
// to the connection's provider ID (route.js:364-383).
func TestModelsList_IncludesSubConfigModels(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetSubConfigModelReader(&fakeSubConfigReader{
		models: []SubConfigModel{
			{ID: "tts-1", Kind: "tts", ProviderID: "prov-1"},
			{ID: "emb-1", Kind: "embedding", ProviderID: "prov-1"},
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
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	foundTTS, foundEmb := false, false
	for _, m := range resp.Data {
		if m.ID == "tts-1" {
			foundTTS = true
			if m.OwnedBy != "prov-1" {
				t.Errorf("tts-1 owned_by = %q, want prov-1 (route.js:375-379)", m.OwnedBy)
			}
		}
		if m.ID == "emb-1" {
			foundEmb = true
			if m.OwnedBy != "prov-1" {
				t.Errorf("emb-1 owned_by = %q, want prov-1", m.OwnedBy)
			}
		}
	}
	if !foundTTS {
		t.Error("tts-1 not found")
	}
	if !foundEmb {
		t.Error("emb-1 not found")
	}
}

// TestModelsList_SubConfigDedup verifies that a sub-config ID colliding with a
// catalog ID is skipped because the catalog ID is already in the seen set.
func TestModelsList_SubConfigDedup(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetSubConfigModelReader(&fakeSubConfigReader{
		models: []SubConfigModel{
			{ID: "deepseek-v4-pro", Kind: "tts", ProviderID: "prov-1"},
		},
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	var resp struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	count := 0
	var survivor string
	for _, m := range resp.Data {
		if m.ID == "deepseek-v4-pro" {
			count++
			survivor = m.OwnedBy
		}
	}
	if count != 1 {
		t.Fatalf("deepseek-v4-pro appears %d times, want 1", count)
	}
	if survivor != "deepseek" {
		t.Errorf("survivor owned_by = %q, want deepseek (catalog wins per route.js:358)", survivor)
	}
}

// TestModelsList_SubConfigReaderError verifies that a failing sub-config reader
// returns a 500 server_error.
func TestModelsList_SubConfigReaderError(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	h := NewModelsHandler(router)
	h.SetSubConfigModelReader(&fakeSubConfigReader{
		err: errors.New("sub-config reader failed"),
	})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/models")
	h.List(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", ctx.Response.StatusCode())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "server_error") {
		t.Errorf("body = %q, want server_error envelope", body)
	}
}
