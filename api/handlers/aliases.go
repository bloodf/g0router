package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type aliasRequest struct {
	Alias    string `json:"alias"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func Aliases(ctx *fasthttp.RequestCtx, s *store.Store, aliasID string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		aliases, err := s.ListModelAliases()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list aliases: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[store.ModelAlias]{Data: aliases})
	case fasthttp.MethodPost:
		alias, ok := decodeAliasRequest(ctx, "")
		if !ok {
			return
		}
		if err := s.SetModelAlias(alias); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("set alias: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, alias)
	case fasthttp.MethodPut:
		if aliasID == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "alias required")
			return
		}
		alias, ok := decodeAliasRequest(ctx, aliasID)
		if !ok {
			return
		}
		if err := s.SetModelAlias(alias); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("set alias: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, alias)
	case fasthttp.MethodDelete:
		if aliasID == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "alias required")
			return
		}
		if err := s.DeleteModelAlias(aliasID); err != nil {
			writeStoreError(ctx, "delete alias", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func decodeAliasRequest(ctx *fasthttp.RequestCtx, aliasID string) (store.ModelAlias, bool) {
	var req aliasRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return store.ModelAlias{}, false
	}
	if aliasID != "" {
		req.Alias = aliasID
	}
	if req.Alias == "" || req.Provider == "" || req.Model == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "alias, provider, and model are required")
		return store.ModelAlias{}, false
	}
	return store.ModelAlias{Alias: req.Alias, Provider: req.Provider, Model: req.Model}, true
}
