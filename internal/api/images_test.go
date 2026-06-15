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

// fakeImagesResolver resolves any model to the embedded fake provider.
type fakeImagesResolver struct {
	prov schemas.Provider
}

func (r *fakeImagesResolver) Resolve(model string) (schemas.Provider, schemas.Key, error) {
	return r.prov, schemas.Key{Provider: "openai"}, nil
}

// fakeImagesProvider records image calls. It embeds fakeMessagesProvider to
// satisfy the full schemas.Provider interface.
type fakeImagesProvider struct {
	fakeMessagesProvider
	genCalled       bool
	editCalled      bool
	variationCalled bool
	genStreamCalled bool
	capturedKey     schemas.Key
	capturedImage   []byte
	capturedMask    []byte
	capturedPrompt  string
	resp            *schemas.ImageGenerationResponse
	perr            *schemas.ProviderError
	streamCh        chan *schemas.StreamChunk
}

func (p *fakeImagesProvider) ImageGeneration(_ *schemas.GatewayContext, key schemas.Key, _ *schemas.ImageGenerationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	p.genCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.resp, nil
}

func (p *fakeImagesProvider) ImageGenerationStream(_ *schemas.GatewayContext, _ schemas.PostHookRunner, key schemas.Key, _ *schemas.ImageGenerationRequest) (chan *schemas.StreamChunk, *schemas.ProviderError) {
	p.genStreamCalled = true
	p.capturedKey = key
	if p.perr != nil {
		return nil, p.perr
	}
	return p.streamCh, nil
}

func (p *fakeImagesProvider) ImageEdit(_ *schemas.GatewayContext, key schemas.Key, req *schemas.ImageEditRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	p.editCalled = true
	p.capturedKey = key
	p.capturedImage = req.Image
	p.capturedMask = req.Mask
	p.capturedPrompt = req.Prompt
	if p.perr != nil {
		return nil, p.perr
	}
	return p.resp, nil
}

func (p *fakeImagesProvider) ImageVariation(_ *schemas.GatewayContext, key schemas.Key, req *schemas.ImageVariationRequest) (*schemas.ImageGenerationResponse, *schemas.ProviderError) {
	p.variationCalled = true
	p.capturedKey = key
	p.capturedImage = req.Image
	if p.perr != nil {
		return nil, p.perr
	}
	return p.resp, nil
}

// TestImagesGenerationsSuccess verifies /v1/images/generations returns the bare
// ImageGenerationResponse with no admin envelope.
func TestImagesGenerationsSuccess(t *testing.T) {
	prov := &fakeImagesProvider{resp: &schemas.ImageGenerationResponse{Created: 1, Data: []schemas.ImageData{{URL: "https://img/1.png"}}}}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.SetBody([]byte(`{"prompt":"a cat","model":"dall-e-3"}`))
	h.Generations(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.genCalled {
		t.Fatal("provider ImageGeneration not called")
	}
	assertBareImageResponse(t, ctx.Response.Body())
	var resp schemas.ImageGenerationResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://img/1.png" {
		t.Errorf("data = %+v", resp.Data)
	}
}

// TestImagesGenerationsInvalidJSON verifies a malformed body returns 400.
func TestImagesGenerationsInvalidJSON(t *testing.T) {
	prov := &fakeImagesProvider{}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.SetBody([]byte(`{not json`))
	h.Generations(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.genCalled {
		t.Fatal("provider should not be called on invalid JSON")
	}
}

// TestImagesGenerationsProviderError verifies a provider 501 passthrough.
func TestImagesGenerationsProviderError(t *testing.T) {
	prov := &fakeImagesProvider{perr: &schemas.ProviderError{StatusCode: 501, Type: "not_implemented", Message: "x"}}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.SetBody([]byte(`{"prompt":"x"}`))
	h.Generations(&ctx)

	if ctx.Response.StatusCode() != 501 {
		t.Fatalf("status = %d, want 501", ctx.Response.StatusCode())
	}
}

// TestImagesGenerationsStream verifies stream:true frames SSE.
func TestImagesGenerationsStream(t *testing.T) {
	ch := make(chan *schemas.StreamChunk, 1)
	ch <- &schemas.StreamChunk{ID: "i1"}
	close(ch)
	prov := &fakeImagesProvider{streamCh: ch}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.SetBody([]byte(`{"prompt":"x","stream":true}`))
	h.Generations(&ctx)

	if !prov.genStreamCalled {
		t.Fatal("provider ImageGenerationStream not called")
	}
	if ct := string(ctx.Response.Header.ContentType()); ct != "text/event-stream" {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}
	if !contains(string(ctx.Response.Body()), "[DONE]") {
		t.Errorf("stream body missing [DONE]: %q", ctx.Response.Body())
	}
}

// TestImagesGenerationsMarshalFailure verifies marshal failure falls back to 500.
func TestImagesGenerationsMarshalFailure(t *testing.T) {
	prev := jsonMarshal
	t.Cleanup(func() { jsonMarshal = prev })
	jsonMarshal = func(v any) ([]byte, error) { return nil, errors.New("boom") }

	router := inference.NewRouter(translation.NewRegistry())
	h := NewImagesHandler(router)
	prov := &fakeImagesProvider{resp: &schemas.ImageGenerationResponse{Created: 1}}
	h.router = &fakeImagesResolver{prov: prov}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.SetBody([]byte(`{"prompt":"x"}`))
	h.Generations(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusInternalServerError {
		t.Errorf("status = %d, want 500", got)
	}
	if got := string(ctx.Response.Body()); got != "internal error" {
		t.Errorf("body = %q, want 'internal error'", got)
	}
}

// TestImagesEditsMultipartSuccess verifies the multipart edit upload reaches the
// provider and returns the bare ImageGenerationResponse.
func TestImagesEditsMultipartSuccess(t *testing.T) {
	imgBytes := []byte("\x89PNG src")
	maskBytes := []byte("\x89PNG mask")
	prov := &fakeImagesProvider{resp: &schemas.ImageGenerationResponse{Created: 1, Data: []schemas.ImageData{{B64JSON: "abc"}}}}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	body, ct := buildMultipart(t,
		map[string][]byte{"image": imgBytes, "mask": maskBytes},
		map[string]string{"prompt": "make it blue", "model": "dall-e-2"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/edits")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Edits(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.editCalled {
		t.Fatal("provider ImageEdit not called")
	}
	if string(prov.capturedImage) != string(imgBytes) {
		t.Errorf("image = %q, want round-trip", prov.capturedImage)
	}
	if string(prov.capturedMask) != string(maskBytes) {
		t.Errorf("mask = %q, want round-trip", prov.capturedMask)
	}
	if prov.capturedPrompt != "make it blue" {
		t.Errorf("prompt = %q", prov.capturedPrompt)
	}
	assertBareImageResponse(t, ctx.Response.Body())
}

// assertBareImageResponse verifies the body is the bare OpenAI
// ImageGenerationResponse — a top-level {created, data:[...]} object — and NOT
// the {data:<obj>,error} admin envelope. The OpenAI shape legitimately has a
// top-level "data" array, so the proof is: it decodes to the bare object, "data"
// is a JSON array, and there is no top-level "error" key.
func assertBareImageResponse(t *testing.T, body []byte) {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		t.Fatalf("unmarshal top-level: %v", err)
	}
	if _, ok := top["error"]; ok {
		t.Error("response has top-level 'error' key on success")
	}
	if _, ok := top["created"]; !ok {
		t.Error("response missing top-level 'created' (not the bare OpenAI image shape)")
	}
	if dataRaw, ok := top["data"]; ok {
		if len(dataRaw) == 0 || dataRaw[0] != '[' {
			t.Errorf("'data' is %s, want a JSON array (admin envelope wraps an object here)", dataRaw)
		}
	}
}

// TestImagesEditsNonMultipart verifies a non-multipart edit returns 400.
func TestImagesEditsNonMultipart(t *testing.T) {
	prov := &fakeImagesProvider{}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/edits")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"prompt":"x"}`))
	h.Edits(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.editCalled {
		t.Fatal("provider should not be called for non-multipart")
	}
}

// TestImagesEditsMissingImage verifies a multipart edit without the image part returns 400.
func TestImagesEditsMissingImage(t *testing.T) {
	prov := &fakeImagesProvider{}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	body, ct := buildMultipart(t, nil, map[string]string{"prompt": "x"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/edits")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Edits(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.editCalled {
		t.Fatal("provider should not be called when image missing")
	}
}

// TestImagesVariationsMultipartSuccess verifies the variation upload reaches the provider.
func TestImagesVariationsMultipartSuccess(t *testing.T) {
	imgBytes := []byte("\x89PNG variation")
	prov := &fakeImagesProvider{resp: &schemas.ImageGenerationResponse{Created: 2, Data: []schemas.ImageData{{URL: "https://v.png"}}}}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	body, ct := buildMultipart(t, map[string][]byte{"image": imgBytes}, map[string]string{"model": "dall-e-2"})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/variations")
	ctx.Request.Header.SetContentType(ct)
	ctx.Request.SetBody(body)
	h.Variations(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if !prov.variationCalled {
		t.Fatal("provider ImageVariation not called")
	}
	if string(prov.capturedImage) != string(imgBytes) {
		t.Errorf("image = %q, want round-trip", prov.capturedImage)
	}
	assertBareImageResponse(t, ctx.Response.Body())
}

// TestImagesVariationsNonMultipart verifies a non-multipart variation returns 400.
func TestImagesVariationsNonMultipart(t *testing.T) {
	prov := &fakeImagesProvider{}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/variations")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{}`))
	h.Variations(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
	if prov.variationCalled {
		t.Fatal("provider should not be called for non-multipart")
	}
}

// TestImagesGenerationsVKDenied verifies the x-g0-vk gate denies before dispatch.
func TestImagesGenerationsVKDenied(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-denied", &VKInfo{
		Key:      "vk-denied",
		Configs:  []VKProviderConfig{{Provider: "openai", AllowedModels: []string{"dall-e-3"}}},
		IsActive: true,
	})
	quota := newFakeVKQuotaChecker(struct {
		ok     bool
		status int
		reason string
	}{ok: false, status: 429, reason: "budget exhausted"})

	prov := &fakeImagesProvider{}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, quota))

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.Header.Set("x-g0-vk", "vk-denied")
	ctx.Request.SetBody([]byte(`{"prompt":"x","model":"dall-e-3"}`))
	h.Generations(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", ctx.Response.StatusCode())
	}
	if prov.genCalled {
		t.Fatal("provider ImageGeneration should not be called")
	}
}

// TestImagesGenerationsVKPinned verifies pinned-key override reaches the provider.
func TestImagesGenerationsVKPinned(t *testing.T) {
	resolver := newFakeVKResolver()
	resolver.set("vk-pinned", &VKInfo{
		Key:      "vk-pinned",
		Configs:  []VKProviderConfig{{Provider: "openai", AllowedModels: []string{"dall-e-3"}, KeyIDs: []string{"conn-2"}}},
		IsActive: true,
	})

	prov := &fakeImagesProvider{resp: &schemas.ImageGenerationResponse{Created: 1}}
	h := &ImagesHandler{router: &fakeImagesResolver{prov: prov}}
	h.SetVKGate(NewVKGate(resolver, newFakeVKQuotaChecker()))
	h.SetVKPinnedResolver(&fakePinnedKeyResolver{connID: "conn-2", credential: "cred-2", ok: true})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/v1/images/generations")
	ctx.Request.Header.Set("x-g0-vk", "vk-pinned")
	ctx.Request.SetBody([]byte(`{"prompt":"x","model":"dall-e-3"}`))
	h.Generations(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if prov.capturedKey.ID != "conn-2" || prov.capturedKey.Value != "cred-2" {
		t.Errorf("key = %+v, want conn-2/cred-2", prov.capturedKey)
	}
}
