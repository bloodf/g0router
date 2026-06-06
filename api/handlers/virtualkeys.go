package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type createVirtualKeyRequest struct {
	Name         string   `json:"name"`
	TeamID       *int64   `json:"team_id"`
	BudgetUSD    *float64 `json:"budget_usd"`
	BudgetPeriod string   `json:"budget_period"`
	RateLimitRPM *int     `json:"rate_limit_rpm"`
	RateLimitTPM *int     `json:"rate_limit_tpm"`
	MCPToolGroup string   `json:"mcp_tool_group"`
}

type updateVirtualKeyRequest struct {
	Name         string   `json:"name"`
	TeamID       *int64   `json:"team_id"`
	BudgetUSD    *float64 `json:"budget_usd"`
	BudgetPeriod string   `json:"budget_period"`
	RateLimitRPM *int     `json:"rate_limit_rpm"`
	RateLimitTPM *int     `json:"rate_limit_tpm"`
	IsActive     bool     `json:"is_active"`
	MCPToolGroup string   `json:"mcp_tool_group"`
}

type virtualKeyView struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	KeyPrefix     string   `json:"key_prefix"`
	BudgetUSD     *float64 `json:"budget_usd"`
	BudgetPeriod  string   `json:"budget_period"`
	BudgetUsedUSD float64  `json:"budget_used_usd"`
	RateLimitRPM  *int     `json:"rate_limit_rpm"`
	RateLimitTPM  *int     `json:"rate_limit_tpm"`
	TeamID        *int64   `json:"team_id"`
	IsActive      bool     `json:"is_active"`
	MCPToolGroup  string   `json:"mcp_tool_group"`
	CreatedAt     string   `json:"created_at"`
}

func newVirtualKeyView(key store.VirtualKey) virtualKeyView {
	return virtualKeyView{
		ID:            key.ID,
		Name:          key.Name,
		KeyPrefix:     key.KeyPrefix,
		BudgetUSD:     key.BudgetUSD,
		BudgetPeriod:  key.BudgetPeriod,
		BudgetUsedUSD: key.BudgetUsedUSD,
		RateLimitRPM:  key.RateLimitRPM,
		RateLimitTPM:  key.RateLimitTPM,
		TeamID:        key.TeamID,
		IsActive:      key.IsActive,
		MCPToolGroup:  key.MCPToolGroup,
		CreatedAt:     key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type createVirtualKeyResponse struct {
	Key virtualKeyView `json:"key"`
	Raw string         `json:"raw"`
}

type virtualKeyStore interface {
	ListVirtualKeys() ([]store.VirtualKey, error)
	CreateVirtualKey(name string, teamID *int64, budgetUSD *float64, budgetPeriod string, rateLimitRPM, rateLimitTPM *int, mcpToolGroup string) (*store.VirtualKey, string, error)
	GetVirtualKey(id int64) (*store.VirtualKey, error)
	UpdateVirtualKey(id int64, name string, teamID *int64, budgetUSD *float64, budgetPeriod string, rateLimitRPM, rateLimitTPM *int, isActive bool, mcpToolGroup string) error
	DeleteVirtualKey(id int64) error
}

func VirtualKeys(ctx *fasthttp.RequestCtx, s virtualKeyStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			keys, err := s.ListVirtualKeys()
			if err != nil {
				log.Printf("list virtual keys: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list virtual keys")
				return
			}
			views := make([]virtualKeyView, 0, len(keys))
			for _, key := range keys {
				views = append(views, newVirtualKeyView(key))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[virtualKeyView]{Data: views})
			return
		}
		keyID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid virtual key id")
			return
		}
		key, err := s.GetVirtualKey(keyID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "virtual key not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newVirtualKeyView(*key))
	case fasthttp.MethodPost:
		var req createVirtualKeyRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.Name) == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		key, raw, err := s.CreateVirtualKey(req.Name, req.TeamID, req.BudgetUSD, req.BudgetPeriod, req.RateLimitRPM, req.RateLimitTPM, req.MCPToolGroup)
		if err != nil {
			log.Printf("create virtual key: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create virtual key")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, createVirtualKeyResponse{Key: newVirtualKeyView(*key), Raw: raw})
	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "virtual key id required")
			return
		}
		keyID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid virtual key id")
			return
		}
		var req updateVirtualKeyRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if err := s.UpdateVirtualKey(keyID, req.Name, req.TeamID, req.BudgetUSD, req.BudgetPeriod, req.RateLimitRPM, req.RateLimitTPM, req.IsActive, req.MCPToolGroup); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "virtual key not found")
				return
			}
			log.Printf("update virtual key: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update virtual key")
			return
		}
		updated, err := s.GetVirtualKey(keyID)
		if err != nil {
			writeError(ctx, fasthttp.StatusNotFound, "virtual key not found")
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, newVirtualKeyView(*updated))
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "virtual key id required")
			return
		}
		keyID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid virtual key id")
			return
		}
		if err := s.DeleteVirtualKey(keyID); err != nil {
			log.Printf("delete virtual key: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete virtual key")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
