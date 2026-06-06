package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type mcpToolGroupRequest struct {
	Name     string   `json:"name"`
	ToolIDs  []string `json:"tool_ids"`
	IsActive bool     `json:"is_active"`
}

type mcpToolGroupResponse struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	ToolIDs   []string `json:"tool_ids"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func toMCPToolGroupResponse(g store.MCPToolGroup) mcpToolGroupResponse {
	return mcpToolGroupResponse{
		ID:        g.ID,
		Name:      g.Name,
		ToolIDs:   g.ToolIDs,
		IsActive:  g.IsActive,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

type mcpToolGroupStore interface {
	ListMCPToolGroups() ([]store.MCPToolGroup, error)
	GetMCPToolGroup(id int64) (*store.MCPToolGroup, error)
	CreateMCPToolGroup(name string, toolIDs []string, isActive bool) (*store.MCPToolGroup, error)
	UpdateMCPToolGroup(id int64, name string, toolIDs []string, isActive bool) error
	DeleteMCPToolGroup(id int64) error
}

func MCPToolGroups(ctx *fasthttp.RequestCtx, s mcpToolGroupStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		if id == "" {
			groups, err := s.ListMCPToolGroups()
			if err != nil {
				log.Printf("list mcp tool groups: %v", err)
				writeError(ctx, fasthttp.StatusInternalServerError, "failed to list mcp tool groups")
				return
			}
			resp := make([]mcpToolGroupResponse, 0, len(groups))
			for _, g := range groups {
				resp = append(resp, toMCPToolGroupResponse(g))
			}
			writeJSON(ctx, fasthttp.StatusOK, listResponse[mcpToolGroupResponse]{Data: resp})
			return
		}

		groupID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
			return
		}
		group, err := s.GetMCPToolGroup(groupID)
		if err != nil {
			writeStoreError(ctx, "get mcp tool group", err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, toMCPToolGroupResponse(*group))

	case fasthttp.MethodPost:
		var req mcpToolGroupRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		group, err := s.CreateMCPToolGroup(req.Name, req.ToolIDs, req.IsActive)
		if err != nil {
			if errors.Is(err, store.ErrDuplicateName) {
				writeError(ctx, fasthttp.StatusConflict, "tool group name already exists")
				return
			}
			log.Printf("create mcp tool group: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to create mcp tool group")
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, toMCPToolGroupResponse(*group))

	case fasthttp.MethodPut:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "id required")
			return
		}
		groupID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
			return
		}
		var req mcpToolGroupRequest
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
			return
		}
		if req.Name == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "name is required")
			return
		}
		if err := s.UpdateMCPToolGroup(groupID, req.Name, req.ToolIDs, req.IsActive); err != nil {
			if errors.Is(err, store.ErrDuplicateName) {
				writeError(ctx, fasthttp.StatusConflict, "tool group name already exists")
				return
			}
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "tool group not found")
				return
			}
			log.Printf("update mcp tool group: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to update mcp tool group")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)

	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "id required")
			return
		}
		groupID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
			return
		}
		if err := s.DeleteMCPToolGroup(groupID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(ctx, fasthttp.StatusNotFound, "tool group not found")
				return
			}
			log.Printf("delete mcp tool group: %v", err)
			writeError(ctx, fasthttp.StatusInternalServerError, "failed to delete mcp tool group")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)

	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}
