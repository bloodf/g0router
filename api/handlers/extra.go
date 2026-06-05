package handlers

import (
	"context"
	"encoding/json"
	"io"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/valyala/fasthttp"
)

// ExtraEngine exposes the optional OpenAI-compatible capabilities (embeddings,
// images, audio) dispatched per-model. It is satisfied by *proxy.Engine. These
// handlers translate the upstream ErrCapabilityUnsupported into a 501 via
// writeDispatchError.
type ExtraEngine interface {
	Embeddings(ctx context.Context, req *providers.EmbeddingsRequest) (*providers.EmbeddingsResponse, error)
	GenerateImages(ctx context.Context, req *providers.ImagesRequest) (*providers.ImagesResponse, error)
	TranscribeAudio(ctx context.Context, req *providers.AudioTranscriptionRequest) (*providers.AudioResponse, error)
	Speech(ctx context.Context, req *providers.SpeechRequest) ([]byte, string, error)
}

func Embeddings(ctx *fasthttp.RequestCtx, engine ExtraEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	var req providers.EmbeddingsRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Model == "" {
		writeOpenAIError(ctx, fasthttp.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}

	resp, err := engine.Embeddings(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, resp)
}

func Images(ctx *fasthttp.RequestCtx, engine ExtraEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	var req providers.ImagesRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Prompt == "" {
		writeOpenAIError(ctx, fasthttp.StatusBadRequest, "prompt is required", "invalid_request_error", "missing_prompt")
		return
	}

	resp, err := engine.GenerateImages(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, resp)
}

func AudioTranscription(ctx *fasthttp.RequestCtx, engine ExtraEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid multipart form")
		return
	}

	model := firstFormValue(form.Value["model"])
	if model == "" {
		writeOpenAIError(ctx, fasthttp.StatusBadRequest, "model is required", "invalid_request_error", "missing_model")
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		writeOpenAIError(ctx, fasthttp.StatusBadRequest, "file is required", "invalid_request_error", "missing_file")
		return
	}
	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "cannot read uploaded file")
		return
	}
	data, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "cannot read uploaded file")
		return
	}

	req := providers.AudioTranscriptionRequest{
		Model:          model,
		File:           data,
		Filename:       fileHeader.Filename,
		Language:       firstFormValue(form.Value["language"]),
		Prompt:         firstFormValue(form.Value["prompt"]),
		ResponseFormat: firstFormValue(form.Value["response_format"]),
		Temperature:    firstFormValue(form.Value["temperature"]),
	}

	resp, err := engine.TranscribeAudio(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, resp)
}

func Speech(ctx *fasthttp.RequestCtx, engine ExtraEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	var req providers.SpeechRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Model == "" || req.Input == "" || req.Voice == "" {
		writeOpenAIError(ctx, fasthttp.StatusBadRequest, "model, input and voice are required", "invalid_request_error", "missing_field")
		return
	}

	audio, contentType, err := engine.Speech(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	if contentType == "" {
		contentType = "audio/mpeg"
	}
	ctx.SetContentType(contentType)
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(audio)
}

func firstFormValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
