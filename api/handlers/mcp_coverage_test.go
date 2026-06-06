package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// TestRedactURLInvalid covers the url.Parse error branch.
func TestRedactURLInvalid(t *testing.T) {
	got := redactURL("://bad-url")
	if got != mcp.RedactedValue {
		t.Fatalf("redactURL(://bad-url) = %q, want %q", got, mcp.RedactedValue)
	}
}

// TestCompactToolListNilClientManifest covers the client.ToolManifest == nil branch.
func TestCompactToolListNilClientManifest(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &store.MCPClient{Name: "no-manifest", Transport: mcp.TransportStdio}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, nil, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// TestCompactInstanceToolListNilManifest covers the instance.ToolManifest == nil branch.
func TestCompactInstanceToolListNilManifest(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := &store.MCPInstance{Name: "no-manifest", ServerKey: "k", LaunchType: mcp.LaunchHTTP, Transport: mcp.TransportStreamableHTTP, URL: stringPtr("https://example.com/mcp"), IsActive: true}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools?instance_id="+instance.ID)
	MCPTools(ctx, s, nil, "")
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// TestMCPClientsDeleteCloseError covers the clients.Close error branch (not ErrClientNotFound).
func TestMCPClientsDeleteCloseErrorPath(t *testing.T) {
	s := openMCPHandlerStore(t)
	client := &store.MCPClient{Name: "close-err", Transport: mcp.TransportStdio, Command: stringPtr("cmd")}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}

	fc := &fakeMCPClient2{closeErr: errors.New("close boom")}
	manager := mcp.NewClientManager(&fakeMCPConnector2{client: fc})
	// Register the client so Close will be called on our fake.
	_, _ = manager.Register(context.Background(), client.ClientConfig())

	ctx := newHandlerCtx(fasthttp.MethodDelete, "/api/mcp/clients/"+client.ID)
	MCPClients(ctx, s, manager, nil, client.ID)
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// TestCompactToolListInstanceStoreError covers the compactInstanceToolList error branch.
func TestCompactToolListInstanceStoreErrorPath(t *testing.T) {
	s := openMCPHandlerStore(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools")
	MCPTools(ctx, s, nil, "")
	if ctx.Response.StatusCode() != fasthttp.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// ---- fakes ----

type fakeMCPClient2 struct {
	closeErr error
}

func (f *fakeMCPClient2) ListTools(ctx context.Context) ([]mcp.Tool, error) { return nil, nil }
func (f *fakeMCPClient2) CallTool(ctx context.Context, req mcp.CallRequest) (mcp.CallResult, error) {
	return mcp.CallResult{}, nil
}
func (f *fakeMCPClient2) Close() error { return f.closeErr }

type fakeMCPConnector2 struct {
	client *fakeMCPClient2
}

func (f *fakeMCPConnector2) Connect(ctx context.Context, cfg mcp.ClientConfig) (mcp.Client, error) {
	return f.client, nil
}
