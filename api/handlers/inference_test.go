package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/proxy"
)

type fakeEngine struct {
	response       *providers.ChatResponse
	stream         <-chan providers.StreamChunk
	models         []providers.Model
	err            error
	streamErr      error
	received       *providers.ChatRequest
	streamReceived *providers.ChatRequest
}

func (f *fakeEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.received = req
	return f.response, f.err
}

func (f *fakeEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	f.streamReceived = req
	return f.stream, f.streamErr
}

func (f *fakeEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return f.models, f.err
}

type fakeValidator struct {
	valid bool
}

func (f fakeValidator) ValidateAPIKey(key, secret string) (bool, error) {
	return f.valid && key == "g0r_valid" && secret == "test-secret", nil
}

func TestSyncInference(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":false}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var decoded providers.ChatResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if decoded.ID != "chatcmpl-test" {
		t.Fatalf("ID = %q, want chatcmpl-test", decoded.ID)
	}
	if decoded.Choices[0].Message.Content != "hello back" {
		t.Fatalf("content = %#v, want hello back", decoded.Choices[0].Message.Content)
	}
	if engine.received == nil || engine.received.Model != "gpt-4o" {
		t.Fatalf("engine received = %+v", engine.received)
	}
}

func TestStreamInference(t *testing.T) {
	role := "assistant"
	content := "hello"
	finish := "stop"
	chunks := make(chan providers.StreamChunk, 3)
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Role: &role}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, Delta: providers.StreamDelta{Content: &content}}},
	}
	chunks <- providers.StreamChunk{
		ID:      "chatcmpl-stream",
		Object:  "chat.completion.chunk",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.StreamChoice{{Index: 0, FinishReason: &finish}},
	}
	close(chunks)

	engine := &fakeEngine{stream: chunks}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}],"stream":true}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	bodyText := string(body)
	if strings.Count(bodyText, "data: ") != 4 {
		t.Fatalf("SSE data count = %d, want 4; body=%s", strings.Count(bodyText, "data: "), bodyText)
	}
	if !strings.Contains(bodyText, `"content":"hello"`) {
		t.Fatalf("stream body missing content chunk: %s", bodyText)
	}
	if !strings.HasSuffix(bodyText, "data: [DONE]\n\n") {
		t.Fatalf("stream body should end with [DONE], got %q", bodyText)
	}
	if engine.streamReceived == nil || engine.streamReceived.Stream == nil || !*engine.streamReceived.Stream {
		t.Fatalf("stream request = %+v", engine.streamReceived)
	}
}

func TestInferenceInvalidJSON(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
}

func TestInferenceUnknownModel(t *testing.T) {
	engine := &fakeEngine{err: proxy.ErrProviderNotFound}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"nonexistent","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
	}
}

func TestInferenceNoAuth(t *testing.T) {
	engine := &fakeEngine{response: chatResponse()}
	_, baseURL := startInferenceServer(t, api.ServerConfig{
		Version:         "test",
		RequireAPIKey:   true,
		APIKeySecret:    "test-secret",
		APIKeyValidator: fakeValidator{valid: true},
		InferenceEngine: engine,
	})

	resp, body := postJSON(t, baseURL+"/v1/chat/completions", `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", resp.StatusCode, body)
	}
	if engine.received != nil {
		t.Fatal("engine should not be called when auth fails")
	}
}

func TestGetModels(t *testing.T) {
	engine := &fakeEngine{models: []providers.Model{
		{ID: "gpt-4o", Object: "model", Created: 1710000000, OwnedBy: "openai", Provider: providers.ProviderOpenAI},
		{ID: "claude-3-5-sonnet", Object: "model", Created: 1710000001, OwnedBy: "anthropic", Provider: providers.ProviderAnthropic},
	}}
	_, baseURL := startInferenceServer(t, api.ServerConfig{Version: "test", InferenceEngine: engine})

	resp, err := httpClient().Get(baseURL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var decoded struct {
		Object string            `json:"object"`
		Data   []providers.Model `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal models: %v", err)
	}
	if decoded.Object != "list" {
		t.Fatalf("object = %q, want list", decoded.Object)
	}
	if len(decoded.Data) != 2 {
		t.Fatalf("models len = %d, want 2", len(decoded.Data))
	}
	if decoded.Data[0].ID != "gpt-4o" || decoded.Data[1].ID != "claude-3-5-sonnet" {
		t.Fatalf("models = %+v", decoded.Data)
	}
}

func chatResponse() *providers.ChatResponse {
	finish := "stop"
	return &providers.ChatResponse{
		ID:      "chatcmpl-test",
		Object:  "chat.completion",
		Created: 1710000000,
		Model:   "gpt-4o",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "hello back",
			},
			FinishReason: &finish,
		}},
		Usage: &providers.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5},
	}
}

func postJSON(t *testing.T, url string, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		t.Fatalf("read body: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, data
}

func startInferenceServer(t *testing.T, config api.ServerConfig) (*api.Server, string) {
	t.Helper()

	srv := api.NewServer(config)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("Serve: %v", err)
		}
	}()
	t.Cleanup(func() { _ = srv.Stop() })

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr is %T, want *net.TCPAddr", ln.Addr())
	}
	return srv, "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(tcpAddr.Port))
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}
