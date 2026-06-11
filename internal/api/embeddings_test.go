package api

import (
	"errors"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// TestEmbeddingsHandlerMarshalFailureFallsBackTo500 verifies that when
// the response marshal seam fails, the embeddings handler eventually
// writes a 500 status (AUD-010). The provider will fail with a network
// error in this test environment, which routes through writeError —
// writeError then exercises the same failing jsonMarshal seam and falls
// back to a plain-text 500 per the AUD-009–012 acceptance contract.
func TestEmbeddingsHandlerMarshalFailureFallsBackTo500(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })

	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}

	router := inference.NewRouter(translation.NewRegistry())
	h := NewEmbeddingsHandler(router)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/embeddings")
	ctx.Request.SetBody([]byte(`{"model":"text-embedding-3-small","input":"hello"}`))
	h.Handle(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want %d", got, fasthttp.StatusInternalServerError)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want %q", got, "internal error")
	}
}
