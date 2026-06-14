package mcp

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// jsonResp builds a canned JSON HTTP response.
func jsonResp(body string) fakeResp {
	return fakeResp{status: http.StatusOK, body: body, headers: map[string]string{"Content-Type": "application/json"}}
}

const initOKBody = `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{}}}`

func TestProbeFullHandshakeJSON(t *testing.T) {
	toolsBody := `{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"search","description":"web search"},{"name":"fetch"}]}}`
	ft := &fakeTransport{responses: []fakeResp{
		{status: http.StatusOK, body: initOKBody, headers: map[string]string{"mcp-session-id": "sess-123", "Content-Type": "application/json"}},
		{status: http.StatusAccepted}, // notifications/initialized
		jsonResp(toolsBody),
	}}
	p := NewProbe(fakeClient(ft))

	res := p.Run(context.Background(), "https://srv.example/mcp")
	if res.Error != "" {
		t.Fatalf("unexpected error: %q", res.Error)
	}
	if res.RequiresAuth {
		t.Fatalf("RequiresAuth should be false")
	}
	if len(res.Tools) != 2 || res.Tools[0].Name != "search" || res.Tools[0].Description != "web search" || res.Tools[1].Name != "fetch" {
		t.Fatalf("tools = %#v", res.Tools)
	}

	// All three requests must carry the protocol-version header (PAR-MCP-010).
	if len(ft.captured) != 3 {
		t.Fatalf("captured %d requests, want 3", len(ft.captured))
	}
	for i, req := range ft.captured {
		if got := req.Header.Get("MCP-Protocol-Version"); got != "2025-06-18" {
			t.Fatalf("request %d MCP-Protocol-Version = %q", i, got)
		}
		if got := req.Header.Get("Accept"); !strings.Contains(got, "text/event-stream") {
			t.Fatalf("request %d Accept = %q", i, got)
		}
	}
	// The session-id read from initialize must be replayed on requests 2 and 3
	// (PAR-MCP-011).
	if got := ft.captured[1].Header.Get("mcp-session-id"); got != "sess-123" {
		t.Fatalf("notifications/initialized session-id = %q", got)
	}
	if got := ft.captured[2].Header.Get("mcp-session-id"); got != "sess-123" {
		t.Fatalf("tools/list session-id = %q", got)
	}
	// The initialize request is id 1; tools/list is id 2.
	if !strings.Contains(ft.bodyAt(0), `"id":1`) || !strings.Contains(ft.bodyAt(0), `"method":"initialize"`) {
		t.Fatalf("initialize body = %s", ft.bodyAt(0))
	}
	if !strings.Contains(ft.bodyAt(2), `"id":2`) || !strings.Contains(ft.bodyAt(2), `"method":"tools/list"`) {
		t.Fatalf("tools/list body = %s", ft.bodyAt(2))
	}
}

func TestProbeToolsListSSEParse(t *testing.T) {
	// tools/list returns text/event-stream; the parser must find the id==2 result
	// among the data frames (PAR-MCP-012).
	sseBody := "event: message\n" +
		"data: {\"jsonrpc\":\"2.0\",\"id\":5,\"result\":{\"tools\":[]}}\n\n" +
		"data: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"tools\":[{\"name\":\"sse_tool\"}]}}\n\n"
	ft := &fakeTransport{responses: []fakeResp{
		{status: http.StatusOK, body: initOKBody, headers: map[string]string{"Content-Type": "application/json"}},
		{status: http.StatusAccepted},
		{status: http.StatusOK, body: sseBody, headers: map[string]string{"Content-Type": "text/event-stream"}},
	}}
	p := NewProbe(fakeClient(ft))
	res := p.Run(context.Background(), "https://srv.example/mcp")
	if res.Error != "" {
		t.Fatalf("error: %q", res.Error)
	}
	if len(res.Tools) != 1 || res.Tools[0].Name != "sse_tool" {
		t.Fatalf("tools = %#v", res.Tools)
	}
}

func TestProbeInitRequiresAuth(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		ft := &fakeTransport{responses: []fakeResp{{status: status, body: "no"}}}
		p := NewProbe(fakeClient(ft))
		res := p.Run(context.Background(), "https://srv.example/mcp")
		if !res.RequiresAuth {
			t.Fatalf("status %d: RequiresAuth should be true", status)
		}
		if res.Tools != nil {
			t.Fatalf("status %d: tools should be nil", status)
		}
	}
}

func TestProbeToolsListRequiresAuth(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		ft := &fakeTransport{responses: []fakeResp{
			jsonResp(initOKBody),
			{status: http.StatusAccepted},
			{status: status, body: "denied"},
		}}
		p := NewProbe(fakeClient(ft))
		res := p.Run(context.Background(), "https://srv.example/mcp")
		if !res.RequiresAuth {
			t.Fatalf("status %d: tools/list RequiresAuth should be true", status)
		}
	}
}

func TestProbeInitNon2xxError(t *testing.T) {
	ft := &fakeTransport{responses: []fakeResp{{status: http.StatusInternalServerError, body: "boom"}}}
	p := NewProbe(fakeClient(ft))
	res := p.Run(context.Background(), "https://srv.example/mcp")
	if res.RequiresAuth {
		t.Fatalf("RequiresAuth should be false for 500")
	}
	if !strings.Contains(res.Error, "init") {
		t.Fatalf("error = %q, want init-related", res.Error)
	}
}

func TestProbeTimeout(t *testing.T) {
	// A blocking transport + an already-short context drives the timeout path with
	// NO real 8s wait (PAR-MCP-058/059).
	ft := &fakeTransport{block: true}
	p := NewProbe(fakeClient(ft))
	ctx, cancel := shortCtx()
	defer cancel()
	res := p.Run(ctx, "https://srv.example/mcp")
	if res.Error != "timeout" {
		t.Fatalf("error = %q, want timeout", res.Error)
	}
}

func TestNewProbeNilClient(t *testing.T) {
	if NewProbe(nil) == nil {
		t.Fatalf("NewProbe(nil) returned nil")
	}
}
