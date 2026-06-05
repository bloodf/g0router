package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamableHTTPClientInitializeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()
	client := NewStreamableHTTPClient(server.Client(), server.URL, nil, "", false)
	if _, err := client.ListTools(t.Context()); err == nil {
		t.Fatal("initialize failure: want error")
	}
}

func TestStreamableHTTPClientToolsListRPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		if _, hasID := req["id"]; !hasID {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "s")
		switch method {
		case "initialize":
			writeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": protocolVersion})
		default:
			writeRPCError(t, w, req["id"], -32601, "unknown method")
		}
	}))
	defer server.Close()
	client := NewStreamableHTTPClient(server.Client(), server.URL, nil, "", false)
	if _, err := client.ListTools(t.Context()); err == nil {
		t.Fatal("tools/list rpc error: want error")
	}
}

func TestStreamableHTTPClientCallToolRPCError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		method, _ := req["method"].(string)
		if _, hasID := req["id"]; !hasID {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Mcp-Session-Id", "s")
		switch method {
		case "initialize":
			writeRPCResult(t, w, req["id"], map[string]any{"protocolVersion": protocolVersion})
		case "tools/list":
			writeRPCResult(t, w, req["id"], map[string]any{"tools": []map[string]any{}})
		default:
			writeRPCError(t, w, req["id"], -32000, "tool failed")
		}
	}))
	defer server.Close()
	client := NewStreamableHTTPClient(server.Client(), server.URL, nil, "", false)
	if _, err := client.ListTools(t.Context()); err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if _, err := client.CallTool(t.Context(), CallRequest{Name: "x"}); err == nil {
		t.Fatal("tools/call rpc error: want error")
	}
}

func TestStreamableHTTPClientNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	client := server.Client()
	server.Close()
	c := NewStreamableHTTPClient(client, url, nil, "", false)
	if _, err := c.ListTools(t.Context()); err == nil {
		t.Fatal("network error: want error")
	}
}
