package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/fasthttp/websocket"
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

func TestIntegrationWSChatUnauthorized(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Store:           s,
		RequireAPIKey:   true,
		InferenceEngine: &wsFakeEngine{},
	})

	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, resp, err := dialer.Dial("ws://"+strings.TrimPrefix(baseURL, "http://")+"/api/ws", nil)
	if err == nil {
		t.Fatal("expected dial error without auth")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestIntegrationWSChatFlagDisabled(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Store:           s,
		InferenceEngine: &wsFakeEngine{},
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+testHarnessAPIKey)
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	_, resp, err := dialer.Dial("ws://"+strings.TrimPrefix(baseURL, "http://")+"/api/ws", headers)
	if err == nil {
		t.Fatal("expected dial error when flag disabled")
	}
	if resp != nil && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestIntegrationWSChatRoundTrip(t *testing.T) {
	s := newAPITestStore(t)
	flag, err := s.GetFeatureFlagByKey("websocket_chat")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}
	if err := s.ToggleFeatureFlag(flag.ID, true); err != nil {
		t.Fatalf("enable websocket_chat flag: %v", err)
	}

	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 3)
	chunks <- providers.StreamChunk{
		ID:      "ws-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "ws-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
		Usage:   &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
	close(chunks)

	engine := &wsFakeEngine{stream: chunks}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Store:           s,
		InferenceEngine: engine,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+testHarnessAPIKey)
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, resp, err := dialer.Dial("ws://"+strings.TrimPrefix(baseURL, "http://")+"/api/ws", headers)
	if err != nil {
		t.Fatalf("dial: %v; status=%d", err, resp.StatusCode)
	}
	defer conn.Close()

	msg := map[string]any{"type": "chat", "session_id": "s1", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("write json: %v", err)
	}

	var delta map[string]any
	if err := conn.ReadJSON(&delta); err != nil {
		t.Fatalf("read delta: %v", err)
	}
	if delta["type"] != "delta" || delta["content"] != "hello" {
		t.Fatalf("delta = %+v", delta)
	}

	var done map[string]any
	if err := conn.ReadJSON(&done); err != nil {
		t.Fatalf("read done: %v", err)
	}
	if done["type"] != "done" {
		t.Fatalf("done = %+v", done)
	}
	usage, ok := done["usage"].(map[string]any)
	if !ok || usage["total_tokens"] != float64(5) {
		t.Fatalf("done usage = %+v", done)
	}
}

func TestIntegrationWSChatDispatchError(t *testing.T) {
	s := newAPITestStore(t)
	flag, err := s.GetFeatureFlagByKey("websocket_chat")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}
	if err := s.ToggleFeatureFlag(flag.ID, true); err != nil {
		t.Fatalf("enable websocket_chat flag: %v", err)
	}

	engine := &wsFakeEngine{streamErr: providers.ErrStreamingUnsupported}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Store:           s,
		InferenceEngine: engine,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+testHarnessAPIKey)
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, resp, err := dialer.Dial("ws://"+strings.TrimPrefix(baseURL, "http://")+"/api/ws", headers)
	if err != nil {
		t.Fatalf("dial: %v; status=%d", err, resp.StatusCode)
	}
	defer conn.Close()

	msg := map[string]any{"type": "chat", "session_id": "s1", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("write json: %v", err)
	}

	var errMsg map[string]any
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("read error: %v", err)
	}
	if errMsg["type"] != "error" {
		t.Fatalf("expected error type, got %+v", errMsg)
	}
}

func TestIntegrationWSChatStreamError(t *testing.T) {
	s := newAPITestStore(t)
	flag, err := s.GetFeatureFlagByKey("websocket_chat")
	if err != nil {
		t.Fatalf("GetFeatureFlagByKey: %v", err)
	}
	if err := s.ToggleFeatureFlag(flag.ID, true); err != nil {
		t.Fatalf("enable websocket_chat flag: %v", err)
	}

	chunks := make(chan providers.StreamChunk, 1)
	chunks <- providers.StreamChunk{
		Error: &providers.StreamError{Message: "upstream failed", Type: "server_error", Code: "upstream_error"},
	}
	close(chunks)

	engine := &wsFakeEngine{stream: chunks}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Store:           s,
		InferenceEngine: engine,
	})

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+testHarnessAPIKey)
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	conn, resp, err := dialer.Dial("ws://"+strings.TrimPrefix(baseURL, "http://")+"/api/ws", headers)
	if err != nil {
		t.Fatalf("dial: %v; status=%d", err, resp.StatusCode)
	}
	defer conn.Close()

	msg := map[string]any{"type": "chat", "session_id": "s1", "model": "gpt-4o", "messages": []map[string]any{{"role": "user", "content": "hi"}}}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("write json: %v", err)
	}

	var errMsg map[string]any
	if err := conn.ReadJSON(&errMsg); err != nil {
		t.Fatalf("read error: %v", err)
	}
	if errMsg["type"] != "error" || errMsg["error"] != "upstream failed" {
		t.Fatalf("expected upstream error, got %+v", errMsg)
	}
}
