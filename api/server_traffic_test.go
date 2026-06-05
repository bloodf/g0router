package api

import (
	"bufio"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/traffic"
)

// sseConnect opens a GET /api/traffic/stream connection and returns the
// response. The caller must close resp.Body when done.
// The context should be cancelled to terminate the SSE connection before the
// test ends, so that the server can shut down cleanly.
func sseConnect(t *testing.T, baseURL string, ctx context.Context) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/traffic/stream", nil)
	if err != nil {
		t.Fatalf("new SSE request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/traffic/stream: %v", err)
	}
	return resp
}

// TestTrafficStreamDeliversBrokerEvent verifies that a published traffic event
// arrives on an SSE subscriber within a short deadline.
func TestTrafficStreamDeliversBrokerEvent(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		UsageStore: s,
	})

	if srv.trafficBroker == nil {
		t.Fatal("trafficBroker is nil after NewServer")
	}

	// Cancel context to close the SSE connection before test cleanup runs.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	resp := sseConnect(t, baseURL, ctx)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	// Publish an event; give the SSE goroutine a moment to reach its select.
	time.Sleep(30 * time.Millisecond)
	srv.trafficBroker.Publish(traffic.Event{
		Timestamp:   time.Now().UTC(),
		KeyID:       "test-key",
		Provider:    "openai",
		Model:       "gpt-4o",
		StatusClass: "2xx",
		StatusCode:  200,
		LatencyMS:   55,
	})

	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				found <- line
				return
			}
		}
	}()

	select {
	case line := <-found:
		if !strings.Contains(line, `"provider":"openai"`) {
			t.Fatalf("SSE line missing provider: %q", line)
		}
		if !strings.Contains(line, `"model":"gpt-4o"`) {
			t.Fatalf("SSE line missing model: %q", line)
		}
		if !strings.Contains(line, `"status_class":"2xx"`) {
			t.Fatalf("SSE line missing status_class: %q", line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for SSE data event")
	}
}

// TestTrafficStreamReplaysRingBuffer verifies that events already in the ring
// buffer are replayed immediately on connect.
func TestTrafficStreamReplaysRingBuffer(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		UsageStore: s,
	})

	// Publish before any subscriber connects.
	srv.trafficBroker.Publish(traffic.Event{
		Provider: "anthropic",
		Model:    "claude-opus-4",
		KeyID:    "replay-key",
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	resp := sseConnect(t, baseURL, ctx)
	defer resp.Body.Close()

	found := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				found <- line
				return
			}
		}
	}()

	select {
	case line := <-found:
		if !strings.Contains(line, `"provider":"anthropic"`) {
			t.Fatalf("replay line missing provider: %q", line)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for replayed SSE event")
	}
}

// TestTrafficStreamRejects405OnPost verifies that non-GET methods get 405.
func TestTrafficStreamRejects405OnPost(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		UsageStore: s,
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/traffic/stream",
		strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/traffic/stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

// TestInferencePublishesToTrafficBroker verifies that a completed inference
// request causes the traffic broker to hold a matching event in Recent().
func TestInferencePublishesToTrafficBroker(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)

	srv, baseURL := startTestServer(t, ServerConfig{
		Port:            0,
		Version:         "test",
		Store:           s,
		UsageStore:      s,
		InferenceEngine: routeInferenceEngine{response: routeChatResponseWithUsage()},
	})

	resp, body := postAPITestJSON(t, baseURL+"/v1/chat/completions",
		`{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("inference status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	recent := srv.trafficBroker.Recent()
	if len(recent) == 0 {
		t.Fatal("trafficBroker.Recent() is empty after inference request")
	}
	ev := recent[len(recent)-1]
	if ev.Provider != "openai" {
		t.Fatalf("event provider = %q, want openai", ev.Provider)
	}
	if ev.Model != "gpt-4o" {
		t.Fatalf("event model = %q, want gpt-4o", ev.Model)
	}
	if ev.StatusClass != "2xx" {
		t.Fatalf("event status_class = %q, want 2xx", ev.StatusClass)
	}
	if ev.StatusCode != http.StatusOK {
		t.Fatalf("event status_code = %d, want 200", ev.StatusCode)
	}

	// Anchor the store import.
	var _ *store.Store = s
}
