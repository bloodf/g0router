package mcp

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestStreamableHTTPClientListsAndCallsTools(t *testing.T) {
	var methods []string
	var gotSession string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		gotSession = r.Header.Get("Mcp-Session-Id")
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		method, _ := req["method"].(string)
		methods = append(methods, method)
		if _, hasID := req["id"]; !hasID {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "session-123")
		switch method {
		case "initialize":
			writeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/list":
			writeRPCResult(t, w, req["id"], map[string]any{"tools": []map[string]any{{"name": "search", "description": "Search docs", "inputSchema": map[string]any{"type": "object"}}}})
		case "tools/call":
			writeRPCResult(t, w, req["id"], map[string]any{"content": []map[string]any{{"type": "text", "text": "found"}}})
		default:
			writeRPCError(t, w, req["id"], -32601, "unknown method")
		}
	}))
	defer server.Close()

	client := NewStreamableHTTPClient(server.Client(), server.URL, nil, "", false)
	tools, err := client.ListTools(t.Context())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "search" {
		t.Fatalf("tools = %+v, want search", tools)
	}
	if gotSession != "session-123" {
		t.Fatalf("session header on tools/list = %q, want session-123", gotSession)
	}
	result, err := client.CallTool(t.Context(), CallRequest{Name: "search", Arguments: json.RawMessage(`{"query":"mcp"}`)})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.Content == nil {
		t.Fatal("call result content is nil")
	}
	want := []string{"initialize", "notifications/initialized", "tools/list", "tools/call"}
	if !stringSlicesEqual(methods, want) {
		t.Fatalf("methods = %#v, want %#v", methods, want)
	}
}

func TestSSEClientDiscoversEndpointAndCallsTools(t *testing.T) {
	responses := make(chan map[string]any, 4)
	var methods []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("response writer does not flush")
			}
			_, _ = w.Write([]byte("event: endpoint\ndata: /message\n\n"))
			flusher.Flush()
			for resp := range responses {
				encoded, err := json.Marshal(resp)
				if err != nil {
					t.Errorf("marshal response: %v", err)
					return
				}
				_, _ = w.Write([]byte("data: " + string(encoded) + "\n\n"))
				flusher.Flush()
			}
		case r.Method == http.MethodPost && r.URL.Path == "/message":
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
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
			case "tools/list":
				responses <- rpcResult(req["id"], map[string]any{"tools": []map[string]any{{"name": "search", "description": "Search docs", "inputSchema": map[string]any{"type": "object"}}}})
			case "tools/call":
				responses <- rpcResult(req["id"], map[string]any{"content": []map[string]any{{"type": "text", "text": "found"}}})
			default:
				responses <- rpcError(req["id"], -32601, "unknown method")
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer close(responses)

	client := NewSSEClient(server.Client(), server.URL, nil)
	tools, err := client.ListTools(t.Context())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "search" {
		t.Fatalf("tools = %+v, want search", tools)
	}
	if _, err := client.CallTool(t.Context(), CallRequest{Name: "search", Arguments: json.RawMessage(`{"query":"mcp"}`)}); err != nil {
		t.Fatalf("CallTool: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), methods...)
	mu.Unlock()
	want := []string{"initialize", "notifications/initialized", "tools/list", "tools/call"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("methods = %#v, want %#v", got, want)
	}
}

func TestParseSSEDataSkipsEndpointEvents(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("event: endpoint\ndata: /message\n\ndata: {\"ok\":true}\n\n"))
	payload, err := readSSEData(reader)
	if err != nil {
		t.Fatalf("readSSEData: %v", err)
	}
	if string(payload) != `{"ok":true}` {
		t.Fatalf("payload = %s, want JSON response", payload)
	}
}

func writeRPCResult(t *testing.T, w http.ResponseWriter, id any, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rpcResult(id, result)); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func writeRPCError(t *testing.T, w http.ResponseWriter, id any, code int, message string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rpcError(id, code, message)); err != nil {
		t.Fatalf("encode error: %v", err)
	}
}
