package handlers

import (
	"encoding/json"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// MCPClients DELETE with live managers closes the client and unregisters tools.
func TestMCPClientsDeleteWithRuntime(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &fakeMCPClient{tools: []mcp.Tool{{Name: "search", Description: "Search", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	manager := mcp.NewClientManager(&fakeMCPConnector{client: client})
	tools := mcp.NewToolManager()

	create := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	create.Request.SetBodyString(`{"name":"docs","transport":"stdio","command":"mcp-docs","is_active":true}`)
	MCPClients(create, s, manager, tools, "")
	if create.Response.StatusCode() != fasthttp.StatusCreated {
		t.Fatalf("create status = %d; body=%s", create.Response.StatusCode(), create.Response.Body())
	}
	var created store.MCPClient
	if err := json.Unmarshal(create.Response.Body(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	del := newHandlerCtx(fasthttp.MethodDelete, "/api/mcp/clients/"+created.ID)
	MCPClients(del, s, manager, tools, created.ID)
	if del.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", del.Response.StatusCode(), del.Response.Body())
	}
}

// MCPClients POST invalid JSON with live managers -> 400.
func TestMCPClientsPostInvalidJSON(t *testing.T) {
	s := openMCPHandlerStore(t)
	manager := mcp.NewClientManager(&fakeMCPConnector{client: &fakeMCPClient{}})
	tools := mcp.NewToolManager()
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/clients")
	ctx.Request.SetBodyString(`{bad`)
	MCPClients(ctx, s, manager, tools, "")
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

// MCPInstances DELETE with runtime returns 204 and closes the instance.
func TestMCPInstancesDeleteWithRuntime(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := createHandlerMCPInstance(t, s, "atlassian-del", "account-del")
	runtime := &fakeMCPInstanceRuntime{}
	ctx := newHandlerCtx(fasthttp.MethodDelete, "/api/mcp/instances/"+instance.ID)
	MCPInstances(ctx, s, runtime, instance.ID)
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if len(runtime.closed) != 1 || runtime.closed[0] != instance.ID {
		t.Fatalf("closed = %+v, want instance", runtime.closed)
	}
}

// MCPOAuthCallback success path builds the callback URL from the full URI.
func TestMCPOAuthCallbackSuccess(t *testing.T) {
	completer := &fakeMCPOAuthCompleter{}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/oauth/callback?instance_id=inst-1&code=c&state=s")
	MCPOAuthCallback(ctx, completer, nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if completer.instanceID != "inst-1" {
		t.Fatalf("instance id = %q, want inst-1", completer.instanceID)
	}
}
