package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// sseTestServer builds a minimal SSE MCP server that enqueues JSON-RPC
// responses via the returned channel. The caller closes the channel when done.
func sseTestServer(t *testing.T) (*httptest.Server, chan map[string]any, *sync.Mutex, *[]string) {
	t.Helper()
	responses := make(chan map[string]any, 8)
	var mu sync.Mutex
	var methods []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Error("writer not a flusher")
				return
			}
			_, _ = w.Write([]byte("event: endpoint\ndata: /message\n\n"))
			flusher.Flush()
			for resp := range responses {
				encoded, _ := json.Marshal(resp)
				_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
				flusher.Flush()
			}
		case r.Method == http.MethodPost && r.URL.Path == "/message":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode: %v", err)
				return
			}
			method, _ := req["method"].(string)
			mu.Lock()
			methods = append(methods, method)
			mu.Unlock()
			if _, hasID := req["id"]; !hasID {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			switch method {
			case "initialize":
				responses <- rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
			default:
				responses <- rpcError(req["id"], -32601, "unknown: "+method)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	return server, responses, &mu, &methods
}

// TestSSEClientListToolsError covers the SSEClient.ListTools error path
// when the RPC call returns a JSON-RPC error.
func TestSSEClientListToolsError(t *testing.T) {
	server, responses, _, _ := sseTestServer(t)
	defer server.Close()
	defer close(responses)

	client := NewSSEClient(server.Client(), server.URL, nil)
	// The fake server returns error for any unknown method including tools/list.
	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("ListTools error = nil, want RPC error")
	}
}

// TestSSEClientCallToolEnsureInitializedError covers the SSEClient.CallTool
// ensureInitialized error path by using a server that returns an error for initialize.
func TestSSEClientCallToolInitError(t *testing.T) {
	responses := make(chan map[string]any, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			_, _ = w.Write([]byte("event: endpoint\ndata: /message\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			for resp := range responses {
				encoded, _ := json.Marshal(resp)
				_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
				if flusher != nil {
					flusher.Flush()
				}
			}
		case r.Method == http.MethodPost:
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			if _, hasID := req["id"]; hasID {
				// Return error for initialize
				responses <- rpcError(req["id"], -32603, "internal error")
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer close(responses)

	client := NewSSEClient(server.Client(), server.URL, nil)
	_, err := client.CallTool(context.Background(), CallRequest{Name: "search"})
	if err == nil {
		t.Fatal("CallTool error = nil, want initialize error")
	}
}

// TestSSEClientCallToolRPCError covers the CallTool error path after successful
// initialization but a failing tools/call RPC.
func TestSSEClientCallToolRPCError(t *testing.T) {
	responses := make(chan map[string]any, 8)
	callCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			_, _ = w.Write([]byte("event: endpoint\ndata: /message\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			for resp := range responses {
				encoded, _ := json.Marshal(resp)
				_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
				if flusher != nil {
					flusher.Flush()
				}
			}
		case r.Method == http.MethodPost:
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			method, _ := req["method"].(string)
			if _, hasID := req["id"]; !hasID {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			mu.Lock()
			callCount++
			mu.Unlock()
			switch method {
			case "initialize":
				responses <- rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
			case "tools/call":
				responses <- rpcError(req["id"], -32000, "tool failed")
			default:
				responses <- rpcError(req["id"], -32601, "unknown")
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer close(responses)

	client := NewSSEClient(server.Client(), server.URL, nil)
	// First call: initialize + notify + tools/call (error)
	_, err := client.CallTool(context.Background(), CallRequest{Name: "search"})
	if err == nil {
		t.Fatal("CallTool error = nil, want RPC error from tools/call")
	}
}

// TestSSEClientNotifyLockedPostError covers the notifyLocked error path by
// using a server that accepts the SSE connection and initialize, closes
// immediately after initialize, so the POST for notifications/initialized fails.
func TestSSEClientNotifyLockedPostError(t *testing.T) {
	responses := make(chan map[string]any, 4)
	// postCount tracks how many POST requests have been received.
	var postCount int
	var postMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			_, _ = w.Write([]byte("event: endpoint\ndata: /message\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
			for resp := range responses {
				encoded, _ := json.Marshal(resp)
				_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
				if flusher != nil {
					flusher.Flush()
				}
			}
		case r.Method == http.MethodPost:
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			method, _ := req["method"].(string)
			postMu.Lock()
			postCount++
			n := postCount
			postMu.Unlock()
			if method == "initialize" && n == 1 {
				// Respond to initialize successfully.
				responses <- rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
				w.WriteHeader(http.StatusAccepted)
				return
			}
			// For notifications/initialized and any subsequent calls: return 500.
			w.WriteHeader(http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer close(responses)

	client := NewSSEClient(server.Client(), server.URL, nil)
	// ensureInitialized will: call initialize (succeeds), then notifyLocked
	// (posts notifications/initialized → server returns 500 → post error).
	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("ListTools error = nil, want notifyLocked post error")
	}
}
