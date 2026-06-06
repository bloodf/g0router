package api

import (
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func TestRtkBytesSavedDisabledOrNil(t *testing.T) {
	if got := rtkBytesSaved(store.Settings{RTKEnabled: false}, &providers.ChatRequest{Model: "m"}); got != nil {
		t.Fatalf("RTK disabled: got %v, want nil", got)
	}
	if got := rtkBytesSaved(store.Settings{RTKEnabled: true}, nil); got != nil {
		t.Fatalf("nil request: got %v, want nil", got)
	}
}

func TestRtkBytesSavedReturnsPositiveGain(t *testing.T) {
	req := &providers.ChatRequest{Model: "m"}
	got := rtkBytesSaved(store.Settings{RTKEnabled: true}, req)
	if got == nil || *got <= 0 {
		t.Fatalf("expected positive savings, got %v", got)
	}
}

func TestRtkBytesSavedMarshalError(t *testing.T) {
	req := &providers.ChatRequest{
		Model: "m",
		Stop:  make(chan int), // channels cannot be JSON-marshaled
	}
	if got := rtkBytesSaved(store.Settings{RTKEnabled: true}, req); got != nil {
		t.Fatalf("expected nil on marshal error, got %v", got)
	}
}

func TestStatusClassForAllClasses(t *testing.T) {
	cases := []struct {
		code int
		want string
	}{
		{500, "5xx"},
		{503, "5xx"},
		{404, "4xx"},
		{400, "4xx"},
		{301, "3xx"},
		{200, "2xx"},
		{204, "2xx"},
		{100, "other"},
		{0, "other"},
	}
	for _, c := range cases {
		if got := statusClassFor(c.code); got != c.want {
			t.Errorf("statusClassFor(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

func TestParseCacheableRequest(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantModel  string
		wantStream bool
		wantOK     bool
	}{
		{"streaming", `{"model":"gpt-4o","stream":true}`, "gpt-4o", true, true},
		{"non-streaming explicit", `{"model":"gpt-4o","stream":false}`, "gpt-4o", false, true},
		{"stream omitted", `{"model":"gpt-4o"}`, "gpt-4o", false, true},
		{"invalid json", `{bad`, "", false, false},
		{"empty model", `{}`, "", false, true},
	}
	for _, c := range cases {
		model, stream, ok := parseCacheableRequest([]byte(c.body))
		if model != c.wantModel || stream != c.wantStream || ok != c.wantOK {
			t.Errorf("%s: parseCacheableRequest = (%q,%v,%v), want (%q,%v,%v)",
				c.name, model, stream, ok, c.wantModel, c.wantStream, c.wantOK)
		}
	}
}

func TestHandleMetricsMethodNotAllowed(t *testing.T) {
	s := &Server{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	s.handleMetrics(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func TestHandleMetricsNilMetricsReturnsEmpty(t *testing.T) {
	s := &Server{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	s.handleMetrics(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if len(ctx.Response.Body()) != 0 {
		t.Fatalf("body = %q, want empty", ctx.Response.Body())
	}
}
