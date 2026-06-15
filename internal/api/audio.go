package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strconv"

	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/valyala/fasthttp"
)

// AudioHandler handles POST /v1/audio/speech and /v1/audio/transcriptions.
// These are OpenAI-compatible routes: speech returns raw audio bytes and
// transcriptions returns the bare OpenAI JSON object — never the {data,error}
// admin envelope.
type AudioHandler struct {
	router         completionsResolver
	usageRecorder  UsageRecorder
	pendingTracker PendingTracker
	detailCapture  DetailCapture
	vkGate         *VKGate
	pinnedResolver VKPinnedKeyResolver
}

// NewAudioHandler creates an audio handler.
func NewAudioHandler(router *inference.Router) *AudioHandler {
	return &AudioHandler{router: router}
}

// SetUsageRecorder wires a usage recorder.
func (h *AudioHandler) SetUsageRecorder(r UsageRecorder) { h.usageRecorder = r }

// SetPendingTracker wires a pending tracker.
func (h *AudioHandler) SetPendingTracker(t PendingTracker) { h.pendingTracker = t }

// SetDetailCapture wires a detail capture.
func (h *AudioHandler) SetDetailCapture(d DetailCapture) { h.detailCapture = d }

// SetVKGate wires a virtual-key gate.
func (h *AudioHandler) SetVKGate(g *VKGate) { h.vkGate = g }

// SetVKPinnedResolver wires the virtual-key pinned-key resolver.
func (h *AudioHandler) SetVKPinnedResolver(r VKPinnedKeyResolver) { h.pinnedResolver = r }

func (h *AudioHandler) recordGlue() recordGlue {
	return recordGlue{recorder: h.usageRecorder, tracker: h.pendingTracker, detail: h.detailCapture}
}

// Speech handles POST /v1/audio/speech. On success it writes the raw audio
// bytes with the upstream Content-Type (ESC-SPEECH-BYTES); on stream:true it
// frames SSE; errors use the OpenAI error shape.
func (h *AudioHandler) Speech(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/audio/speech"
	raw := ctx.PostBody()
	headers := requestHeadersFromCtx(ctx)
	g := h.recordGlue()

	var req schemas.SpeechRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "invalid JSON body", nil)
		return
	}

	// SpeechRequest has no Stream field in the schema; detect stream from the
	// raw body so the streaming row is reachable without a schema change.
	stream := requestWantsStream(raw)

	provider, key, perr := h.resolveAndGate(ctx, req.Model, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}

	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if stream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ch, sperr := provider.SpeechStream(gatewayCtx, nil, key, &req)
		if sperr != nil {
			g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
			writeProviderError(ctx, sperr)
			return
		}
		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		if sErr := writeSSEStream(streamCtx, ctx, ch); sErr != nil {
			log.Printf("audio speech stream error: %v", sErr)
		}
		return
	}

	resp, sperr := provider.Speech(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}

	contentType := resp.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType(contentType)
	ctx.SetBody(resp.Audio)

	g.recordNonStream(endpoint, req.Model, key.Provider, key.ID, raw, headers, 0, 0, nil)
}

// Transcription handles POST /v1/audio/transcriptions. It parses a
// multipart/form-data upload (ESC-MULTIPART) and returns the bare
// TranscriptionResponse JSON or, on stream, SSE frames.
func (h *AudioHandler) Transcription(ctx *fasthttp.RequestCtx) {
	const endpoint = "/v1/audio/transcriptions"
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
	model := formValue(form, "model")
	if model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", "missing required field: model", nil)
		return
	}

	req := schemas.TranscriptionRequest{File: file, Model: model}
	if v := formValue(form, "language"); v != "" {
		req.Language = &v
	}
	if v := formValue(form, "prompt"); v != "" {
		req.Prompt = &v
	}
	if v := formValue(form, "response_format"); v != "" {
		req.ResponseFormat = &v
	}
	if v := formValue(form, "temperature"); v != "" {
		if f, perr := strconv.ParseFloat(v, 64); perr == nil {
			req.Temperature = &f
		}
	}
	if g, ok := form.Value["timestamp_granularities[]"]; ok {
		req.TimestampGranularities = g
	}
	stream := formValue(form, "stream") == "true"

	provider, key, perr := h.resolveAndGate(ctx, req.Model, raw, headers, endpoint, &g)
	if perr != nil {
		return
	}

	if h.pendingTracker != nil {
		h.pendingTracker.Start(req.Model, key.Provider, key.ID)
	}

	gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}

	if stream {
		ctx.SetContentTypeBytes([]byte("text/event-stream"))
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		ch, sperr := provider.TranscriptionStream(gatewayCtx, nil, key, &req)
		if sperr != nil {
			g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
			writeProviderError(ctx, sperr)
			return
		}
		streamCtx, cancel := withRequestCancel(ctx)
		defer cancel()
		if sErr := writeSSEStream(streamCtx, ctx, ch); sErr != nil {
			log.Printf("audio transcription stream error: %v", sErr)
		}
		return
	}

	resp, sperr := provider.Transcription(gatewayCtx, key, &req)
	if sperr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, sperr)
		writeProviderError(ctx, sperr)
		return
	}

	b, mErr := jsonMarshal(resp)
	if mErr != nil {
		g.recordError(endpoint, req.Model, key.Provider, key.ID, raw, headers, &schemas.ProviderError{StatusCode: 500, Message: "marshal failure", Type: "internal"})
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentTypeBytes([]byte("text/plain"))
		ctx.SetBodyString("internal error")
		return
	}
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes([]byte("application/json"))
	ctx.SetBody(b)

	g.recordNonStream(endpoint, req.Model, key.Provider, key.ID, raw, headers, 0, 0, resp)
}

// resolveAndGate resolves the model and applies the x-g0-vk gate + pinning,
// mirroring completions.go. It writes the error response and returns a non-nil
// error sentinel when the request must stop; otherwise the resolved provider
// and (possibly pinned) key. g.apiKey is populated on admission.
func (h *AudioHandler) resolveAndGate(ctx *fasthttp.RequestCtx, model string, raw []byte, headers map[string]string, endpoint string, g *recordGlue) (schemas.Provider, schemas.Key, *schemas.ProviderError) {
	provider, key, err := h.router.Resolve(model)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid_request_error", err.Error(), nil)
		return nil, schemas.Key{}, &schemas.ProviderError{StatusCode: 400}
	}

	vkHeader := string(ctx.Request.Header.Peek("x-g0-vk"))
	if vkHeader != "" {
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
	}
	return provider, key, nil
}

// writeProviderError maps a *ProviderError to an OpenAI error response.
func writeProviderError(ctx *fasthttp.RequestCtx, perr *schemas.ProviderError) {
	status := perr.StatusCode
	if status == 0 {
		status = fasthttp.StatusBadGateway
	}
	writeError(ctx, status, perr.Type, perr.Message, perr.Code)
}

// requestWantsStream reports whether a JSON body sets stream:true. It is used
// for endpoints whose request schema does not model a Stream field.
func requestWantsStream(raw []byte) bool {
	var probe struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return false
	}
	return probe.Stream
}

// isMultipart reports whether the request Content-Type is multipart/form-data.
func isMultipart(ctx *fasthttp.RequestCtx) bool {
	return bytes.HasPrefix(ctx.Request.Header.ContentType(), []byte("multipart/form-data"))
}

// formValue returns the first value for a multipart form field, or "".
func formValue(form *multipart.Form, field string) string {
	if vs, ok := form.Value[field]; ok && len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// readMultipartFile reads the first file part for field into a byte slice. The
// bool reports whether the part was present.
func readMultipartFile(form *multipart.Form, field string) ([]byte, bool, error) {
	fhs, ok := form.File[field]
	if !ok || len(fhs) == 0 {
		return nil, false, nil
	}
	f, err := fhs[0].Open()
	if err != nil {
		return nil, true, fmt.Errorf("open file part %q: %w", field, err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, true, fmt.Errorf("read file part %q: %w", field, err)
	}
	return data, true, nil
}
