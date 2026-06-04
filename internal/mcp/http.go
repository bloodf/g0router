package mcp

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
)

const protocolVersion = "2025-11-25"

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPTransport struct {
	client HTTPDoer
}

func NewHTTPTransport(client HTTPDoer) *HTTPTransport {
	if client == nil {
		client = http.DefaultClient
	}
	return &HTTPTransport{client: client}
}

func (t *HTTPTransport) InitializeStreamable(ctx context.Context, url string, headers map[string]string) (string, int, error) {
	session, status, err := t.legacyInitializeStreamable(ctx, url, headers)
	if err != nil {
		return "", status, err
	}
	client := NewStreamableHTTPClient(t.client, url, headers, session, true)
	if err := client.notifyLocked(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return "", status, err
	}
	return session, status, nil
}

func (t *HTTPTransport) InitializeSSE(ctx context.Context, url string, headers map[string]string) (string, error) {
	client := NewSSEClient(t.client, url, headers)
	if err := client.ensureEndpoint(ctx); err != nil {
		return "", err
	}
	defer client.Close()
	return client.endpoint, nil
}

func (t *HTTPTransport) legacyInitializeStreamable(ctx context.Context, url string, headers map[string]string) (string, int, error) {
	encoded, err := marshalJSONRPCRequest(1, "initialize", initializeParams())
	if err != nil {
		return "", 0, fmt.Errorf("marshal mcp initialize request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		return "", 0, fmt.Errorf("build mcp initialize request: %w", err)
	}
	applyHTTPHeaders(req, headers)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", protocolVersion)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("initialize streamable mcp: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", resp.StatusCode, fmt.Errorf("initialize streamable mcp: status %d", resp.StatusCode)
	}
	return resp.Header.Get("Mcp-Session-Id"), resp.StatusCode, nil
}

func applyHTTPHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}
