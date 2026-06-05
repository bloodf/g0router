package handlers

import (
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/valyala/fasthttp"
)

// Responses translation error: an unsupported input type yields a 400.
func TestResponsesTranslationUnsupportedInputType(t *testing.T) {
	body := `{"model":"gpt-4o","input":[{"type":"function","role":"user","content":[{"type":"input_text","text":"hi"}]}]}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Responses(ctx, &coverageEngine{resp: chatResp()})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), respBody)
	}
}

// MCPOAuthStart BuildOAuthStartFlow error: missing redirect_uri -> 400.
func TestMCPOAuthStartBuildFlowError(t *testing.T) {
	s := openMCPHandlerStore(t)
	instance := createHandlerMCPInstance(t, s, "atlassian-c", "account-c")
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/instances/"+instance.ID+"/auth/start")
	// authorization_url + resource_uri present, but no redirect_uri.
	ctx.Request.SetBodyString(`{"authorization_url":"https://auth.example/authorize","resource_uri":"https://mcp.example"}`)
	MCPOAuthStart(ctx, s, instance.ID)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// Messages rejects an object-shaped (non-array, non-string) content block.
func TestMessagesRejectsUnsupportedContentShape(t *testing.T) {
	body := `{"model":"claude","messages":[{"role":"user","content":{"unexpected":true}}]}`
	ctx, respBody := runHandler(t, fasthttp.MethodPost, body, func(ctx *fasthttp.RequestCtx) {
		Messages(ctx, &coverageEngine{resp: chatResp()})
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", ctx.Response.StatusCode(), respBody)
	}
}

// MCPTools POST validation: missing name and invalid JSON.
func TestMCPToolsPostValidation(t *testing.T) {
	tools := mcp.NewToolManager()
	s := openMCPHandlerStore(t)

	// Missing tool name -> 400.
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/mcp/tools//execute")
	ctx.Request.SetBodyString(`{"arguments":{}}`)
	MCPTools(ctx, s, tools, "")
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing name = %d, want 400", ctx.Response.StatusCode())
	}

	// Invalid JSON body -> 400.
	ctx = newHandlerCtx(fasthttp.MethodPost, "/api/mcp/tools/docs__search/execute")
	ctx.Request.SetBodyString(`{bad`)
	MCPTools(ctx, s, tools, "docs__search")
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
}
