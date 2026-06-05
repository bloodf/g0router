package api

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

func enableCache(t *testing.T, s *store.Store, ttlSeconds int) {
	t.Helper()
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.CacheEnabled = true
	settings.CacheTTLSeconds = ttlSeconds
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
}

func TestCacheServesIdenticalNonStreamingRequestFromCache(t *testing.T) {
	s := newAPITestStore(t)
	enableCache(t, s, 300)

	var received []providers.ChatRequest
	engine := &dispatchCountingEngine{response: routeChatResponseWithUsage(), received: &received}

	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: engine,
	})

	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`

	resp1, b1 := postAPITestJSON(t, baseURL+"/v1/chat/completions", body)
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("first status = %d; body=%s", resp1.StatusCode, b1)
	}
	if got := resp1.Header.Get("X-Cache"); got != "MISS" {
		t.Fatalf("first X-Cache = %q, want MISS", got)
	}

	// Reorder fields to confirm canonical-key matching produces a hit.
	resp2, b2 := postAPITestJSON(t, baseURL+"/v1/chat/completions", `{"messages":[{"role":"user","content":"hello"}],"model":"gpt-4o"}`)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second status = %d; body=%s", resp2.StatusCode, b2)
	}
	if got := resp2.Header.Get("X-Cache"); got != "HIT" {
		t.Fatalf("second X-Cache = %q, want HIT", got)
	}
	if string(b1) != string(b2) {
		t.Fatalf("cached body mismatch:\n%s\n%s", b1, b2)
	}
	if engine.dispatches() != 1 {
		t.Fatalf("engine dispatched %d times, want 1", engine.dispatches())
	}
}

func TestCacheDisabledAlwaysDispatches(t *testing.T) {
	s := newAPITestStore(t)
	// Cache stays disabled (default).

	engine := &dispatchCountingEngine{response: routeChatResponseWithUsage()}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: engine,
	})

	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hello"}]}`
	for i := 0; i < 2; i++ {
		resp, b := postAPITestJSON(t, baseURL+"/v1/chat/completions", body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d; body=%s", resp.StatusCode, b)
		}
		if got := resp.Header.Get("X-Cache"); got != "" {
			t.Fatalf("X-Cache = %q, want empty when cache disabled", got)
		}
	}
	if engine.dispatches() != 2 {
		t.Fatalf("engine dispatched %d times, want 2", engine.dispatches())
	}
}

func TestCacheNeverCachesStreamingRequests(t *testing.T) {
	s := newAPITestStore(t)
	enableCache(t, s, 300)

	makeChunks := func() <-chan providers.StreamChunk {
		ch := make(chan providers.StreamChunk, 1)
		ch <- providers.StreamChunk{ID: "c", Object: "chat.completion.chunk", Model: "gpt-4o"}
		close(ch)
		return ch
	}

	engine := &dispatchCountingEngine{streamFactory: makeChunks}
	_, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: engine,
	})

	body := `{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}],"stream":true}`
	for i := 0; i < 2; i++ {
		resp, b := postAPITestJSON(t, baseURL+"/v1/chat/completions", body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d; body=%s", resp.StatusCode, b)
		}
		if got := resp.Header.Get("X-Cache"); got == "HIT" {
			t.Fatalf("streaming request must never be a cache HIT")
		}
	}
	if engine.streamDispatches() != 2 {
		t.Fatalf("stream dispatched %d times, want 2 (never cached)", engine.streamDispatches())
	}
}

// dispatchCountingEngine counts Dispatch and DispatchStream calls. It mirrors
// routeInferenceEngine but is safe for concurrent counting and supports a fresh
// stream channel per call.
type dispatchCountingEngine struct {
	mu            sync.Mutex
	dispatchCount int
	streamCount   int
	response      *providers.ChatResponse
	streamFactory func() <-chan providers.StreamChunk
	received      *[]providers.ChatRequest
}

func (e *dispatchCountingEngine) Dispatch(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	e.mu.Lock()
	e.dispatchCount++
	if e.received != nil {
		*e.received = append(*e.received, *req)
	}
	e.mu.Unlock()
	return e.response, nil
}

func (e *dispatchCountingEngine) DispatchStream(ctx context.Context, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	e.mu.Lock()
	e.streamCount++
	e.mu.Unlock()
	if e.streamFactory != nil {
		return e.streamFactory(), nil
	}
	return nil, nil
}

func (e *dispatchCountingEngine) ListModels(ctx context.Context) ([]providers.Model, error) {
	return nil, nil
}

func (e *dispatchCountingEngine) dispatches() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dispatchCount
}

func (e *dispatchCountingEngine) streamDispatches() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.streamCount
}
