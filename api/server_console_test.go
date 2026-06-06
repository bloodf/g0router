package api

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/console"
	"github.com/valyala/fasthttp"
)

// sseConnectConsole opens a GET /api/console-logs/stream connection.
func sseConnectConsole(t *testing.T, baseURL string, ctx context.Context) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/console-logs/stream", nil)
	if err != nil {
		t.Fatalf("new SSE request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/console-logs/stream: %v", err)
	}
	return resp
}

func TestConsoleLogsStreamReplaysAndLive(t *testing.T) {
	s := newAPITestStore(t)
	broker := console.NewBroker(10)
	broker.Publish(console.Entry{Level: "INFO", Message: "replay"})

	srv, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		UsageStore:    s,
		ConsoleBroker: broker,
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	resp := sseConnectConsole(t, baseURL, ctx)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	// Wait for the stream to start, then publish a live event.
	time.Sleep(30 * time.Millisecond)
	broker.Publish(console.Entry{Level: "WARN", Message: "live"})

	found := make(chan string, 2)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				found <- line
			}
		}
	}()

	var replayed, lived bool
	timeout := time.After(3 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case line := <-found:
			if strings.Contains(line, `"message":"replay"`) {
				replayed = true
			}
			if strings.Contains(line, `"message":"live"`) {
				lived = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for SSE events")
		}
	}
	if !replayed {
		t.Fatal("did not receive replayed event")
	}
	if !lived {
		t.Fatal("did not receive live event")
	}

	_ = srv
}

func TestConsoleLogsStreamClientDisconnect(t *testing.T) {
	s := newAPITestStore(t)
	broker := console.NewBroker(10)

	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		UsageStore:    s,
		ConsoleBroker: broker,
	})

	ctx, cancel := context.WithCancel(context.Background())
	resp := sseConnectConsole(t, baseURL, ctx)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Give the server time to register the subscriber.
	time.Sleep(30 * time.Millisecond)

	// Cancel the client context to force disconnect.
	cancel()

	// Drain the body so the connection closes.
	_, _ = io.Copy(io.Discard, resp.Body)

	// Give the server time to process the disconnect.
	time.Sleep(50 * time.Millisecond)

	// After disconnect there should be no subscribers.
	// We verify this by publishing; if no panic occurs the broker is fine.
	broker.Publish(console.Entry{Level: "INFO", Message: "after disconnect"})
}

func TestConsoleLogsStreamNilBrokerUnavailable(t *testing.T) {
	srv := &Server{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	srv.handleConsoleLogsStream(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestConsoleLogsStreamRejects405OnPost(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		UsageStore: s,
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/console-logs/stream", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /api/console-logs/stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestConsoleLogsStreamStopEndsStream(t *testing.T) {
	s := newAPITestStore(t)
	broker := console.NewBroker(10)
	srv, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		UsageStore:    s,
		ConsoleBroker: broker,
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	resp := sseConnectConsole(t, baseURL, ctx)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	time.Sleep(30 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
		}
		close(done)
	}()

	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("SSE stream did not end after Stop()")
	}
}

func TestConsoleBrokerWired(t *testing.T) {
	broker := console.NewBroker(10)
	srv := NewServer(ServerConfig{ConsoleBroker: broker})
	if srv.consoleBroker != broker {
		t.Fatal("consoleBroker not wired into Server")
	}
}

func TestTeeHandlerPublishesToBroker(t *testing.T) {
	broker := console.NewBroker(10)
	baseHandler := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(console.NewTeeHandler(baseHandler, broker, slog.LevelDebug))

	logger.Info("tee integration", slog.String("key", "value"))

	recent := broker.Recent()
	if len(recent) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(recent))
	}
	if recent[0].Message != "tee integration" {
		t.Fatalf("message = %q, want tee integration", recent[0].Message)
	}
	if recent[0].Level != "INFO" {
		t.Fatalf("level = %q, want INFO", recent[0].Level)
	}
}

func TestConsoleLogsClearEndpoint(t *testing.T) {
	s := newAPITestStore(t)
	broker := console.NewBroker(10)
	broker.Publish(console.Entry{Level: "INFO", Message: "before"})

	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		UsageStore:    s,
		ConsoleBroker: broker,
	})

	req, err := http.NewRequest(http.MethodDelete, baseURL+"/api/console-logs", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/console-logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}

	if len(broker.Recent()) != 0 {
		t.Fatalf("broker not cleared, recent = %d", len(broker.Recent()))
	}
}

func TestConsoleLogsClearNilBrokerUnavailable(t *testing.T) {
	s := newAPITestStore(t)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:       0,
		Version:    "test",
		Store:      s,
		UsageStore: s,
	})

	req, err := http.NewRequest(http.MethodDelete, baseURL+"/api/console-logs", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/console-logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestConsoleLogsClearRejects405OnGet(t *testing.T) {
	s := newAPITestStore(t)
	broker := console.NewBroker(10)
	_, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		UsageStore:    s,
		ConsoleBroker: broker,
	})

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/console-logs", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-API-Key", testHarnessAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /api/console-logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}
