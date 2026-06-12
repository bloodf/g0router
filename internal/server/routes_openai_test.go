package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/providers/catalog"
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
	RegisterOpenAIRoutes(r, inference.NewRouter(translation.NewRegistry()), nil, nil, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/v1/responses")
	ctx.Request.SetBody([]byte(`{"model":"gpt-4"}`))
	r.Handler(&ctx)

	if ctx.Response.StatusCode() == fasthttp.StatusNotFound {
		t.Fatalf("/v1/responses returned 404 — route not registered")
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
	RegisterOpenAIRoutes(r, router, nil, nil, &fakeComboDispatcherForRoutes{})

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
	RegisterOpenAIRoutes(r2, inference.NewRouter(translation.NewRegistry()), nil, nil, nil)

	var ctx2 fasthttp.RequestCtx
	ctx2.Request.Header.SetMethod("POST")
	ctx2.Request.SetRequestURI("/v1/chat/completions")
	ctx2.Request.SetBody([]byte(`{"model":"combomodel","messages":[{"role":"user","content":"hi"}]}`))
	r2.Handler(&ctx2)

	if ctx2.Response.StatusCode() == fasthttp.StatusOK {
		t.Fatalf("nil-dispatcher control status = 200, want error (model unknown)")
	}
}
