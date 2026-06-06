package handlers

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestSkillsReturnsCatalog(t *testing.T) {
	ctx, body := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		Skills(ctx)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), body)
	}

	var decoded struct {
		Data []skillItem `json:"data"`
	}
	decodeJSON(t, body, &decoded)
	if len(decoded.Data) == 0 {
		t.Fatal("expected non-empty skills catalog")
	}
	for _, item := range decoded.Data {
		if item.Name == "" {
			t.Fatalf("skill name empty: %+v", item)
		}
		if item.Category == "" {
			t.Fatalf("skill category empty: %+v", item)
		}
	}
}
