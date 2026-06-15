package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
)

// fakeFilesResolver resolves any model to the embedded fake provider.
type fakeFilesResolver struct {
	prov schemas.Provider
}

func (r *fakeFilesResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{Provider: "openai"}, nil
}

// fakeFilesProvider records File* calls. It embeds fakeMessagesProvider to
// satisfy the full schemas.Provider interface.
type fakeFilesProvider struct {
	fakeMessagesProvider
	uploadCalled   bool
	listCalled     bool
	retrieveCalled bool
	deleteCalled   bool
	contentCalled  bool
	capturedKey    schemas.Key
	capturedFile   []byte
	capturedPurp   string
	capturedID     string
	uploadResp     *schemas.FileObject
	listResp       *schemas.FileListResponse
	retrieveResp   *schemas.FileObject
	deleteResp     *schemas.FileDeleteResponse
	contentResp    []byte
	perr           *schemas.ProviderError
}

func (p *fakeFilesProvider) FileUpload(_ *schemas.GatewayContext, key schemas.Key, req *schemas.FileUploadRequest) (*schemas.FileObject, *schemas.ProviderError) {
	p.uploadCalled = true
	p.capturedKey = key
	p.capturedFile = req.File
	p.capturedPurp = req.Purpose
	if p.perr != nil {
		return nil, p.perr
	}
	return p.uploadResp, nil
}

func (p *fakeFilesProvider) FileList(_ *schemas.GatewayContext, key schemas.Key) (*schemas.FileListResponse, *schemas.ProviderError) {
	p.listCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.listResp, nil
}

func (p *fakeFilesProvider) FileRetrieve(_ *schemas.GatewayContext, key schemas.Key, fileID string) (*schemas.FileObject, *schemas.ProviderError) {
	p.retrieveCalled = true
	p.capturedKey = key
	p.capturedID = fileID
	if p.perr != nil {
		return nil, p.perr
	}
	return p.retrieveResp, nil
}

func (p *fakeFilesProvider) FileDelete(_ *schemas.GatewayContext, key schemas.Key, fileID string) (*schemas.FileDeleteResponse, *schemas.ProviderError) {
	p.deleteCalled = true
	p.capturedKey = key
	p.capturedID = fileID
	if p.perr != nil {
		return nil, p.perr
	}
	return p.deleteResp, nil
}

func (p *fakeFilesProvider) FileContent(_ *schemas.GatewayContext, key schemas.Key, fileID string) ([]byte, *schemas.ProviderError) {
	p.contentCalled = true
	p.capturedKey = key
	p.capturedID = fileID
	if p.perr != nil {
		return nil, p.perr
	}
	return p.contentResp, nil
}

// TestFilesUploadMultipartSuccess verifies a multipart upload reaches the
// provider and returns the bare FileObject (no envelope).
func TestFilesUploadMultipartSuccess(t *testing.T) {
	fileBytes := []byte("a,b,c\n1,2,3\n")
	prov := &fakeFilesProvider{uploadResp: &schemas.FileObject{ID: "file-1", Object: "file"}}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	body, ct := buildMultipart(t, map[string][]byte{"file": fileBytes}, map[string]string{"purpose": "batch"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/files")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Upload(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.uploadCalled {
		t.Fatal("provider FileUpload not called")
	}
	if !bytes.Equal(prov.capturedFile, fileBytes) {
		t.Errorf("file = %q, want round-trip", prov.capturedFile)
	}
	if prov.capturedPurp != "batch" {
		t.Errorf("purpose = %q, want batch", prov.capturedPurp)
	}
	assertNoEnvelope(t, ctx.Response.Body())
	var resp schemas.FileObject
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != "file-1" {
		t.Errorf("ID = %q, want file-1", resp.ID)
	}
}

// TestFilesUploadNonMultipart verifies a non-multipart request returns 400.
func TestFilesUploadNonMultipart(t *testing.T) {
	prov := &fakeFilesProvider{}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/files")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"purpose":"batch"}`))
	h.Upload(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.uploadCalled {
		t.Fatal("provider should not be called for non-multipart")
	}
}

// TestFilesUploadMissingFile verifies a multipart body without the file part 400s.
func TestFilesUploadMissingFile(t *testing.T) {
	prov := &fakeFilesProvider{}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	body, ct := buildMultipart(t, nil, map[string]string{"purpose": "batch"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/files")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Upload(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.uploadCalled {
		t.Fatal("provider should not be called when file missing")
	}
}

// TestFilesUploadMissingPurpose verifies a multipart body without purpose 400s.
func TestFilesUploadMissingPurpose(t *testing.T) {
	prov := &fakeFilesProvider{}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	body, ct := buildMultipart(t, map[string][]byte{"file": []byte("x")}, nil)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/files")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Upload(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.uploadCalled {
		t.Fatal("provider should not be called when purpose missing")
	}
}

// TestFilesListSuccess verifies List returns the bare FileListResponse.
func TestFilesListSuccess(t *testing.T) {
	prov := &fakeFilesProvider{listResp: &schemas.FileListResponse{Object: "list", Data: []schemas.FileObject{{ID: "file-1"}}}}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files")
	h.List(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.listCalled {
		t.Fatal("provider FileList not called")
	}
	assertNoEnvelope(t, ctx.Response.Body())
}

// TestFilesRetrieveSuccess verifies the {file_id} param reaches the provider.
func TestFilesRetrieveSuccess(t *testing.T) {
	prov := &fakeFilesProvider{retrieveResp: &schemas.FileObject{ID: "file-7", Object: "file"}}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files/file-7")
	ctx.SetUserValue("file_id", "file-7")
	h.Retrieve(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedID != "file-7" {
		t.Errorf("id = %q, want file-7", prov.capturedID)
	}
	assertNoEnvelope(t, ctx.Response.Body())
}

// TestFilesRetrieveEmptyID verifies an empty file_id returns 400 before dispatch.
func TestFilesRetrieveEmptyID(t *testing.T) {
	prov := &fakeFilesProvider{}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files/")
	ctx.SetUserValue("file_id", "")
	h.Retrieve(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.retrieveCalled {
		t.Fatal("provider should not be called for empty id")
	}
}

// TestFilesDeleteSuccess verifies Delete returns the bare FileDeleteResponse.
func TestFilesDeleteSuccess(t *testing.T) {
	prov := &fakeFilesProvider{deleteResp: &schemas.FileDeleteResponse{ID: "file-9", Object: "file", Deleted: true}}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodDelete)
	ctx.Request.SetRequestURI("/v1/files/file-9")
	ctx.SetUserValue("file_id", "file-9")
	h.Delete(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedID != "file-9" {
		t.Errorf("id = %q, want file-9", prov.capturedID)
	}
	assertNoEnvelope(t, ctx.Response.Body())
}

// TestFilesContentReturnsRawBytes verifies Content writes raw bytes with
// application/octet-stream and NO JSON envelope (ESC-FILE-CONTENT-BYTES).
func TestFilesContentReturnsRawBytes(t *testing.T) {
	content := []byte("{\"id\":\"req-1\"}\n{\"id\":\"req-2\"}\n")
	prov := &fakeFilesProvider{contentResp: content}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files/file-c/content")
	ctx.SetUserValue("file_id", "file-c")
	h.Content(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.contentCalled {
		t.Fatal("provider FileContent not called")
	}
	if got := ctx.Response.Body(); !bytes.Equal(got, content) {
		t.Errorf("body = %q, want raw content", got)
	}
	if ct := string(ctx.Response.Header.ContentType()); ct != "application/octet-stream" {
		t.Errorf("content-type = %q, want application/octet-stream", ct)
	}
	// The body must NOT be a JSON object with data/error keys.
	var top map[string]json.RawMessage
	if json.Unmarshal(ctx.Response.Body(), &top) == nil {
		if _, ok := top["data"]; ok {
			t.Error("content body parsed as JSON with 'data' key — must be raw bytes")
		}
		if _, ok := top["error"]; ok {
			t.Error("content body parsed as JSON with 'error' key — must be raw bytes")
		}
	}
}

// TestFilesContentEmptyID verifies an empty file_id 400s before dispatch.
func TestFilesContentEmptyID(t *testing.T) {
	prov := &fakeFilesProvider{}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files//content")
	ctx.SetUserValue("file_id", "")
	h.Content(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.contentCalled {
		t.Fatal("provider should not be called for empty id")
	}
}

// TestFilesProviderError verifies a provider 501 is passed through.
func TestFilesProviderError(t *testing.T) {
	prov := &fakeFilesProvider{perr: &schemas.ProviderError{StatusCode: 501, Type: "not_implemented", Message: "file_list not implemented"}}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files")
	h.List(&ctx)

	if ctx.Response.StatusCode() != 501 {
		t.Fatalf("status = %d, want 501", ctx.Response.StatusCode())
	}
}

// TestFilesMarshalFailure verifies a marshal failure falls back to plain 500.
func TestFilesMarshalFailure(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("boom") }

	router := inference.NewRouter(translation.NewRegistry())
	h := NewFilesHandler(router)
	prov := &fakeFilesProvider{listResp: &schemas.FileListResponse{Object: "list"}}
	h.router = &fakeFilesResolver{prov: prov}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/v1/files")
	h.List(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", got)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want 'internal error'", got)
	}
}

// TestFilesUploadVKDenied verifies the x-g0-vk gate denies before dispatch.
func TestFilesUploadVKDenied(t *testing.T) {
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

	prov := &fakeFilesProvider{}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, quota))

	body, ct := buildMultipart(t, map[string][]byte{"file": []byte("x")}, map[string]string{"purpose": "batch"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/files")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody(body)
	h.Upload(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
	if prov.uploadCalled {
		t.Fatal("provider FileUpload should not be called")
	}
}

// TestFilesUploadVKPinned verifies pinned-key override reaches the provider.
func TestFilesUploadVKPinned(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-pinned", &VKInfo{
		Key:      "vk-pinned",
		Configs:  []VKProviderConfig{{Provider: "openai", KeyIDs: []string{"conn-2"}}},
		IsActive: true,
	})

	prov := &fakeFilesProvider{uploadResp: &schemas.FileObject{ID: "file-1"}}
	h := &FilesHandler{router: &fakeFilesResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, newFakeVKQuotaChecker()))
	h.SetVKPinnedResolver(&fakePinnedKeyResolver{connID: "conn-2", credential: "cred-2", ok: true})

	body, ct := buildMultipart(t, map[string][]byte{"file": []byte("x")}, map[string]string{"purpose": "batch"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/files")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.Header.Set("x-g0-vk", "vk-pinned")
	ctx.Request.SetBody(body)
	h.Upload(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedKey.ID != "conn-2" || prov.capturedKey.Value != "cred-2" {
		t.Errorf("key = %+v, want conn-2/cred-2", prov.capturedKey)
	}
}
