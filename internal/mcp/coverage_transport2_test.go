package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// doerFunc adapts a function to HTTPDoer/OAuthHTTPClient.
type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func jsonResponse(status int, body string, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// --- StreamableHTTP ctx-cancel + cancelled notification ---

func TestStreamableHTTPCallLockedContextCancelSendsCancelled(t *testing.T) {
	var cancelledSeen atomic.Bool
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if strings.Contains(string(body), "notifications/cancelled") {
			cancelledSeen.Store(true)
			return jsonResponse(http.StatusAccepted, "", nil), nil
		}
		// Simulate the transport-level error after ctx cancellation.
		return nil, req.Context().Err()
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.ListTools(ctx); err == nil {
		t.Fatal("want error from cancelled context")
	}
	if !cancelledSeen.Load() {
		t.Fatal("notifications/cancelled was not sent after cancel")
	}
}

// --- StreamableHTTP non-2xx and read paths ---

func TestStreamableHTTPCallLockedNon2xx(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, "nope", nil), nil
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("want non-2xx error")
	}
}

func TestStreamableHTTPResponseBodySSE(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		h := http.Header{}
		h.Set("Content-Type", "text/event-stream")
		h.Set("Mcp-Session-Id", "newsess")
		body := "event: message\ndata: " + `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}` + "\n\n"
		return jsonResponse(http.StatusOK, body, h), nil
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "", true)
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools sse body: %v", err)
	}
	if len(tools) != 0 {
		t.Fatalf("tools = %v", tools)
	}
}

// --- StreamableHTTP notifyLocked non-2xx ---

func TestStreamableHTTPNotifyLockedNon2xx(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusInternalServerError, "", nil), nil
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	if err := client.notifyLocked(context.Background(), "notifications/initialized", map[string]any{}); err == nil {
		t.Fatal("want notify non-2xx error")
	}
}

func TestStreamableHTTPNotifyLockedNetworkError(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	if err := client.notifyLocked(context.Background(), "notifications/initialized", map[string]any{}); err == nil {
		t.Fatal("want notify network error")
	}
}

// --- StreamableHTTP ensureInitialized notify error path ---

func TestStreamableHTTPEnsureInitializedNotifyError(t *testing.T) {
	var stage atomic.Int32
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if strings.Contains(string(body), "notifications/initialized") {
			return nil, errors.New("notify failed")
		}
		_ = stage.Add(1)
		return jsonResponse(http.StatusOK, `{"jsonrpc":"2.0","id":1,"result":{}}`, nil), nil
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "", false)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("want notify error during initialize")
	}
}

// --- StreamableHTTP Close: terminate session happy + error ---

func TestStreamableHTTPCloseTerminatesSession(t *testing.T) {
	var deleteHit atomic.Bool
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			deleteHit.Store(true)
			if req.Header.Get("Mcp-Session-Id") != "sess" {
				t.Errorf("missing session header")
			}
			return jsonResponse(http.StatusOK, "", nil), nil
		}
		return jsonResponse(http.StatusOK, "", nil), nil
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !deleteHit.Load() {
		t.Fatal("DELETE session terminate not sent")
	}
	// Second close is a no-op.
	if err := client.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestStreamableHTTPCloseTerminateError(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("delete failed")
	})
	client := NewStreamableHTTPClient(doer, "http://x", nil, "sess", true)
	if err := client.Close(); err == nil {
		t.Fatal("want terminate error")
	}
}

func TestStreamableHTTPCloseNoSession(t *testing.T) {
	client := NewStreamableHTTPClient(doerFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("no request expected without session")
		return nil, nil
	}), "http://x", nil, "", true)
	if err := client.Close(); err != nil {
		t.Fatalf("Close no session: %v", err)
	}
}

// --- StreamableHTTP CallTool happy path through full init ---

func TestStreamableHTTPCallToolHappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		if _, hasID := req["id"]; !hasID {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "s")
		switch method {
		case "initialize":
			writeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/call":
			writeRPCResult(t, w, req["id"], map[string]any{"content": "ok"})
		default:
			writeRPCError(t, w, req["id"], -32601, "unknown")
		}
	}))
	defer server.Close()
	client := NewStreamableHTTPClient(server.Client(), server.URL, map[string]string{"X-Test": "1"}, "", false)
	res, err := client.CallTool(context.Background(), CallRequest{Name: "x", Arguments: json.RawMessage(`{"a":1}`)})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.Content != "ok" {
		t.Fatalf("content = %v", res.Content)
	}
}

// --- SSE: reconnect / endpoint resolution / errors ---

// sseFakeServer models a real MCP SSE server: the GET /sse stream stays open and
// is the channel down which JSON-RPC responses are pushed. POST /messages returns
// 202 and the matching response is streamed back on the open /sse connection.
type sseFakeServer struct {
	server      *httptest.Server
	responses   chan string
	handler     func(method string, id any) map[string]any
	endpointURL func(base string) string // resolves the advertised endpoint path/url
}

func newSSEFakeServer(t *testing.T, handler func(method string, id any) map[string]any) *sseFakeServer {
	t.Helper()
	f := &sseFakeServer{responses: make(chan string, 8), handler: handler}
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		endpoint := "/messages"
		if f.endpointURL != nil {
			endpoint = f.endpointURL(f.server.URL)
		}
		_, _ = io.WriteString(w, "event: endpoint\ndata: "+endpoint+"\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		for {
			select {
			case <-r.Context().Done():
				return
			case msg := <-f.responses:
				_, _ = io.WriteString(w, "event: message\ndata: "+msg+"\n\n")
				if flusher != nil {
					flusher.Flush()
				}
			}
		}
	})
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		if _, hasID := req["id"]; !hasID {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		resp := f.handler(method, req["id"])
		enc, _ := json.Marshal(resp)
		f.responses <- string(enc)
		w.WriteHeader(http.StatusAccepted)
	})
	f.server = httptest.NewServer(mux)
	return f
}

func defaultSSEHandler(method string, id any) map[string]any {
	switch method {
	case "initialize":
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"protocolVersion": protocolVersion}}
	case "tools/list":
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{{"name": "t"}}}}
	default:
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}}
	}
}

func TestSSEClientListToolsHappyPath(t *testing.T) {
	f := newSSEFakeServer(t, defaultSSEHandler)
	defer f.server.Close()
	client := NewSSEClient(f.server.Client(), f.server.URL, map[string]string{"X-H": "1"})
	defer client.Close()
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "t" {
		t.Fatalf("tools = %v", tools)
	}
	// CallTool over the same already-initialized connection.
	if _, err := client.CallTool(context.Background(), CallRequest{Name: "t"}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
}

func TestSSEClientEnsureEndpointBadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	client := NewSSEClient(server.Client(), server.URL, nil)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("want bad status error from sse connect")
	}
}

func TestSSEClientEnsureEndpointNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	c := server.Client()
	server.Close()
	client := NewSSEClient(c, url, nil)
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("want network error from sse connect")
	}
}

func TestSSEClientPostNon2xx(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: endpoint\ndata: /messages\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		<-r.Context().Done()
	})
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewSSEClient(server.Client(), server.URL, nil)
	defer client.Close()
	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatal("want post non-2xx error")
	}
}

func TestSSEClientResolvesAbsoluteEndpoint(t *testing.T) {
	// Advertise the message endpoint as an absolute URL (same server) so
	// resolveEndpoint takes its IsAbs() branch.
	f := newSSEFakeServer(t, defaultSSEHandler)
	f.endpointURL = func(base string) string { return base + "/messages" }
	defer f.server.Close()

	client := NewSSEClient(f.server.Client(), f.server.URL, nil)
	defer client.Close()
	if _, err := client.ListTools(context.Background()); err != nil {
		t.Fatalf("ListTools absolute endpoint: %v", err)
	}
}

func TestSSEClientCloseNilBody(t *testing.T) {
	client := NewSSEClient(http.DefaultClient, "http://x", nil)
	if err := client.Close(); err != nil {
		t.Fatalf("Close nil body: %v", err)
	}
}

// resolveEndpoint direct branches.
func TestResolveEndpointBranches(t *testing.T) {
	if got := resolveEndpoint("http://base.example/x", "https://abs.example/y"); got != "https://abs.example/y" {
		t.Fatalf("absolute = %q", got)
	}
	if got := resolveEndpoint("http://base.example", "/messages"); got != "http://base.example/messages" {
		t.Fatalf("relative = %q", got)
	}
	// Unparseable endpoint returns endpoint as-is via final fallback paths.
	if got := resolveEndpoint("http://base.example", "://bad"); got == "" {
		t.Fatalf("bad endpoint = %q", got)
	}
}

// --- HTTPTransport InitializeSSE + InitializeStreamable ---

func TestHTTPTransportInitializeSSE(t *testing.T) {
	f := newSSEFakeServer(t, defaultSSEHandler)
	defer f.server.Close()
	tr := NewHTTPTransport(f.server.Client())
	endpoint, err := tr.InitializeSSE(context.Background(), f.server.URL, nil)
	if err != nil {
		t.Fatalf("InitializeSSE: %v", err)
	}
	if !strings.HasSuffix(endpoint, "/messages") {
		t.Fatalf("endpoint = %q", endpoint)
	}
}

func TestHTTPTransportInitializeStreamableNotifyError(t *testing.T) {
	var initDone atomic.Bool
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if strings.Contains(string(body), "notifications/initialized") {
			return nil, errors.New("notify failed")
		}
		initDone.Store(true)
		h := http.Header{}
		h.Set("Mcp-Session-Id", "sess")
		return jsonResponse(http.StatusOK, "", h), nil
	})
	tr := NewHTTPTransport(doer)
	if _, _, err := tr.InitializeStreamable(context.Background(), "http://x", nil); err == nil {
		t.Fatal("want notify error from InitializeStreamable")
	}
}

func TestHTTPTransportInitializeStreamableBadStatus(t *testing.T) {
	doer := doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusForbidden, "", nil), nil
	})
	tr := NewHTTPTransport(doer)
	if _, status, err := tr.InitializeStreamable(context.Background(), "http://x", nil); err == nil || status != http.StatusForbidden {
		t.Fatalf("want bad status error, got status=%d err=%v", status, err)
	}
}

// --- stdio notifyLocked + writeLineLocked error paths ---

type errWriteCloser struct{}

func (errWriteCloser) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func (errWriteCloser) Close() error              { return nil }

func TestStdioWriteLineLockedError(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
	})
	defer server.Close()
	client := NewStdioClient(process)
	// Replace writer with one backed by a failing writer to exercise writeLineLocked error.
	client.writer.Reset(errWriteCloser{})
	if err := client.notifyLocked("notifications/test", map[string]any{}); err == nil {
		t.Fatal("want writeLineLocked error from notifyLocked")
	}
}

func TestStdioStartReadLoopIdempotent(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
	})
	defer server.Close()
	client := NewStdioClient(process)
	client.startReadLoop()
	client.startReadLoop() // second call returns early (already started).
}

// --- healthmonitor checkClient refresh error (tool already registered conflict) ---

type dupToolClient struct{}

func (dupToolClient) ListTools(context.Context) ([]Tool, error) {
	return []Tool{{Name: "dup"}, {Name: "dup"}}, nil
}
func (dupToolClient) CallTool(context.Context, CallRequest) (CallResult, error) {
	return CallResult{}, nil
}
func (dupToolClient) Close() error { return nil }

func TestHealthMonitorCheckRefreshManifestError(t *testing.T) {
	manager := &ClientManager{clients: map[string]Client{"docs": dupToolClient{}}}
	tools := NewToolManager()
	monitor := NewHealthMonitor(manager, tools)
	status, err := monitor.Check(context.Background(), "docs")
	if err == nil {
		t.Fatal("want refresh manifest error")
	}
	if status.Healthy {
		t.Fatal("status should be unhealthy on refresh error")
	}
}

// --- toolmanager refcount: UnregisterClient waits for in-flight call ---

type blockingCallClient struct {
	release chan struct{}
	entered chan struct{}
}

func (c *blockingCallClient) ListTools(context.Context) ([]Tool, error) { return nil, nil }
func (c *blockingCallClient) CallTool(context.Context, CallRequest) (CallResult, error) {
	close(c.entered)
	<-c.release
	return CallResult{Content: "done"}, nil
}
func (c *blockingCallClient) Close() error { return nil }

func TestToolManagerUnregisterWaitsForInflight(t *testing.T) {
	tm := NewToolManager()
	client := &blockingCallClient{release: make(chan struct{}), entered: make(chan struct{})}
	tm.RegisterClient("docs", client)
	if err := tm.RegisterManifest(Manifest{ClientID: "docs", Tools: []Tool{{Name: "search"}}}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	callDone := make(chan error, 1)
	go func() {
		_, err := tm.Call(context.Background(), "docs__search", json.RawMessage(`{}`))
		callDone <- err
	}()
	<-client.entered // call is in-flight, holding a refcount

	unregisterReturned := make(chan struct{})
	go func() {
		tm.UnregisterClient("docs")
		close(unregisterReturned)
	}()

	// UnregisterClient must block until the in-flight call releases.
	select {
	case <-unregisterReturned:
		t.Fatal("UnregisterClient returned before in-flight call finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(client.release)
	if err := <-callDone; err != nil {
		t.Fatalf("Call: %v", err)
	}
	select {
	case <-unregisterReturned:
	case <-time.After(time.Second):
		t.Fatal("UnregisterClient did not return after call finished")
	}
}

func TestToolManagerReleaseClientMultipleRefs(t *testing.T) {
	tm := NewToolManager()
	tm.RegisterClient("docs", &fakeClient{callResult: CallResult{Content: "x"}})
	// Acquire twice, release twice -> exercises n>1 branch.
	if _, ok := tm.acquireClient("docs"); !ok {
		t.Fatal("acquire 1")
	}
	if _, ok := tm.acquireClient("docs"); !ok {
		t.Fatal("acquire 2")
	}
	tm.releaseClient("docs") // n>1 branch
	tm.releaseClient("docs") // n<=1 branch (delete)
}

// --- oauth postToken: 3xx redirect-status -> unavailable, decode error ---

func TestPostTokenRedirectStatusUnavailable(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusFound, "", nil), nil
	}))
	if _, err := engine.postToken(context.Background(), "http://x/token", nil); !errors.Is(err, errOAuthTokenEndpointUnavailable) {
		t.Fatalf("err = %v, want unavailable", err)
	}
}

func TestPostTokenDecodeError(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, "notjson", nil), nil
	}))
	if _, err := engine.postToken(context.Background(), "http://x/token", nil); err == nil {
		t.Fatal("want decode error")
	}
}

func TestPostTokenNon2xxNon3xx(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, "err", nil), nil
	}))
	if _, err := engine.postToken(context.Background(), "http://x/token", nil); err == nil {
		t.Fatal("want non-2xx error")
	}
}

func TestPostTokenRequestBuildError(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, "{}", nil), nil
	}))
	// Control character in URL forces NewRequestWithContext to fail.
	if _, err := engine.postToken(context.Background(), "http://\x7f/token", nil); err == nil {
		t.Fatal("want request build error")
	}
}

// --- oauth RefreshAccount guards ---

func TestRefreshAccountNoRefreshToken(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), http.DefaultClient)
	if _, err := engine.RefreshAccount(context.Background(), OAuthAccount{}); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want reauth", err)
	}
}

func TestRefreshAccountNoTokenEndpoint(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), http.DefaultClient)
	if _, err := engine.RefreshAccount(context.Background(), OAuthAccount{RefreshToken: "r"}); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want reauth (no endpoint)", err)
	}
}

func TestRefreshAccountMissingAccessTokenInResponse(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"expires_in":3600}`, nil), nil
	}))
	_, err := engine.RefreshAccount(context.Background(), OAuthAccount{
		RefreshToken: "r",
		AuthMetadata: map[string]string{"token_endpoint": "http://x/token"},
	})
	if err == nil || !strings.Contains(err.Error(), "access token is required") {
		t.Fatalf("err = %v, want access token required", err)
	}
}

func TestRefreshAccountSaveError(t *testing.T) {
	store := &saveErrStore{fakeOAuthStore: newFakeOAuthStore()}
	engine := NewOAuthEngine(store, doerFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"access_token":"a","refresh_token":"r2","scope":"s","expires_in":10}`, nil), nil
	}))
	_, err := engine.RefreshAccount(context.Background(), OAuthAccount{
		RefreshToken: "r",
		AuthMetadata: map[string]string{"token_endpoint": "http://x/token"},
	})
	if err == nil {
		t.Fatal("want save error")
	}
}

type saveErrStore struct {
	*fakeOAuthStore
}

func (s *saveErrStore) SaveAccount(OAuthAccount) error { return errors.New("save failed") }

// --- oauth CompleteCallback guards ---

func TestCompleteCallbackMissingCodeOrState(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), http.DefaultClient)
	if _, err := engine.CompleteCallback(context.Background(), "inst", "https://cb?state=s"); !errors.Is(err, ErrOAuthFlowNotFound) {
		t.Fatalf("missing code err = %v", err)
	}
}

func TestCompleteCallbackBadCallbackURL(t *testing.T) {
	engine := NewOAuthEngine(newFakeOAuthStore(), http.DefaultClient)
	if _, err := engine.CompleteCallback(context.Background(), "inst", "://bad"); err == nil {
		t.Fatal("want parse callback url error")
	}
}

func TestCompleteCallbackExpiredFlow(t *testing.T) {
	store := newFakeOAuthStore()
	_ = store.CreateFlow(OAuthFlow{InstanceID: "inst", State: "s", ExpiresAt: time.Now().Add(-time.Minute)})
	engine := NewOAuthEngine(store, http.DefaultClient)
	if _, err := engine.CompleteCallback(context.Background(), "inst", "https://cb?code=c&state=s"); !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("err = %v, want reauth for expired flow", err)
	}
}

// --- BuildOAuthStartFlow validation branches ---

func TestBuildOAuthStartFlowValidation(t *testing.T) {
	base := OAuthStartConfig{
		InstanceID:       "inst",
		AuthorizationURL: "https://a/authorize",
		RedirectURI:      "http://localhost/cb",
		ResourceURI:      "https://r",
	}
	cases := []OAuthStartConfig{
		{AuthorizationURL: base.AuthorizationURL, RedirectURI: base.RedirectURI, ResourceURI: base.ResourceURI},
		{InstanceID: base.InstanceID, RedirectURI: base.RedirectURI, ResourceURI: base.ResourceURI},
		{InstanceID: base.InstanceID, AuthorizationURL: base.AuthorizationURL, ResourceURI: base.ResourceURI},
		{InstanceID: base.InstanceID, AuthorizationURL: base.AuthorizationURL, RedirectURI: base.RedirectURI},
	}
	for i, c := range cases {
		if _, err := BuildOAuthStartFlow(c); err == nil {
			t.Fatalf("case %d: want validation error", i)
		}
	}
	// Bad authorization URL parse.
	bad := base
	bad.AuthorizationURL = "://bad"
	if _, err := BuildOAuthStartFlow(bad); err == nil {
		t.Fatal("want parse authorization url error")
	}
	// Default expiration branch (no ExpirationSeconds).
	flow, err := BuildOAuthStartFlow(base)
	if err != nil {
		t.Fatalf("BuildOAuthStartFlow default exp: %v", err)
	}
	if flow.ExpiresAt.IsZero() {
		t.Fatal("expires at should be set")
	}
}

// --- DiscoverOAuthAuthorizationURL guard branches ---

func TestDiscoverOAuthAuthorizationURLGuards(t *testing.T) {
	// Non-http scheme.
	if _, err := DiscoverOAuthAuthorizationURL(context.Background(), http.DefaultClient, "ftp://x"); !errors.Is(err, errOAuthTokenEndpointUnavailable) {
		t.Fatalf("scheme err = %v", err)
	}
	// Empty authorization_servers list.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp":
			w.WriteHeader(http.StatusUnauthorized)
		case "/.well-known/oauth-protected-resource":
			_, _ = io.WriteString(w, `{"authorization_servers":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	if _, err := DiscoverOAuthAuthorizationURL(context.Background(), server.Client(), server.URL+"/mcp"); !errors.Is(err, errOAuthTokenEndpointUnavailable) {
		t.Fatalf("empty servers err = %v", err)
	}
}

// keep sync import used
var _ = sync.Mutex{}
