package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/translation"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// fakeComboDispatcherForRoutes is an api.ComboDispatcher that reports a single
// combo name and invokes the handler callback with a model routable to a local
// test server.
type fakeComboDispatcherForRoutes struct{}

var _ api.ComboDispatcher = (*fakeComboDispatcherForRoutes)(nil)

func (f *fakeComboDispatcherForRoutes) IsCombo(name string) bool { return name == "combomodel" }

func (f *fakeComboDispatcherForRoutes) ExecuteCombo(name string, fn func(model, connID, credential string) (inference.Verdict, error)) error {
	_, err := fn("testprov/canned-model", "conn-1", "key-1")
	return err
}

func TestResponsesRouteRegistered(t *testing.T) {
	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterOpenAIRoutes(r, inference.NewRouter(translation.NewRegistry()), nil, nil, nil, nil, nil, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4"}`))
	r.Handler(&ctx)

	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/responses returned 404 — route not registered")
	}
}

// TestInputTokensRouteRegistered verifies POST /v1/responses/input_tokens is
// wired and coexists with the static /v1/responses route (PAR-BF-OAI-004).
func TestInputTokensRouteRegistered(t *testing.T) {
	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterOpenAIRoutes(r, inference.NewRouter(translation.NewRegistry()), nil, nil, nil, nil, nil, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/v1/responses/input_tokens")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4","input":[{"role":"user","content":"hi"}]}`))
	r.Handler(&ctx)

	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/responses/input_tokens returned 404 — route not registered")
	}
}

func TestRegisterOpenAIRoutesPlumbsComboDispatcher(t *testing.T) {
	// Local stub that returns a canned chat completion.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"canned","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"canned-content"}}]}`))
	}))
	defer srv.Close()

	// Inject a test provider whose base URL points at the local stub.
	orig, ok := catalog.Providers["testprov"]
	catalog.Providers["testprov"] = catalog.ProviderConfig{
		Name:    "testprov",
		BaseURL: srv.URL,
		Format:  "openai",
		NoAuth:  true,
	}
	if ok {
		t.Cleanup(func() { catalog.Providers["testprov"] = orig })
	} else {
		t.Cleanup(func() { delete(catalog.Providers, "testprov") })
	}

	router := inference.NewRouter(translation.NewRegistry())

	r := httprouter.New()
	r.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterOpenAIRoutes(r, router, nil, nil, &fakeComboDispatcherForRoutes{}, nil, nil, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/v1/chat/completions")
	ctx.Request.SetBody([]byte(`{"model":"combomodel","messages":[{"role":"user","content":"hi"}]}`))
	r.Handler(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("combo dispatcher request status = %d, want 200: %s", ctx.Response.StatusCode(), string(ctx.Response.Body()))
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, "canned-content") {
		t.Errorf("response body = %q, want canned-content", body)
	}

	// Nil-dispatcher control: the same model is unknown, so the handler resolves
	// to an error instead of reaching the combo path.
	r2 := httprouter.New()
	r2.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("not found")
	}
	RegisterOpenAIRoutes(r2, inference.NewRouter(translation.NewRegistry()), nil, nil, nil, nil, nil, nil)

	var ctx2 fasthttp.RequestCtx
	ctx2.Request.Header.SetMethod("POST")
	ctx2.Request.SetRequestURI("/v1/chat/completions")
	ctx2.Request.SetBody([]byte(`{"model":"combomodel","messages":[{"role":"user","content":"hi"}]}`))
	r2.Handler(&ctx2)

	if ctx2.Response.StatusCode() == fasthttp.StatusOK {
		t.Fatalf("nil-dispatcher control status = 200, want error (model unknown)")
	}
}

// createTestProvider inserts a provider and returns its generated ID.
func createTestProvider(t *testing.T, st *store.Store, name string) string {
	t.Helper()
	p := &store.ProviderRecord{
		Name:    name,
		Type:    name,
		BaseURL: "http://localhost",
		Enabled: true,
	}
	if err := st.CreateProvider(p); err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	return p.ID
}

// TestCustomModelsAdapter_ParsesSetting verifies the adapter reads the customModels
// setting and filters out entries with empty IDs or non-LLM types (route.js:318-319).
func TestCustomModelsAdapter_ParsesSetting(t *testing.T) {
	st := newTestStore(t)
	if err := st.SetSetting("customModels", `[{"id":"custom-1","provider":"openai","type":"llm"},{"id":"custom-2","provider":"anthropic","type":"tts"}]`); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	adapter := customModelsAdapter{st: st}
	got, err := adapter.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels: %v", err)
	}
	if len(got) != 1 || got[0].ID != "custom-1" || got[0].Provider != "openai" || got[0].Type != "llm" {
		t.Errorf("got %+v, want one custom-1/openai/llm", got)
	}
}

// TestCustomModelsAdapter_MissingSettingEmpty verifies ErrNotFound maps to an empty list.
func TestCustomModelsAdapter_MissingSettingEmpty(t *testing.T) {
	st := newTestStore(t)
	adapter := customModelsAdapter{st: st}
	got, err := adapter.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d models, want 0", len(got))
	}
}

// TestCustomModelsAdapter_MalformedJSON verifies malformed JSON is treated as an
// empty list, matching route.js helpers/jsonCol.js behavior.
func TestCustomModelsAdapter_MalformedJSON(t *testing.T) {
	st := newTestStore(t)
	if err := st.SetSetting("customModels", `[not json`); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	adapter := customModelsAdapter{st: st}
	got, err := adapter.ListCustomModels()
	if err != nil {
		t.Fatalf("ListCustomModels: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d models, want 0 for malformed JSON", len(got))
	}
}

// TestAliasModelsAdapter_ListsNames verifies the alias adapter returns alias names.
func TestAliasModelsAdapter_ListsNames(t *testing.T) {
	st := newTestStore(t)
	if err := st.CreateAlias("fast", "openai/gpt-4"); err != nil {
		t.Fatalf("CreateAlias: %v", err)
	}
	if err := st.CreateAlias("slow", "anthropic/claude-opus-4"); err != nil {
		t.Fatalf("CreateAlias: %v", err)
	}

	adapter := aliasModelsAdapter{st: st}
	got, err := adapter.ListAliasNames()
	if err != nil {
		t.Fatalf("ListAliasNames: %v", err)
	}
	want := []string{"fast", "slow"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestSubConfigModelsAdapter_ParsesConnectionMetadata verifies TTS and embedding
// models are read from providerSpecificData.*.models (route.js:364-383).
func TestSubConfigModelsAdapter_ParsesConnectionMetadata(t *testing.T) {
	st := newTestStore(t)
	provID := createTestProvider(t, st, "openai")
	conn := &store.Connection{
		ProviderID: provID,
		Name:       "test-conn",
		Kind:       "api_key",
		Secret:     "secret",
		Metadata:   `{"providerSpecificData":{"ttsConfig":{"models":[{"id":"tts-1"}]},"embeddingConfig":{"models":[{"id":"emb-1"}]}}}`,
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	adapter := subConfigModelsAdapter{st: st}
	got, err := adapter.ListSubConfigModels()
	if err != nil {
		t.Fatalf("ListSubConfigModels: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d models, want 2", len(got))
	}
	if got[0].ID != "tts-1" || got[0].Kind != "tts" || got[0].ProviderID != provID {
		t.Errorf("first = %+v, want tts-1/tts/%s", got[0], provID)
	}
	if got[1].ID != "emb-1" || got[1].Kind != "embedding" || got[1].ProviderID != provID {
		t.Errorf("second = %+v, want emb-1/embedding/%s", got[1], provID)
	}
}

// TestSubConfigModelsAdapter_UnparseableMetadata verifies unparseable metadata on
// one connection contributes zero entries without failing the whole list.
func TestSubConfigModelsAdapter_UnparseableMetadata(t *testing.T) {
	st := newTestStore(t)
	provID := createTestProvider(t, st, "openai")
	conn := &store.Connection{
		ProviderID: provID,
		Name:       "bad-conn",
		Kind:       "api_key",
		Secret:     "secret",
		Metadata:   `not-json`,
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	adapter := subConfigModelsAdapter{st: st}
	got, err := adapter.ListSubConfigModels()
	if err != nil {
		t.Fatalf("ListSubConfigModels: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d models, want 0 for unparseable metadata", len(got))
	}
}

// TestStoreVKToAPI_MapsKeyIDs verifies the persisted KeyIDs field is mapped into
// api.VKProviderConfig.KeyIDs.
func TestStoreVKToAPI_MapsKeyIDs(t *testing.T) {
	vk := &store.VirtualKey{
		VirtualKey: schemas.VirtualKey{
			ProviderConfigs: []schemas.ProviderConfig{
				{Provider: "openai", AllowedModels: []string{"gpt-4o"}, KeyIDs: []string{"conn-1", "conn-2"}},
			},
		},
		Key:      "g0vk-test",
		IsActive: true,
	}
	info := storeVKToAPI(vk)
	if len(info.Configs) != 1 {
		t.Fatalf("got %d configs, want 1", len(info.Configs))
	}
	want := []string{"conn-1", "conn-2"}
	if len(info.Configs[0].KeyIDs) != len(want) || info.Configs[0].KeyIDs[0] != want[0] || info.Configs[0].KeyIDs[1] != want[1] {
		t.Errorf("KeyIDs = %v, want %v", info.Configs[0].KeyIDs, want)
	}
}

// TestVKPinnedSelector_PinsEligibleKeyID verifies a KeyID that matches an eligible
// connection is returned with its credential.
func TestVKPinnedSelector_PinsEligibleKeyID(t *testing.T) {
	st := newTestStore(t)
	provID := createTestProvider(t, st, "openai")
	conn := &store.Connection{
		ProviderID: provID,
		Name:       "pinned",
		Kind:       "api_key",
		Secret:     "secret-2",
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	selector := &vkPinnedSelector{
		st:     st,
		engine: inference.NewSelectionEngine(st, st, nil, time.Now),
		rr:     make(map[string]int),
	}
	connID, cred, ok := selector.ResolvePinned(provID, "gpt-4o", []string{conn.ID})
	if !ok {
		t.Fatal("ResolvePinned returned ok=false")
	}
	if connID != conn.ID {
		t.Errorf("connID = %q, want %q", connID, conn.ID)
	}
	if cred != "secret-2" {
		t.Errorf("cred = %q, want secret-2", cred)
	}
}

// TestVKPinnedSelector_RoundRobinAcrossKeyIDs verifies the cursor rotates across
// multiple eligible KeyIDs.
func TestVKPinnedSelector_RoundRobinAcrossKeyIDs(t *testing.T) {
	st := newTestStore(t)
	provID := createTestProvider(t, st, "openai")
	var ids []string
	for _, name := range []string{"conn-a", "conn-b"} {
		conn := &store.Connection{
			ProviderID: provID,
			Name:       name,
			Kind:       "api_key",
			Secret:     "secret-" + name,
		}
		if err := st.CreateConnection(conn); err != nil {
			t.Fatalf("CreateConnection %s: %v", name, err)
		}
		ids = append(ids, conn.ID)
	}

	selector := &vkPinnedSelector{
		st:     st,
		engine: inference.NewSelectionEngine(st, st, nil, time.Now),
		rr:     make(map[string]int),
	}
	first, _, _ := selector.ResolvePinned(provID, "gpt-4o", ids)
	second, _, _ := selector.ResolvePinned(provID, "gpt-4o", ids)
	third, _, _ := selector.ResolvePinned(provID, "gpt-4o", ids)
	if first != ids[0] || second != ids[1] || third != ids[0] {
		t.Errorf("round-robin = %s, %s, %s, want %s, %s, %s", first, second, third, ids[0], ids[1], ids[0])
	}
}

// TestVKPinnedSelector_FallbackWhenAllIneligible verifies ok=false when every
// pinned KeyID is locked for the model.
func TestVKPinnedSelector_FallbackWhenAllIneligible(t *testing.T) {
	st := newTestStore(t)
	provID := createTestProvider(t, st, "openai")
	conn := &store.Connection{
		ProviderID: provID,
		Name:       "locked",
		Kind:       "api_key",
		Secret:     "secret-2",
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := st.LockModel(conn.ID, provID, "gpt-4o", time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatalf("LockModel: %v", err)
	}

	selector := &vkPinnedSelector{
		st:     st,
		engine: inference.NewSelectionEngine(st, st, nil, time.Now),
		rr:     make(map[string]int),
	}
	_, _, ok := selector.ResolvePinned(provID, "gpt-4o", []string{conn.ID})
	if ok {
		t.Error("ResolvePinned returned ok=true for locked connection")
	}
}
