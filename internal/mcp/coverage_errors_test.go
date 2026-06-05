package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- stdio: initialize error -> ensureInitialized error propagates to ListTools/CallTool ---

func TestStdioListToolsInitializeError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcError(req["id"], -32000, "init refused")
	})
	defer server.Close()
	client := NewStdioClient(process)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("ListTools init error: want error")
	}
}

func TestStdioCallToolInitializeError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcError(req["id"], -32000, "init refused")
	})
	defer server.Close()
	client := NewStdioClient(process)
	if _, err := client.CallTool(context.Background(), CallRequest{Name: "x"}); err == nil {
		t.Fatal("CallTool init error: want error")
	}
}

func TestStdioListToolsListError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		if req["method"] == "initialize" {
			return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
		}
		return rpcError(req["id"], -32000, "list refused")
	})
	defer server.Close()
	client := NewStdioClient(process)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("ListTools list error: want error")
	}
}

func TestStdioListToolsSnakeCaseSchema(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		switch req["method"] {
		case "initialize":
			return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/list":
			return rpcResult(req["id"], map[string]any{
				"tools": []map[string]any{
					{"name": "s", "input_schema": map[string]any{"type": "object"}},
				},
			})
		default:
			return rpcError(req["id"], -32601, "x")
		}
	})
	defer server.Close()
	client := NewStdioClient(process)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || !bytes.Contains(tools[0].InputSchema, []byte(`"object"`)) {
		t.Fatalf("snake schema not used: %+v", tools)
	}
}

// --- stdio: result decode error ---

func TestStdioCallToolResultDecodeError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		switch req["method"] {
		case "initialize":
			return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/call":
			// content as object where struct expects any - actually any accepts it;
			// force decode failure by returning result as a JSON string for a struct target.
			return map[string]any{"jsonrpc": "2.0", "id": req["id"], "result": "notanobject"}
		default:
			return rpcError(req["id"], -32601, "x")
		}
	})
	defer server.Close()
	client := NewStdioClient(process)
	if _, err := client.CallTool(context.Background(), CallRequest{Name: "x"}); err == nil {
		t.Fatal("CallTool decode error: want error")
	}
}

// --- stdio: context already cancelled before callLocked ---

func TestStdioCallLockedPreCancelled(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
	})
	defer server.Close()
	client := NewStdioClient(process)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.ListTools(ctx); err == nil {
		t.Fatal("pre-cancelled ctx: want error")
	}
}

// --- stdio: marshal error (unmarshalable params via CallTool with bad arguments is normalized;
// use notifyLocked with unmarshalable param to hit marshal error branch) ---

func TestStdioNotifyMarshalError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
	})
	defer server.Close()
	client := NewStdioClient(process)
	if err := client.notifyLocked("x", map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("notify marshal error: want error")
	}
}

// --- stdio: readLoop fails all pending when process output closes ---

func TestStdioReadLoopFailsPendingOnClose(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		switch req["method"] {
		case "initialize":
			return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/list":
			return rpcResult(req["id"], map[string]any{"tools": []map[string]any{}})
		default:
			return rpcError(req["id"], -32601, "x")
		}
	})
	client := NewStdioClient(process)
	// Initialize + first list succeeds (starts the read loop).
	if _, err := client.ListTools(context.Background()); err != nil {
		t.Fatalf("first ListTools: %v", err)
	}
	// Close the process so the read loop sees EOF and fails any future read; a
	// subsequent call's read errors out via failAllPending.
	if err := process.Close(); err != nil {
		t.Fatalf("process Close: %v", err)
	}
	server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := client.CallTool(ctx, CallRequest{Name: "x"}); err == nil {
		t.Fatal("call after process close: want error")
	}
}

// --- rawArguments: invalid json -> empty object ---

func TestRawArgumentsBranches(t *testing.T) {
	if got := rawArguments(nil); !isEmptyObject(got) {
		t.Fatalf("nil -> empty object, got %#v", got)
	}
	if got := rawArguments(json.RawMessage(`notjson`)); !isEmptyObject(got) {
		t.Fatalf("bad json -> empty object, got %#v", got)
	}
	if got := rawArguments(json.RawMessage(`{"a":1}`)); isEmptyObject(got) {
		t.Fatalf("valid json should decode, got %#v", got)
	}
}

func isEmptyObject(v any) bool {
	m, ok := v.(map[string]any)
	return ok && len(m) == 0
}

// --- oauth metadata helpers: request build error via control-char URL ---

func TestOAuthMetadataHelpersRequestBuildError(t *testing.T) {
	ctx := context.Background()
	bad := "http://\x7f/x"
	if _, err := protectedResourceMetadataURL(ctx, http.DefaultClient, bad); err == nil {
		t.Fatal("protectedResourceMetadataURL build: want error")
	}
	if _, err := fetchProtectedResourceMetadata(ctx, http.DefaultClient, bad); err == nil {
		t.Fatal("fetchProtectedResourceMetadata build: want error")
	}
	if _, err := fetchAuthorizationServerMetadata(ctx, http.DefaultClient, bad); err == nil {
		t.Fatal("fetchAuthorizationServerMetadata build: want error")
	}
}

// --- oauth: discoverTokenEndpoint request build error & network error ---

func TestDiscoverTokenEndpointNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	engine := NewOAuthEngine(newFakeOAuthStore(), server.Client())
	server.Close()
	if _, err := engine.discoverTokenEndpoint(context.Background(), url); err == nil {
		t.Fatal("discoverTokenEndpoint network: want error")
	}
}

// --- oauth: derivedTokenEndpointForFlow branches ---

func TestDerivedTokenEndpointForFlow(t *testing.T) {
	// Has /authorize suffix -> derived token endpoint.
	if got, ok := derivedTokenEndpointForFlow(OAuthFlow{AuthorizationURL: "https://a/oauth/authorize?x=1"}); !ok || got != "https://a/oauth/token" {
		t.Fatalf("authorize suffix: got %q ok=%v", got, ok)
	}
	// Root path -> not derivable.
	if _, ok := derivedTokenEndpointForFlow(OAuthFlow{AuthorizationURL: "https://a/"}); ok {
		t.Fatal("root path should not derive")
	}
	// Non-authorize path -> not derivable.
	if _, ok := derivedTokenEndpointForFlow(OAuthFlow{AuthorizationURL: "https://a/login"}); ok {
		t.Fatal("non-authorize path should not derive")
	}
	// Unparseable URL.
	if _, ok := derivedTokenEndpointForFlow(OAuthFlow{AuthorizationURL: "://bad"}); ok {
		t.Fatal("bad url should not derive")
	}
}

// --- oauth: redirectWithInstanceID bad url -> BuildOAuthStartFlow error ---

func TestBuildOAuthStartFlowBadRedirect(t *testing.T) {
	_, err := BuildOAuthStartFlow(OAuthStartConfig{
		InstanceID:       "inst",
		AuthorizationURL: "https://a/authorize",
		RedirectURI:      "://bad",
		ResourceURI:      "https://r",
	})
	if err == nil {
		t.Fatal("bad redirect uri: want error")
	}
}

// --- oauth: bearerResourceMetadataURL branches ---

func TestBearerResourceMetadataURLBranches(t *testing.T) {
	// Non-bearer header skipped; bearer with valid resource_metadata returned.
	headers := []string{
		`Basic realm="x"`,
		`Bearer realm="mcp", resource_metadata="https://meta.example/x"`,
	}
	if got := bearerResourceMetadataURL(headers); got != "https://meta.example/x" {
		t.Fatalf("got %q", got)
	}
	// Bearer with no resource_metadata param.
	if got := bearerResourceMetadataURL([]string{`Bearer realm="mcp"`}); got != "" {
		t.Fatalf("no metadata param: got %q", got)
	}
	// Bearer with invalid url value.
	if got := bearerResourceMetadataURL([]string{`Bearer resource_metadata="not a url"`}); got != "" {
		t.Fatalf("invalid url: got %q", got)
	}
	// Empty headers.
	if got := bearerResourceMetadataURL(nil); got != "" {
		t.Fatalf("nil headers: got %q", got)
	}
}

// --- httpclient streamable: callLocked marshal error path is unreachable (params always marshalable);
// cover ListTools/CallTool error propagation + notifyLocked network error via doer ---

func TestStreamableListToolsCallLockedError(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("net down")
	})
	c := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	if _, err := c.ListTools(context.Background()); err == nil {
		t.Fatal("ListTools net error: want error")
	}
	if _, err := c.CallTool(context.Background(), CallRequest{Name: "x"}); err == nil {
		t.Fatal("CallTool net error: want error")
	}
}

// --- httpclient SSE: callLocked read error (server closes after endpoint) ---

func TestSSECallLockedReadError(t *testing.T) {
	// Server advertises endpoint then closes the stream so readSSEData hits EOF.
	mux := http.NewServeMux()
	closeCh := make(chan struct{})
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: endpoint\ndata: /messages\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-closeCh
	})
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		// Accept the POST, then close the SSE stream so the awaited read fails.
		w.WriteHeader(http.StatusAccepted)
		close(closeCh)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewSSEClient(server.Client(), server.URL, nil)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := client.ListTools(ctx); err == nil {
		t.Fatal("SSE read error: want error")
	}
}

// --- httpclient SSE notifyLocked error (post fails) covered via post non-2xx already;
// cover readSSEEndpoint error when stream closes before endpoint event ---

func TestSSEEnsureEndpointNoEndpointEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Send a non-endpoint event then close -> readSSEEndpoint loops then EOF.
		_, _ = io.WriteString(w, "event: ping\ndata: x\n\n")
	}))
	defer server.Close()
	client := NewSSEClient(server.Client(), server.URL, nil)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("no endpoint event: want error")
	}
}

// --- legacyInitializeStreamable network error ---

func TestLegacyInitializeStreamableNetworkError(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("dial fail")
	})
	tr := NewHTTPTransport(doer)
	if _, _, err := tr.InitializeStreamable(context.Background(), "http://x", nil); err == nil {
		t.Fatal("legacy init network error: want error")
	}
}

func TestInitializeSSEError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	tr := NewHTTPTransport(server.Client())
	if _, err := tr.InitializeSSE(context.Background(), server.URL, nil); err == nil {
		t.Fatal("InitializeSSE error: want error")
	}
}

// --- toolmanager: Call provider error wrap + decodePropertySchemas nested error ---

func TestToolManagerCallClientError(t *testing.T) {
	tm := NewToolManager()
	tm.RegisterClient("docs", &fakeClient{err: errors.New("call boom")})
	if err := tm.RegisterManifest(Manifest{ClientID: "docs", Tools: []Tool{{Name: "t"}}}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	if _, err := tm.Call(context.Background(), "docs__t", json.RawMessage(`{}`)); err == nil {
		t.Fatal("Call client error: want error")
	}
}

func TestValidateToolArgumentsNestedPropertyError(t *testing.T) {
	// Nested property schema with a non-object property schema -> decodePropertySchemas
	// inner unmarshal error path, and nested type mismatch.
	schema := json.RawMessage(`{"type":"object","properties":{"a":{"type":"object","properties":{"b":{"type":"string"}}}}}`)
	if err := validateToolArguments(schema, json.RawMessage(`{"a":{"b":5}}`)); err == nil {
		t.Fatal("nested type mismatch: want error")
	}
}

func TestMatchesJSONTypeNull(t *testing.T) {
	if !matchesJSONType(nil, "null") {
		t.Fatal("null type should match nil")
	}
	if matchesJSONType("x", "null") {
		t.Fatal("non-nil should not match null")
	}
}

// --- constructors default to http.DefaultClient when client is nil ---

func TestTransportConstructorsNilClientDefault(t *testing.T) {
	if c := NewStreamableHTTPClient(nil, "http://x", nil, "", false); c.client == nil {
		t.Fatal("NewStreamableHTTPClient nil client should default")
	}
	if c := NewSSEClient(nil, "http://x", nil); c.client == nil {
		t.Fatal("NewSSEClient nil client should default")
	}
}
