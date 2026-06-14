package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultHTTPClient is the shared nil-fallback HTTP client for every MCP network
// component (probe, registry, OAuth engine, SSE transport). It mirrors the
// nil-able client default in internal/auth/oauth.go:128-134.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
}

// sseClient is the CLIENT side of the bridge transport: it connects to a remote
// MCP server over Server-Sent Events, reads the initial "endpoint" event to learn
// the message URL, then POSTs JSON-RPC messages to that URL (mirrors
// src/app/api/mcp/[plugin]/{sse,message}/route.js — PAR-MCP-001/002/055/056, client
// half). The PURE frame parsers below are the fully unit-tested core; the live
// streaming reader (Stream) is the thin integration-only surface.
type sseClient struct {
	client *http.Client
}

// newSSEClient builds a client. A nil http.Client falls back to the package default
// (mirrors internal/auth/oauth.go:128's nil-able client).
func newSSEClient(client *http.Client) *sseClient {
	if client == nil {
		client = defaultHTTPClient()
	}
	return &sseClient{client: client}
}

// postMessage POSTs a JSON-RPC frame to the server's message URL and expects a 202
// Accepted (mirrors message/route.js:17 "sendToChild + 202").
func (c *sseClient) postMessage(ctx context.Context, messageURL string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messageURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build message request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("post mcp message: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("mcp message endpoint returned %d", resp.StatusCode)
	}
	return nil
}

// Stream connects to the remote SSE endpoint, reads the first "endpoint" event to
// discover the message URL, and dispatches every subsequent "data:" frame to sink.
// INTEGRATION-ONLY: it opens a long-lived text/event-stream response and is never
// exercised by a unit test (the frame parsers + postMessage carry the tested logic).
func (c *sseClient) Stream(ctx context.Context, sseURL string, sink SessionSink) (messageURL string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sseURL, nil)
	if err != nil {
		return "", fmt.Errorf("build sse request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connect sse: %w", err)
	}
	defer resp.Body.Close()

	var block []byte
	buf := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			block = append(block, buf[:n]...)
			for {
				i := bytes.Index(block, []byte("\n\n"))
				if i < 0 {
					break
				}
				event, data := parseSSEFrame(block[:i+2])
				block = block[i+2:]
				if event == "endpoint" && messageURL == "" {
					messageURL = data
					continue
				}
				if data != "" && sink != nil {
					_ = sink([]byte(data))
				}
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				return messageURL, nil
			}
			return messageURL, fmt.Errorf("read sse stream: %w", rerr)
		}
	}
}

// parseSSEFrame parses one SSE event block ("event: X\ndata: Y\n\n") into its
// event name and data payload. PURE — no I/O. An absent event line yields "".
// Mirrors the client side of sse/route.js:22.
func parseSSEFrame(block []byte) (event, data string) {
	for _, line := range strings.Split(string(block), "\n") {
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " ")
		}
	}
	return event, data
}

// parseSSEDataFrames extracts every "data:" JSON payload from an SSE text body
// (PAR-MCP-012; mirrors cowork-mcp-tools/route.js:60-69 — split on "\n", keep lines
// starting with "data:", strip the prefix and one optional leading space). PURE.
func parseSSEDataFrames(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		out = append(out, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
	}
	return out
}
