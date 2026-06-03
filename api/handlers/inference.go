package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
	"github.com/valyala/fasthttp"
)

type errorResponse struct {
	Error string `json:"error"`
}

func Inference(ctx *fasthttp.RequestCtx, engine InferenceEngine) {
	if engine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "inference engine unavailable")
		return
	}

	var req providers.ChatRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Stream != nil && *req.Stream {
		streamInference(ctx, engine, &req)
		return
	}

	resp, err := engine.Dispatch(requestContext(ctx), &req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, resp)
}

func streamInference(ctx *fasthttp.RequestCtx, engine InferenceEngine, req *providers.ChatRequest) {
	stream, err := engine.DispatchStream(requestContext(ctx), req)
	if err != nil {
		writeDispatchError(ctx, err)
		return
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		for chunk := range stream {
			data, err := json.Marshal(chunk)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			_ = w.Flush()
		}
		_, _ = w.WriteString("data: [DONE]\n\n")
		_ = w.Flush()
	})
}

func writeDispatchError(ctx *fasthttp.RequestCtx, err error) {
	switch {
	case errors.Is(err, proxy.ErrProviderNotFound):
		writeError(ctx, fasthttp.StatusNotFound, err.Error())
	case errors.Is(err, proxy.ErrNoConnections):
		writeError(ctx, fasthttp.StatusServiceUnavailable, err.Error())
	default:
		writeError(ctx, fasthttp.StatusInternalServerError, err.Error())
	}
}

func writeJSON(ctx *fasthttp.RequestCtx, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "marshal response")
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	ctx.SetBody(body)
}

func writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	body, err := json.Marshal(errorResponse{Error: message})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	ctx.SetBody(body)
}
