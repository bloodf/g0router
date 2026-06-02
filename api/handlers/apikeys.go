package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type createAPIKeyRequest struct {
	Name string `json:"name"`
}

type createAPIKeyResponse struct {
	Key *store.APIKey `json:"key"`
	Raw string        `json:"raw"`
}

func APIKeys(ctx *fasthttp.RequestCtx, s *store.Store, secret, id string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		keys, err := s.ListAPIKeys()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list api keys: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[store.APIKey]{Data: keys})
	case fasthttp.MethodPost:
		var req createAPIKeyRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		key, raw, err := s.CreateAPIKey(req.Name, secret)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create api key: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, createAPIKeyResponse{Key: key, Raw: raw})
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "api key id required")
			return
		}
		if err := s.DeleteAPIKey(id); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("delete api key: %v", err))
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
