package handlers_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

type fakeModelTestProvider struct {
	response *providers.ChatResponse
	err      error
	delay    time.Duration
}

func (f *fakeModelTestProvider) Name() providers.ModelProvider {
	return "openai"
}

func (f *fakeModelTestProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	return f.response, f.err
}

func (f *fakeModelTestProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f *fakeModelTestProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

type fakeModelTestAdapterSource struct {
	provider providers.Provider
	ok       bool
}

func (f fakeModelTestAdapterSource) GetProvider(name providers.ModelProvider) (providers.Provider, bool) {
	return f.provider, f.ok
}

func newModelTestStore(t *testing.T) *store.Store {
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

func startModelTestServer(t *testing.T, config api.ServerConfig) (*api.Server, string) {
	t.Helper()
	if config.APIKeyValidator == nil {
		config.APIKeyValidator = fakeValidator{valid: true}
		config.APIKeySecret = "test-secret"
	}
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

func postModelTestJSON(t *testing.T, url, body string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer g0r_valid")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		t.Fatalf("read body: %v", err)
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, data
}

func TestModelTestActiveConnection(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{response: &providers.ChatResponse{ID: "test"}, delay: 5 * time.Millisecond}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/openai/models/gpt-4o/test", "")
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var decoded struct {
		Data struct {
			OK        bool    `json:"ok"`
			LatencyMS int64   `json:"latency_ms"`
			Error     *string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, body)
	}
	if !decoded.Data.OK {
		t.Fatalf("ok = false, want true; error=%v", decoded.Data.Error)
	}
	if decoded.Data.LatencyMS <= 0 {
		t.Fatalf("latency_ms = %d, want > 0", decoded.Data.LatencyMS)
	}
	if decoded.Data.Error != nil {
		t.Fatalf("error = %v, want nil", decoded.Data.Error)
	}
}

func TestModelTestCustomMessages(t *testing.T) {
	s := newModelTestStore(t)
	var received *providers.ChatRequest
	provider := &fakeModelTestProvider{response: &providers.ChatResponse{ID: "test"}}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version: "test",
		Store:   s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: &fakeModelTestProviderWithCapture{
			provider: provider,
			capture:  &received,
		}, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/openai/models/gpt-4o/test", `{"messages":[{"role":"system","content":"be brief"}]}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if received == nil || len(received.Messages) != 1 || received.Messages[0].Role != "system" {
		t.Fatalf("custom messages not passed: %+v", received)
	}
}

type fakeModelTestProviderWithCapture struct {
	provider providers.Provider
	capture  **providers.ChatRequest
}

func (f *fakeModelTestProviderWithCapture) Name() providers.ModelProvider {
	return f.provider.Name()
}

func (f *fakeModelTestProviderWithCapture) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	*f.capture = req
	return f.provider.ChatCompletion(ctx, key, req)
}

func (f *fakeModelTestProviderWithCapture) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f *fakeModelTestProviderWithCapture) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

func TestModelTestNoConnections(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/openai/models/gpt-4o/test", "")
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "no active connections") {
		t.Fatalf("body = %s, want no active connections", body)
	}
}

func TestModelTestUpstreamFailure(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{err: errors.New("upstream error")}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/openai/models/gpt-4o/test", "")
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var decoded struct {
		Data struct {
			OK        bool    `json:"ok"`
			LatencyMS int64   `json:"latency_ms"`
			Error     *string `json:"error"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal: %v; body=%s", err, body)
	}
	if decoded.Data.OK {
		t.Fatalf("ok = true, want false")
	}
	if decoded.Data.LatencyMS != 0 {
		t.Fatalf("latency_ms = %d, want 0", decoded.Data.LatencyMS)
	}
	if decoded.Data.Error == nil || !strings.Contains(*decoded.Data.Error, "upstream error") {
		t.Fatalf("error = %v, want upstream error", decoded.Data.Error)
	}
}

func TestModelTestBatchSSE(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{response: &providers.ChatResponse{ID: "test"}}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn1",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn2",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/test-batch", `{"providers":["openai"]}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	scanner := bufio.NewScanner(bytes.NewReader(body))
	var results int
	var done bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: result") {
			results++
		}
		if strings.HasPrefix(line, "event: done") {
			done = true
		}
	}
	if results != 2 {
		t.Fatalf("result events = %d, want 2; body=%s", results, body)
	}
	if !done {
		t.Fatalf("missing done event; body=%s", body)
	}
}

func TestModelTestBatchEmptyProviderList(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{response: &providers.ChatResponse{ID: "test"}}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "conn1",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "anthropic",
		Name:     "conn1",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/test-batch", `{}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(bytes.NewReader(body))
	var results int
	var done bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: result") {
			results++
		}
		if strings.HasPrefix(line, "event: done") {
			done = true
		}
	}
	if results != 2 {
		t.Fatalf("result events = %d, want 2; body=%s", results, body)
	}
	if !done {
		t.Fatalf("missing done event; body=%s", body)
	}
}

func TestModelTestMethodNotAllowed(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/providers/openai/models/gpt-4o/test", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestModelTestBatchMethodNotAllowed(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/providers/test-batch", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestModelTestProviderNotFound(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/nonexistent/models/gpt-4o/test", "")
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "provider not found") {
		t.Fatalf("body = %s, want provider not found", body)
	}
}

func TestModelTestInvalidBody(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/openai/models/gpt-4o/test", `{"messages":`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
}

func TestModelTestBatchInvalidBody(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/test-batch", `{"providers":`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
}

func TestModelTestBatchSkipsInactiveConnections(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestProvider{response: &providers.ChatResponse{ID: "test"}}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "active",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "inactive",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: false,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/test-batch", `{"providers":["openai"]}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(bytes.NewReader(body))
	var results int
	var done bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: result") {
			results++
		}
		if strings.HasPrefix(line, "event: done") {
			done = true
		}
	}
	if results != 1 {
		t.Fatalf("result events = %d, want 1; body=%s", results, body)
	}
	if !done {
		t.Fatalf("missing done event; body=%s", body)
	}
}

func TestModelTestBatchHandlesProviderPanic(t *testing.T) {
	s := newModelTestStore(t)
	provider := &fakeModelTestPanickingProvider{}
	_, baseURL := startModelTestServer(t, api.ServerConfig{
		Version:               "test",
		Store:                 s,
		ProviderAdapterSource: fakeModelTestAdapterSource{provider: provider, ok: true},
	})

	apiKey := "sk-test"
	if err := s.CreateConnection(&store.Connection{
		Provider: "openai",
		Name:     "primary",
		AuthType: store.AuthTypeAPIKey,
		APIKey:   &apiKey,
		IsActive: true,
	}); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	resp, body := postModelTestJSON(t, baseURL+"/api/providers/test-batch", `{"providers":["openai"]}`)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(bytes.NewReader(body))
	var results int
	var done bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: result") {
			results++
		}
		if strings.HasPrefix(line, "event: done") {
			done = true
		}
	}
	if results != 1 {
		t.Fatalf("result events = %d, want 1; body=%s", results, body)
	}
	if !done {
		t.Fatalf("missing done event; body=%s", body)
	}
}

type fakeModelTestPanickingProvider struct{}

func (f *fakeModelTestPanickingProvider) Name() providers.ModelProvider {
	return "openai"
}

func (f *fakeModelTestPanickingProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	panic("intentional panic")
}

func (f *fakeModelTestPanickingProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, nil
}

func (f *fakeModelTestPanickingProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}
