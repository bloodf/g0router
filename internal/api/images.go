package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// ImagesHandler handles POST /v1/images/{generations,edits,variations}. These
// are OpenAI-compatible routes: success returns the bare ImageGenerationResponse
// JSON, never the {data,error} admin envelope.
type ImagesHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewImagesHandler creates an images handler.
func NewImagesHandler(router *inference.Router) *ImagesHandler {
	return &ImagesHandler{router: router}
}

// SetUsageRecorder wires a usage recorder.
func (h *ImagesHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a pending tracker.
func (h *ImagesHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a detail capture.
func (h *ImagesHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate.
func (h *ImagesHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the virtual-key pinned-key resolver.
func (h *ImagesHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

func (h *ImagesHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}

// gate resolves the model and applies the x-g0-vk gate + pinning. It mirrors
// AudioHandler.resolveAndGate. Returns a non-nil error sentinel (after writing
// the response) when the request must stop.
func (h *ImagesHandler) gate(ctx *fasthttp.RequestCtx, model string, raw []byte, headers map[string]string, endpoint string, g *recordGlue) (schemas.Provider, schemas.Key, *schemas.ProviderError) {
	provider, key, err := h.router.Resolve(model)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return nil, schemas.Key{}, &schemas.ProviderError{StatusCode: 400}
	}

	// x-g0-vk gate: unconditional so AllowVK("") reaches the mandatory branch
	// (bf-gov-4 Option-A). When mandatory OFF and vkHeader=="": returns
	// (true,0,"",nil) — all blocks below are no-ops, byte-identical to before.
	vkHeader := string(ctx.Request.Header.Peek("x-g0-vk"))
	ok, status, reason, keyIDs := h.vkGate.AllowVK(vkHeader, model, key.Provider)
	if !ok {
		errType := "invalid_request_error"
		if status == 429 {
			errType = "rate_limit_exceeded"
		}
		g.recordError(endpoint, model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: status, Message: reason, Type: errType})
		writeError(ctx, status, errType, reason, nil)
		return nil, schemas.Key{}, &schemas.ProviderError{StatusCode: status}
	}
	if len(keyIDs) > 0 && h.pinnedResolver != nil {
		if connID, credential, ok := h.pinnedResolver.ResolvePinned(key.Provider, model, keyIDs); ok {
			key.ID = connID
			key.Value = credential
		}
	}
	g.apiKey = vkHeader
	return provider, key, nil
}

// writeImageResponse writes a bare ImageGenerationResponse or, on marshal
// failure, a plain-text 500. It records usage on success.
func (h *ImagesHandler) writeImageResponse(ctx *fasthttp.RequestCtx, g *recordGlue, endpoint, model string, key schemas.Key, raw []byte, headers map[string]string, resp *schemas.ImageGenerationResponse) {
	b, mErr := jsonMarshal(resp)
	if mErr != nil {
		g.recordError(endpoint, model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)
	g.recordNonStream(endpoint, model, key.Provider, key.ID, raw, headers, 0, 0, resp)
}

// Generations handles POST /v1/images/generations (JSON in, bare JSON out, or SSE).
func (h *ImagesHandler) Generations(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/images/generations"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var req schemas.ImageGenerationRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}
	stream := requestWantsStream(raw)

	provider, key, perr := h.gate(ctx, req.Model, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}

	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if stream {
		// Open the provider stream BEFORE setting SSE headers so a stream-open
		// *ProviderError returns an application/json error, not a
		// text/event-stream framing mismatch (PAR-BF-OAI-201).
		ch, sperr := provider.ImageGenerationStream(gatewayCtx, nil, key, &req)
		if sperr != nil {
			g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
			writeProviderError(ctx, sperr)
			return
		}

		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		if sErr := writeSSEStream(streamCtx, ctx, ch); sErr != nil {
			log.Printf("images generations stream error: %v", sErr)
		}
		return
	}

	resp, sperr := provider.ImageGeneration(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeImageResponse(ctx, &g, endpoint, req.Model, key, raw, headers, resp)
}

// Edits handles POST /v1/images/edits (multipart in, bare JSON out).
func (h *ImagesHandler) Edits(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/images/edits"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	if !isMultipart(ctx) {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "expected multipart/form-data", nil)
		return
	}
	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid multipart body", nil)
		return
	}

	image, ok, err := readMultipartFile(form, "image")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "read image part: "+err.Error(), nil)
		return
	}
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing required image part", nil)
		return
	}
	mask, _, err := readMultipartFile(form, "mask")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "read mask part: "+err.Error(), nil)
		return
	}

	req := schemas.ImageEditRequest{
		Image:  image,
		Mask:   mask,
		Prompt: formValue(form, "prompt"),
		Model:  formValue(form, "model"),
		User:   formValue(form, "user"),
	}
	if v := formValue(form, "n"); v != "" {
		if n, perr := strconv.Atoi(v); perr == nil {
			req.N = &n
		}
	}
	if v := formValue(form, "size"); v != "" {
		req.Size = &v
	}
	if v := formValue(form, "response_format"); v != "" {
		req.ResponseFormat = &v
	}

	provider, key, perr := h.gate(ctx, req.Model, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}

	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.ImageEdit(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeImageResponse(ctx, &g, endpoint, req.Model, key, raw, headers, resp)
}

// Variations handles POST /v1/images/variations (multipart in, bare JSON out).
func (h *ImagesHandler) Variations(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/images/variations"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	if !isMultipart(ctx) {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "expected multipart/form-data", nil)
		return
	}
	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid multipart body", nil)
		return
	}

	image, ok, err := readMultipartFile(form, "image")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "read image part: "+err.Error(), nil)
		return
	}
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing required image part", nil)
		return
	}

	req := schemas.ImageVariationRequest{
		Image: image,
		Model: formValue(form, "model"),
		User:  formValue(form, "user"),
	}
	if v := formValue(form, "n"); v != "" {
		if n, perr := strconv.Atoi(v); perr == nil {
			req.N = &n
		}
	}
	if v := formValue(form, "size"); v != "" {
		req.Size = &v
	}
	if v := formValue(form, "response_format"); v != "" {
		req.ResponseFormat = &v
	}

	provider, key, perr := h.gate(ctx, req.Model, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}

	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.ImageVariation(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeImageResponse(ctx, &g, endpoint, req.Model, key, raw, headers, resp)
}
