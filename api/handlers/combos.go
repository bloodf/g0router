package handlers

import (
	"encoding/json"
	"log"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type comboRequest struct {
	Name         string            `json:"name"`
	Steps        []store.ComboStep `json:"steps"`
	Strategy     string            `json:"strategy"`
	IsActive     bool              `json:"is_active"`
	MCPToolGroup string            `json:"mcp_tool_group"`
}

type combosStore interface {
	ListCombos() ([]*store.Combo, error)
	CreateCombo(*store.Combo) error
	UpdateCombo(*store.Combo) error
	GetCombo(string) (*store.Combo, error)
	DeleteCombo(string) error
}

func Combos(ctx *fasthttp.RequestCtx, s combosStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		combos, err := s.ListCombos()
		if err != nil {
			log.Printf("list combos: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to list combos")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[*store.Combo]{Data: combos})
	case fasthttp.MethodPost:
		combo, ok := decodeComboRequest(ctx)
		if !ok {
			return
		}
		if err := s.CreateCombo(combo); err != nil {
			log.Printf("create combo: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create combo")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, combo)
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "combo id required")
			return
		}
		combo, ok := decodeComboRequest(ctx)
		if !ok {
			return
		}
		combo.ID = id
		if err := s.UpdateCombo(combo); err != nil {
			writeStoreError(ctx, "update combo", err)
			return
		}
		got, err := s.GetCombo(id)
		if err != nil {
			writeStoreError(ctx, "get combo", err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, got)
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "combo id required")
			return
		}
		if err := s.DeleteCombo(id); err != nil {
			writeStoreError(ctx, "delete combo", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func decodeComboRequest(ctx *fasthttp.RequestCtx) (*store.Combo, bool) {
	var req comboRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return nil, false
	}
	strategy, err := store.NormalizeComboStrategy(req.Strategy)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid strategy")
		return nil, false
	}
	return &store.Combo{
		Name:         req.Name,
		Steps:        req.Steps,
		Strategy:     strategy,
		IsActive:     req.IsActive,
		MCPToolGroup: req.MCPToolGroup,
	}, true
}
