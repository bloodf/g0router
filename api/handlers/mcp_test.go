package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeMCPConnector struct {
	client *fakeMCPClient
	err    error
}

func (f *fakeMCPConnector) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.client.config = cfg
	return f.client, nil
}

type fakeMCPClient struct {
	config mcp.ClientConfig
	tools  []mcp.Tool
	calls  []mcp.CallRequest
	err    error
}

func (f *fakeMCPClient) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	return f.tools, f.err
}

func (f *fakeMCPClient) CallTool(ctx context.Context, req mcp.CallRequest) (mcp.CallResult, error) {
	f.calls = append(f.calls, req)
	return mcp.CallResult{Content: map[string]any{"ok": true}}, nil
}

func (f *fakeMCPClient) Close() error {
	return nil
}

func TestMCPClientsCreateDiscoversAndPersistsManifest(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &fakeMCPClient{tools: []mcp.Tool{{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	manager := mcp.NewClientManager(&fakeMCPConnector{client: client})
	tools := mcp.NewToolManager()

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	ctx.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","args":["--stdio"],"env":{"TOKEN":"secret"},"is_active":true}`)
	MCPClients(ctx, s, manager, tools, "")

	if ctx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var created store.MCPClient
	if err := json.Unmarshal(ctx.Response.Body(), &created); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if created.ID == "" || created.Name != "docs" {
		t.Fatalf("created = %+v, want docs with ID", created)
	}
	if client.config.ID != created.ID || client.config.Command != "mcp-docs" {
		t.Fatalf("config = %+v, want created ID and command", client.config)
	}

	got, err := s.GetMCPClient(created.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	if got.ToolManifest == nil || len(got.ToolManifest.Tools) != 1 {
		t.Fatalf("stored manifest = %+v, want one tool", got.ToolManifest)
	}

	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/mcp/clients")
	MCPClients(ctx, s, manager, tools, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var listed struct {
		Data []store.MCPClient `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].ID != created.ID {
		t.Fatalf("listed = %+v, want created client", listed.Data)
	}
}

func TestMCPClientsCreateRollsBackStoreWhenDiscoveryFails(t *testing.T) {
	s := openMCPHandlerStore(t)
	manager := mcp.NewClientManager(&fakeMCPConnector{client: &fakeMCPClient{err: errors.New("offline")}})
	tools := mcp.NewToolManager()

	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	ctx.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`)
	MCPClients(ctx, s, manager, tools, "")

	if ctx.Response.StatusCode() != fasthttp.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	clients, err := s.ListMCPClients()
	if err != nil {
		t.Fatalf("ListMCPClients: %v", err)
	}
	if len(clients) != 0 {
		t.Fatalf("clients = %+v, want rollback", clients)
	}
}

func TestMCPToolsListAndExecute(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &fakeMCPClient{tools: []mcp.Tool{{Name: "search", Description: "Search docs", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	manager := mcp.NewClientManager(&fakeMCPConnector{client: client})
	tools := mcp.NewToolManager()

	createCtx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	createCtx.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`)
	MCPClients(createCtx, s, manager, tools, "")
	if createCtx.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", createCtx.Response.StatusCode(), createCtx.Response.Body())
	}
	var created store.MCPClient
	if err := json.Unmarshal(createCtx.Response.Body(), &created); err != nil {
		t.Fatalf("unmarshal created client: %v", err)
	}
	toolName := created.ID + "__search"

	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, tools, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("list status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var listed struct {
		Data []struct {
			Type     string `json:"type"`
			Function struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters,omitempty"`
			} `json:"function"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &listed); err != nil {
		t.Fatalf("unmarshal tools: %v", err)
	}
	if len(listed.Data) != 1 || listed.Data[0].Function.Name != toolName {
		t.Fatalf("tools = %+v, want %s", listed.Data, toolName)
	}
	if listed.Data[0].Function.Parameters != nil {
		t.Fatalf("compact tool includes parameters: %s", listed.Data[0].Function.Parameters)
	}

	ctx = newHandlerCtx(fasthttp.MethodPost, "/api/mcp/tools/"+toolName+"/execute")
	ctx.Request.SetBodyString(`{"arguments":{"query":"mcp"}}`)
	MCPTools(ctx, s, tools, toolName)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("execute status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if len(client.calls) != 1 || client.calls[0].Name != "search" || string(client.calls[0].Arguments) != `{"query":"mcp"}` {
		t.Fatalf("calls = %+v, want search with args", client.calls)
	}
}

func TestMCPToolsExecuteMissingTool(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/tools/missing/execute")
	ctx.Request.SetBodyString(`{"arguments":{}}`)

	MCPTools(ctx, openMCPHandlerStore(t), mcp.NewToolManager(), "missing")

	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func openMCPHandlerStore(t *testing.T) *store.Store {
	t.Helper()

	s, err := store.NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return s
}
