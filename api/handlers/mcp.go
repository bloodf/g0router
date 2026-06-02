package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type mcpClientRequest struct {
	Name      string            `json:"name"`
	Transport mcp.Transport     `json:"transport"`
	Command   *string           `json:"command"`
	Args      []string          `json:"args"`
	URL       *string           `json:"url"`
	Env       map[string]string `json:"env"`
	IsActive  bool              `json:"is_active"`
}

type mcpToolExecuteRequest struct {
	Arguments json.RawMessage `json:"arguments"`
}

func MCPClients(ctx *fasthttp.RequestCtx, s *store.Store, clients *mcp.ClientManager, tools *mcp.ToolManager, id string) {
	if s == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}

	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		list, err := s.ListMCPClients()
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list mcp clients: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[*store.MCPClient]{Data: list})
	case fasthttp.MethodPost:
		if clients == nil || tools == nil {
			writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp runtime unavailable")
			return
		}
		client, ok := decodeMCPClientRequest(ctx)
		if !ok {
			return
		}
		if err := s.CreateMCPClient(client); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create mcp client: %v", err))
			return
		}
		manifest, err := registerMCPClient(clients, tools, client)
		if err != nil {
			_ = s.DeleteMCPClient(client.ID)
			writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("register mcp client: %v", err))
			return
		}
		if err := s.UpdateMCPClientManifest(client.ID, manifest); err != nil {
			_ = clients.Close(client.ID)
			_ = s.DeleteMCPClient(client.ID)
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("cache mcp manifest: %v", err))
			return
		}
		got, err := s.GetMCPClient(client.ID)
		if err != nil {
			writeStoreError(ctx, "get mcp client", err)
			return
		}
		writeJSON(ctx, fasthttp.StatusCreated, got)
	case fasthttp.MethodDelete:
		if id == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "mcp client id required")
			return
		}
		if clients != nil {
			if err := clients.Close(id); err != nil && !errors.Is(err, mcp.ErrClientNotFound) {
				writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("close mcp client: %v", err))
				return
			}
		}
		if err := s.DeleteMCPClient(id); err != nil {
			writeStoreError(ctx, "delete mcp client", err)
			return
		}
		ctx.SetStatusCode(fasthttp.StatusNoContent)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func MCPTools(ctx *fasthttp.RequestCtx, s *store.Store, tools *mcp.ToolManager, name string) {
	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		compact, err := compactToolList(s, tools)
		if err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list mcp tools: %v", err))
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, listResponse[providers.Tool]{Data: compact})
	case fasthttp.MethodPost:
		if tools == nil {
			writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp tools unavailable")
			return
		}
		if name == "" {
			writeError(ctx, fasthttp.StatusBadRequest, "mcp tool name required")
			return
		}
		req, ok := decodeMCPToolExecuteRequest(ctx)
		if !ok {
			return
		}
		result, err := tools.Call(context.Background(), name, req.Arguments)
		if err != nil {
			writeMCPToolError(ctx, err)
			return
		}
		writeJSON(ctx, fasthttp.StatusOK, result)
	default:
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	}
}

func decodeMCPClientRequest(ctx *fasthttp.RequestCtx) (*store.MCPClient, bool) {
	var req mcpClientRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return nil, false
	}
	return &store.MCPClient{
		Name:      req.Name,
		Transport: req.Transport,
		Command:   req.Command,
		Args:      req.Args,
		URL:       req.URL,
		Env:       req.Env,
		IsActive:  req.IsActive,
	}, true
}

func decodeMCPToolExecuteRequest(ctx *fasthttp.RequestCtx) (mcpToolExecuteRequest, bool) {
	var req mcpToolExecuteRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return mcpToolExecuteRequest{}, false
	}
	if len(req.Arguments) == 0 {
		req.Arguments = json.RawMessage(`{}`)
	}
	return req, true
}

func registerMCPClient(clients *mcp.ClientManager, tools *mcp.ToolManager, client *store.MCPClient) (mcp.Manifest, error) {
	manifest, err := clients.Register(context.Background(), client.ClientConfig())
	if err != nil {
		return mcp.Manifest{}, err
	}
	if err := tools.RegisterManifest(manifest); err != nil {
		_ = clients.Close(client.ID)
		return mcp.Manifest{}, err
	}
	registered, ok := clients.Client(client.ID)
	if !ok {
		return mcp.Manifest{}, mcp.ErrClientNotFound
	}
	tools.RegisterClient(client.ID, registered)
	return manifest, nil
}

func compactToolList(s *store.Store, tools *mcp.ToolManager) ([]providers.Tool, error) {
	if tools != nil {
		return tools.CompactTools(), nil
	}
	if s == nil {
		return nil, nil
	}

	clients, err := s.ListMCPClients()
	if err != nil {
		return nil, err
	}
	var compact []providers.Tool
	for _, client := range clients {
		if client.ToolManifest == nil {
			continue
		}
		manifest, err := mcp.BuildCompactManifest(*client.ToolManifest)
		if err != nil {
			return nil, err
		}
		compact = append(compact, manifest.Tools...)
	}
	return compact, nil
}

func writeMCPToolError(ctx *fasthttp.RequestCtx, err error) {
	switch {
	case errors.Is(err, mcp.ErrToolNotFound), errors.Is(err, mcp.ErrClientNotFound):
		writeError(ctx, fasthttp.StatusNotFound, err.Error())
	default:
		writeError(ctx, fasthttp.StatusBadGateway, fmt.Sprintf("execute mcp tool: %v", err))
	}
}
