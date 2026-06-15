package api

import (
	"fmt"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// FilesHandler handles the /v1/files CRUD routes (upload/list/retrieve/delete/
// content). These are OpenAI-compatible routes: the JSON endpoints return the
// bare OpenAI object and content returns raw file bytes — never the {data,error}
// admin envelope. State lives upstream at OpenAI (Option A, stateless passthrough);
// requests carry no model, so resolution uses the empty-model sentinel which
// lands the openai provider.
type FilesHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewFilesHandler creates a files handler.
func NewFilesHandler(router *inference.Router) *FilesHandler {
	return &FilesHandler{router: router}
}

// SetUsageRecorder wires a usage recorder.
func (h *FilesHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a pending tracker.
func (h *FilesHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a detail capture.
func (h *FilesHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate.
func (h *FilesHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the virtual-key pinned-key resolver.
func (h *FilesHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

func (h *FilesHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}

// resolveAndGate resolves the openai provider (empty-model sentinel) and applies
// the x-g0-vk gate + pinning, mirroring AudioHandler.resolveAndGate.
func (h *FilesHandler) resolveAndGate(ctx *fasthttp.RequestCtx, raw []byte, headers map[string]string, endpoint string, g *recordGlue) (schemas.Provider, schemas.Key, *schemas.ProviderError) {
	const model = ""
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

// writeJSON marshals resp as the bare OpenAI object and writes it 200, falling
// back to a plain-text 500 on marshal failure (mirrors AudioHandler).
func (h *FilesHandler) writeJSON(ctx *fasthttp.RequestCtx, g *recordGlue, endpoint, model string, key schemas.Key, raw []byte, headers map[string]string, resp any) {
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

// Upload handles POST /v1/files. It parses a multipart/form-data upload (file +
// purpose) (ESC-MULTIPART-UPLOAD) and returns the bare FileObject JSON.
func (h *FilesHandler) Upload(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/files"
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

	file, ok, err := readMultipartFile(form, "file")
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "read file part: "+err.Error(), nil)
		return
	}
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing required file part", nil)
		return
	}
	purpose := formValue(form, "purpose")
	if purpose == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing required field: purpose", nil)
		return
	}

	req := schemas.FileUploadRequest{File: file, Purpose: purpose}
	if v := formValue(form, "filename"); v != "" {
		req.Filename = v
	} else if fhs := form.File["file"]; len(fhs) > 0 && fhs[0].Filename != "" {
		req.Filename = fhs[0].Filename
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.FileUpload(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// List handles GET /v1/files and returns the bare FileListResponse JSON.
func (h *FilesHandler) List(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/files"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.FileList(gatewayCtx, key)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// Retrieve handles GET /v1/files/{file_id} and returns the bare FileObject JSON.
func (h *FilesHandler) Retrieve(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/files/{file_id}"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	fileID, _ := ctx.UserValue("file_id").(string)
	if fileID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing file id", nil)
		return
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.FileRetrieve(gatewayCtx, key, fileID)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// Delete handles DELETE /v1/files/{file_id} and returns the bare
// FileDeleteResponse JSON.
func (h *FilesHandler) Delete(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/files/{file_id}"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	fileID, _ := ctx.UserValue("file_id").(string)
	if fileID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing file id", nil)
		return
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	resp, sperr := provider.FileDelete(gatewayCtx, key, fileID)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	h.writeJSON(ctx, &g, endpoint, "", key, raw, headers, resp)
}

// Content handles GET /v1/files/{file_id}/content. On success it writes the raw
// file bytes with Content-Type application/octet-stream (ESC-FILE-CONTENT-BYTES);
// it never JSON-marshals or wraps the body.
func (h *FilesHandler) Content(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/files/{file_id}/content"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	fileID, _ := ctx.UserValue("file_id").(string)
	if fileID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing file id", nil)
		return
	}

	provider, key, perr := h.resolveAndGate(ctx, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}
	if h.pendingTracker != nil {
		h.pendingTracker.Start("", key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}
	body, sperr := provider.FileContent(gatewayCtx, key, fileID)
	if sperr != nil {
		g.recordError(endpoint, "", key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType("application/octet-stream")
	ctx.SetBody(body)
	g.recordNonStream(endpoint, "", key.Provider, key.ID, raw, headers, 0, 0, nil)
}
