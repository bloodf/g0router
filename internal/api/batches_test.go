package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeBatchesResolver resolves any model to the embedded fake provider.
type fakeBatchesResolver struct {
	prov schemas.Provider
}

func (r *fakeBatchesResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{Provider: "openai"}, nil
}

// fakeBatchesProvider records Batch* calls. It embeds fakeMessagesProvider to
// satisfy the full schemas.Provider interface.
type fakeBatchesProvider struct {
	fakeMessagesProvider
	createCalled   bool
	listCalled     bool
	retrieveCalled bool
	cancelCalled   bool
	capturedKey    schemas.Key
	capturedReq    *schemas.BatchCreateRequest
	capturedID     string
	createResp     *schemas.Batch
	listResp       *schemas.BatchListResponse
	retrieveResp   *schemas.Batch
	cancelResp     *schemas.Batch
	perr           *schemas.ProviderError
}

func (p *fakeBatchesProvider) BatchCreate(_ *schemas.GatewayContext, key schemas.Key, req *schemas.BatchCreateRequest) (*schemas.Batch, *schemas.ProviderError) {
	p.createCalled = true
	p.capturedKey = key
	p.capturedReq = req
	if p.perr != nil {
		return nil, p.perr
	}
	return p.createResp, nil
}

func (p *fakeBatchesProvider) BatchList(_ *schemas.GatewayContext, key schemas.Key) (*schemas.BatchListResponse, *schemas.ProviderError) {
	p.listCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.listResp, nil
}

func (p *fakeBatchesProvider) BatchRetrieve(_ *schemas.GatewayContext, key schemas.Key, batchID string) (*schemas.Batch, *schemas.ProviderError) {
	p.retrieveCalled = true
	p.capturedKey = key
	p.capturedID = batchID
	if p.perr != nil {
		return nil, p.perr
	}
	return p.retrieveResp, nil
}

func (p *fakeBatchesProvider) BatchCancel(_ *schemas.GatewayContext, key schemas.Key, batchID string) (*schemas.Batch, *schemas.ProviderError) {
	p.cancelCalled = true
	p.capturedKey = key
	p.capturedID = batchID
	if p.perr != nil {
		return nil, p.perr
	}
	return p.cancelResp, nil
}

// TestBatchesCreateSuccess verifies a JSON body reaches the provider and returns
// the bare Batch (no envelope).
func TestBatchesCreateSuccess(t *testing.T) {
	prov := &fakeBatchesProvider{createResp: &schemas.Batch{ID: "batch_1", Object: "batch", Status: "validating"}}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/batches")
	ctx.Request.SetBody([]byte(`{"input_file_id":"file-in","endpoint":"/v1/chat/completions","completion_window":"24h"}`))
	h.Create(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.createCalled {
		t.Fatal("provider BatchCreate not called")
	}
	if prov.capturedReq == nil || prov.capturedReq.InputFileID != "file-in" {
		t.Errorf("req = %+v, want InputFileID file-in", prov.capturedReq)
	}
	assertNoEnvelope(t, ctx.Response.Body())
	var resp schemas.Batch
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != "batch_1" {
		t.Errorf("ID = %q, want batch_1", resp.ID)
	}
}

// TestBatchesCreateInvalidJSON verifies a malformed body returns 400.
func TestBatchesCreateInvalidJSON(t *testing.T) {
	prov := &fakeBatchesProvider{}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/batches")
	ctx.Request.SetBody([]byte(`{not json`))
	h.Create(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.createCalled {
		t.Fatal("provider should not be called on invalid JSON")
	}
}

// TestBatchesListSuccess verifies List returns the bare BatchListResponse.
func TestBatchesListSuccess(t *testing.T) {
	prov := &fakeBatchesProvider{listResp: &schemas.BatchListResponse{Object: "list", Data: []schemas.Batch{{ID: "batch_1"}}}}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/batches")
	h.List(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.listCalled {
		t.Fatal("provider BatchList not called")
	}
	// BatchListResponse is the bare OpenAI list shape; "data" is the real payload.
	assertNoErrorKey(t, ctx.Response.Body())
	var resp schemas.BatchListResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Object != "list" || len(resp.Data) != 1 {
		t.Errorf("resp = %+v, want bare BatchListResponse with 1 item", resp)
	}
}

// TestBatchesRetrieveSuccess verifies the {batch_id} param reaches the provider.
func TestBatchesRetrieveSuccess(t *testing.T) {
	prov := &fakeBatchesProvider{retrieveResp: &schemas.Batch{ID: "batch_7", Object: "batch", Status: "completed"}}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/batches/batch_7")
	ctx.SetUserValue("batch_id", "batch_7")
	h.Retrieve(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedID != "batch_7" {
		t.Errorf("id = %q, want batch_7", prov.capturedID)
	}
	assertNoEnvelope(t, ctx.Response.Body())
}

// TestBatchesRetrieveEmptyID verifies an empty batch_id returns 400.
func TestBatchesRetrieveEmptyID(t *testing.T) {
	prov := &fakeBatchesProvider{}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/batches/")
	ctx.SetUserValue("batch_id", "")
	h.Retrieve(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.retrieveCalled {
		t.Fatal("provider should not be called for empty id")
	}
}

// TestBatchesCancelSuccess verifies the {batch_id} param reaches Cancel (POST).
func TestBatchesCancelSuccess(t *testing.T) {
	prov := &fakeBatchesProvider{cancelResp: &schemas.Batch{ID: "batch_c", Object: "batch", Status: "cancelling"}}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/batches/batch_c/cancel")
	ctx.SetUserValue("batch_id", "batch_c")
	h.Cancel(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.cancelCalled {
		t.Fatal("provider BatchCancel not called")
	}
	if prov.capturedID != "batch_c" {
		t.Errorf("id = %q, want batch_c", prov.capturedID)
	}
	assertNoEnvelope(t, ctx.Response.Body())
}

// TestBatchesCancelEmptyID verifies an empty batch_id returns 400.
func TestBatchesCancelEmptyID(t *testing.T) {
	prov := &fakeBatchesProvider{}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/batches//cancel")
	ctx.SetUserValue("batch_id", "")
	h.Cancel(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.cancelCalled {
		t.Fatal("provider should not be called for empty id")
	}
}

// TestBatchesProviderError verifies a provider 501 is passed through.
func TestBatchesProviderError(t *testing.T) {
	prov := &fakeBatchesProvider{perr: &schemas.ProviderError{StatusCode: 501, Type: "not_implemented", Message: "batch_list not implemented"}}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/batches")
	h.List(&ctx)

	if ctx.Response.StatusCode() != 501 {
		t.Fatalf("status = %d, want 501", ctx.Response.StatusCode())
	}
}

// TestBatchesMarshalFailure verifies a marshal failure falls back to plain 500.
func TestBatchesMarshalFailure(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("boom") }

	router := inference.NewRouter(translation.NewRegistry())
	h := NewBatchesHandler(router)
	prov := &fakeBatchesProvider{listResp: &schemas.BatchListResponse{Object: "list"}}
	h.router = &fakeBatchesResolver{prov: prov}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/batches")
	h.List(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", got)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want 'internal error'", got)
	}
}

// TestBatchesCreateVKDenied verifies the x-g0-vk gate denies before dispatch.
func TestBatchesCreateVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key:      "vk-denied",
		Configs:  []VKProviderConfig{{Provider: "openai"}},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 429, reason: "budget exhausted"})

	prov := &fakeBatchesProvider{}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, quota))

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/batches")
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody([]byte(`{"input_file_id":"f","endpoint":"/v1/chat/completions","completion_window":"24h"}`))
	h.Create(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
	if prov.createCalled {
		t.Fatal("provider BatchCreate should not be called")
	}
}

// TestBatchesCreateVKPinned verifies pinned-key override reaches the provider.
func TestBatchesCreateVKPinned(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-pinned", &VKInfo{
		Key:      "vk-pinned",
		Configs:  []VKProviderConfig{{Provider: "openai", KeyIDs: []string{"conn-2"}}},
		IsActive: true,
	})

	prov := &fakeBatchesProvider{createResp: &schemas.Batch{ID: "batch_1"}}
	h := &BatchesHandler{router: &fakeBatchesResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, newFakeVKQuotaChecker()))
	h.SetVKPinnedResolver(&fakePinnedKeyResolver{connID: "conn-2", credential: "cred-2", ok: true})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/batches")
	ctx.Request.Header.Set("x-g0-vk", "vk-pinned")
	ctx.Request.SetBody([]byte(`{"input_file_id":"f","endpoint":"/v1/chat/completions","completion_window":"24h"}`))
	h.Create(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedKey.ID != "conn-2" || prov.capturedKey.Value != "cred-2" {
		t.Errorf("key = %+v, want conn-2/cred-2", prov.capturedKey)
	}
}
