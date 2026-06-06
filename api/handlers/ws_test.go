package handlers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

type wsFakeEngine struct {
	stream    <-chan providers.StreamChunk
	streamErr error
}

func (f *wsFakeEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, nil
}
func (f *wsFakeEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	return f.stream, nil
}
func (f *wsFakeEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return nil, nil
}

type fakeWSFeatureFlagStore struct {
	flag *store.FeatureFlag
	err  error
}

func (f *fakeWSFeatureFlagStore) GetFeatureFlagByKey(key string) (*store.FeatureFlag, error) {
	return f.flag, f.err
}

func TestWSChatEngineNil(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		WSChat(ctx, nil, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestWSChatFlagDisabled(t *testing.T) {
	fs := &fakeWSFeatureFlagStore{flag: &store.FeatureFlag{Key: "websocket_chat", Enabled: false}}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		WSChat(ctx, &wsFakeEngine{}, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestWSChatFlagStoreError(t *testing.T) {
	fs := &fakeWSFeatureFlagStore{err: store.ErrNotFound}
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		WSChat(ctx, &wsFakeEngine{}, fs)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestWSChatFlagNilStore(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		WSChat(ctx, &wsFakeEngine{}, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestWSChatMethodNotAllowed(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		WSChat(ctx, &wsFakeEngine{}, nil)
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", ctx.Response.StatusCode())
	}
}

func wsTestServer(t *testing.T) (clientConn, serverConn *websocket.Conn, cleanup func()) {
	t.Helper()

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	var sc *websocket.Conn
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		sc, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		mu.Unlock()
	}))
	t.Cleanup(server.Close)

	mu.Lock()
	clientConn, _, err := websocket.DefaultDialer.Dial("ws://"+strings.TrimPrefix(server.URL, "http://")+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	mu.Lock()
	if sc == nil {
		t.Fatal("server connection not set")
	}

	return clientConn, sc, func() {
		_ = clientConn.Close()
		_ = sc.Close()
	}
}

func TestWSHandleConnectionInvalidJSON(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{})
	}()

	_ = client.WriteJSON(map[string]any{"type": 123}) // invalid: type is number

	var msg map[string]any
	if err := client.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["type"] != "error" || msg["error"] != "invalid message" {
		t.Fatalf("msg = %+v", msg)
	}
	<-done
}

func TestWSHandleConnectionWrongType(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{})
	}()

	_ = client.WriteJSON(map[string]any{"type": "ping"})

	var msg map[string]any
	if err := client.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["type"] != "error" || msg["error"] != "expected type chat" {
		t.Fatalf("msg = %+v", msg)
	}
	<-done
}

func TestWSHandleConnectionMissingModel(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{})
	}()

	_ = client.WriteJSON(map[string]any{"type": "chat", "messages": []any{}})

	var msg map[string]any
	if err := client.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["type"] != "error" || msg["error"] != "model required" {
		t.Fatalf("msg = %+v", msg)
	}
	<-done
}

func TestWSHandleConnectionDispatchError(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{streamErr: providers.ErrStreamingUnsupported})
	}()

	_ = client.WriteJSON(map[string]any{"type": "chat", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}})

	var msg map[string]any
	if err := client.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["type"] != "error" || msg["error"] != "dispatch error" {
		t.Fatalf("msg = %+v", msg)
	}
	<-done
}

func TestWSHandleConnectionStreamError(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		Error: &providers.StreamError{Message: "upstream failed", Type: "server_error", Code: "upstream_error"},
	}
	close(chunks)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{stream: chunks})
	}()

	_ = client.WriteJSON(map[string]any{"type": "chat", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}})

	var msg map[string]any
	if err := client.ReadJSON(&msg); err != nil {
		t.Fatalf("read: %v", err)
	}
	if msg["type"] != "error" || msg["error"] != "upstream failed" {
		t.Fatalf("msg = %+v", msg)
	}
	<-done
}

func TestWSHandleConnectionRoundTrip(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 2)
	chunks <- providers.StreamChunk{
		ID:      "ws-test",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "ws-test",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
		Usage:   &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
	close(chunks)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{stream: chunks})
	}()

	_ = client.WriteJSON(map[string]any{"type": "chat", "session_id": "s1", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}})

	var delta map[string]any
	if err := client.ReadJSON(&delta); err != nil {
		t.Fatalf("read delta: %v", err)
	}
	if delta["type"] != "delta" || delta["content"] != "hello" {
		t.Fatalf("delta = %+v", delta)
	}

	var doneMsg map[string]any
	if err := client.ReadJSON(&doneMsg); err != nil {
		t.Fatalf("read done: %v", err)
	}
	if doneMsg["type"] != "done" {
		t.Fatalf("done = %+v", doneMsg)
	}
	usage, ok := doneMsg["usage"].(map[string]any)
	if !ok || usage["total_tokens"] != float64(5) {
		t.Fatalf("done usage = %+v", doneMsg)
	}

	// Wait for handler to finish before returning so fasthttp doesn't
	 // reclaim the connection while the goroutine is still running.
	 select {
	 case <-done:
	 case <-time.After(2 * time.Second):
	 	t.Fatal("timeout waiting for wsHandleConnection")
	 }
}

func TestWSHandleConnectionClientCloseCancelsStream(t *testing.T) {
	client, server, cleanup := wsTestServer(t)
	defer cleanup()

	// Block the stream so the handler stays alive.
	chunks := make(chan providers.StreamChunk)

	done := make(chan struct{})
	go func() {
		defer close(done)
		wsHandleConnection(server, &wsFakeEngine{stream: chunks})
	}()

	_ = client.WriteJSON(map[string]any{"type": "chat", "session_id": "s1", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}})

	// Give handler time to start reading from stream.
	time.Sleep(50 * time.Millisecond)

	// Close client abruptly.
	_ = client.Close()

	// Handler should cancel and exit without panic.
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for wsHandleConnection after close")
	}
}

func TestWSChatUpgradeSuccess(t *testing.T) {
	ln := fasthttputil.NewInmemoryListener()

	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 2)
	chunks <- providers.StreamChunk{
		ID:      "ws-mem",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "ws-mem",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
		Usage:   &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
	close(chunks)

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			WSChat(ctx, &wsFakeEngine{stream: chunks}, nil)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Shutdown()

	dialer := websocket.Dialer{
		HandshakeTimeout: 2 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}

	conn, resp, err := dialer.Dial("ws://localhost/api/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v; status=%d", err, resp.StatusCode)
	}
	defer conn.Close()

	_ = conn.WriteJSON(map[string]any{"type": "chat", "session_id": "s1", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}})

	var delta map[string]any
	if err := conn.ReadJSON(&delta); err != nil {
		t.Fatalf("read delta: %v", err)
	}
	if delta["type"] != "delta" || delta["content"] != "hello" {
		t.Fatalf("delta = %+v", delta)
	}

	var doneMsg map[string]any
	if err := conn.ReadJSON(&doneMsg); err != nil {
		t.Fatalf("read done: %v", err)
	}
	if doneMsg["type"] != "done" {
		t.Fatalf("done = %+v", doneMsg)
	}
}
