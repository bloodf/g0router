package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/valyala/fasthttp"
)

// TestModelsHandlerMarshalFailureFallsBackTo500 verifies that when the
// response marshal seam fails, the models handler eventually writes a
// 500 status (AUD-011/012). The provider will fail with a network error
// in this test environment, which routes through writeError — writeError
// then exercises the same failing jsonMarshal seam and falls back to a
// plain-text 500 per the AUD-009–012 acceptance contract.
func TestModelsHandlerMarshalFailureFallsBackTo500(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	router := inference.NewRouter()
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
