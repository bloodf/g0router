package g0router_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/bloodf/g0router/api"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func TestE2EServerAPIKeyRequestAndUsage(t *testing.T) {
	const apiKeySecret = "test-secret"

	s := newE2EStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.EnableRequestLogs = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	_, baseURL := startE2EServer(t, api.ServerConfig{
		Port:            0,
		Version:         "e2e-test",
		RequireAPIKey:   true,
		APIKeySecret:    apiKeySecret,
		APIKeyValidator: storeAPIKeyValidator{s: s},
		InferenceEngine: e2eInferenceEngine{},
		Store:           s,
		UsageStore:      s,
	})

	rawKey := createE2EAPIKey(t, s, apiKeySecret, "default")
	response := postE2EChatCompletion(t, baseURL, rawKey)

	if response.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", response.Model)
	}
	if response.Usage == nil || response.Usage.TotalTokens != 7 {
		t.Fatalf("usage = %+v, want total tokens", response.Usage)
	}

	summary := getE2EUsageSummary(t, baseURL, rawKey)
	if summary.RequestCount != 1 {
		t.Fatalf("request count = %d, want 1", summary.RequestCount)
	}
	if summary.TotalTokens != 7 {
		t.Fatalf("total tokens = %d, want 7", summary.TotalTokens)
	}
}

type storeAPIKeyValidator struct {
	s *store.Store
}

func (v storeAPIKeyValidator) ValidateAPIKey(key, secret string) (bool, error) {
	_, ok, err := v.s.ValidateAPIKey(key, secret)
	return ok, err
}

type e2eInferenceEngine struct{}

func (e2eInferenceEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	finish := "stop"
	return &providers.ChatResponse{
		ID:      "chatcmpl-e2e",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []providers.Choice{{
			Index:        0,
			Message:      providers.Message{Role: "assistant", Content: "pong"},
			FinishReason: &finish,
		}},
		Usage: &providers.Usage{
			PromptTokens:     3,
			CompletionTokens: 4,
			TotalTokens:      7,
		},
	}, nil
}

func (e2eInferenceEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	ch := make(chan providers.StreamChunk)
	close(ch)
	return ch, nil
}

func (e2eInferenceEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{{ID: "gpt-4o", Object: "model", Provider: providers.ProviderOpenAI}}, nil
}

func newE2EStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(filepath.Join(t.TempDir(), "g0router.db"))
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

func startE2EServer(t *testing.T, config api.ServerConfig) (*api.Server, string) {
	t.Helper()
	srv := api.NewServer(config)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })
	return srv, "http://" + ln.Addr().String()
}

func createE2EAPIKey(t *testing.T, s *store.Store, secret, name string) string {
	t.Helper()
	_, raw, err := s.CreateAPIKey(name, secret)
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if raw == "" {
		t.Fatal("raw api key is empty")
	}
	return raw
}

func postE2EChatCompletion(t *testing.T, baseURL, rawKey string) providers.ChatResponse {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewBufferString(`{"model":"gpt-4o","messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatalf("new chat request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+rawKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /v1/chat/completions: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /v1/chat/completions status = %d, want 200; body = %s", resp.StatusCode, readBody(t, resp.Body))
	}

	var decoded providers.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode chat response: %v", err)
	}
	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("X-Request-ID header is empty")
	}
	return decoded
}

func getE2EUsageSummary(t *testing.T, baseURL, rawKey string) struct {
	RequestCount int64 `json:"request_count"`
	TotalTokens  int64 `json:"total_tokens"`
} {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/usage/summary", nil)
	if err != nil {
		t.Fatalf("new usage summary request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+rawKey)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/usage/summary: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/usage/summary status = %d, want 200; body = %s", resp.StatusCode, readBody(t, resp.Body))
	}

	var decoded struct {
		RequestCount int64 `json:"request_count"`
		TotalTokens  int64 `json:"total_tokens"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode usage summary: %v", err)
	}
	return decoded
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}

func readBody(t *testing.T, r io.Reader) string {
	t.Helper()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(body)
}
