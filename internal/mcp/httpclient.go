package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type StreamableHTTPClient struct {
	client      HTTPDoer
	url         string
	headers     map[string]string
	sessionID   string
	mu          sync.Mutex
	nextID      int64
	initialized bool
}

func NewStreamableHTTPClient(client HTTPDoer, url string, headers map[string]string, sessionID string, initialized bool) *StreamableHTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &StreamableHTTPClient{
		client:      client,
		url:         url,
		headers:     copyMap(headers),
		sessionID:   sessionID,
		initialized: initialized,
	}
}

func (c *StreamableHTTPClient) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	var result toolsListResult
	if err := c.callLocked(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return result.Tools(), nil
}

func (c *StreamableHTTPClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return CallResult{}, err
	}
	var result struct {
		Content any `json:"content"`
	}
	params := map[string]any{"name": req.Name, "arguments": rawArguments(req.Arguments)}
	if err := c.callLocked(ctx, "tools/call", params, &result); err != nil {
		return CallResult{}, err
	}
	return CallResult{Content: result.Content}, nil
}

func (c *StreamableHTTPClient) Close() error {
	return nil
}

func (c *StreamableHTTPClient) ensureInitialized(ctx context.Context) error {
	if c.initialized {
		return nil
	}
	params := initializeParams()
	var result map[string]any
	if err := c.callLocked(ctx, "initialize", params, &result); err != nil {
		return err
	}
	if err := c.notifyLocked(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return err
	}
	c.initialized = true
	return nil
}

func (c *StreamableHTTPClient) callLocked(ctx context.Context, method string, params any, result any) error {
	c.nextID++
	id := c.nextID
	encoded, err := marshalJSONRPCRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("marshal mcp %s request: %w", method, err)
	}
	resp, err := c.post(ctx, encoded)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if session := resp.Header.Get("Mcp-Session-Id"); session != "" {
		c.sessionID = session
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp %s request: status %d", method, resp.StatusCode)
	}
	body, err := responseBody(resp)
	if err != nil {
		return fmt.Errorf("read mcp %s response: %w", method, err)
	}
	return decodeJSONRPCResult(method, body, id, result)
}

func (c *StreamableHTTPClient) notifyLocked(ctx context.Context, method string, params any) error {
	encoded, err := marshalJSONRPCNotification(method, params)
	if err != nil {
		return fmt.Errorf("marshal mcp %s notification: %w", method, err)
	}
	resp, err := c.post(ctx, encoded)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp %s notification: status %d", method, resp.StatusCode)
	}
	return nil
}

func (c *StreamableHTTPClient) post(ctx context.Context, encoded []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build streamable mcp request: %w", err)
	}
	applyHTTPHeaders(req, c.headers)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", protocolVersion)
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send streamable mcp request: %w", err)
	}
	return resp, nil
}

type SSEClient struct {
	client      HTTPDoer
	baseURL     string
	headers     map[string]string
	endpoint    string
	body        io.Closer
	reader      *bufio.Reader
	mu          sync.Mutex
	nextID      int64
	initialized bool
}

func NewSSEClient(client HTTPDoer, baseURL string, headers map[string]string) *SSEClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &SSEClient{client: client, baseURL: strings.TrimRight(baseURL, "/"), headers: copyMap(headers)}
}

func (c *SSEClient) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}
	var result toolsListResult
	if err := c.callLocked(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return result.Tools(), nil
}

func (c *SSEClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return CallResult{}, err
	}
	var result struct {
		Content any `json:"content"`
	}
	if err := c.callLocked(ctx, "tools/call", map[string]any{"name": req.Name, "arguments": rawArguments(req.Arguments)}, &result); err != nil {
		return CallResult{}, err
	}
	return CallResult{Content: result.Content}, nil
}

func (c *SSEClient) Close() error {
	if c.body == nil {
		return nil
	}
	return c.body.Close()
}

func (c *SSEClient) ensureInitialized(ctx context.Context) error {
	if err := c.ensureEndpoint(ctx); err != nil {
		return err
	}
	if c.initialized {
		return nil
	}
	var result map[string]any
	if err := c.callLocked(ctx, "initialize", initializeParams(), &result); err != nil {
		return err
	}
	if err := c.notifyLocked(ctx, "notifications/initialized", map[string]any{}); err != nil {
		return err
	}
	c.initialized = true
	return nil
}

func (c *SSEClient) ensureEndpoint(ctx context.Context) error {
	if c.endpoint != "" && c.reader != nil {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/sse", nil)
	if err != nil {
		return fmt.Errorf("build sse mcp request: %w", err)
	}
	applyHTTPHeaders(req, c.headers)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("MCP-Protocol-Version", protocolVersion)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("connect sse mcp: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return fmt.Errorf("connect sse mcp: status %d", resp.StatusCode)
	}
	c.body = resp.Body
	c.reader = bufio.NewReader(resp.Body)
	endpoint, err := readSSEEndpoint(c.reader)
	if err != nil {
		return err
	}
	c.endpoint = resolveEndpoint(c.baseURL, endpoint)
	return nil
}

func (c *SSEClient) callLocked(ctx context.Context, method string, params any, result any) error {
	c.nextID++
	id := c.nextID
	encoded, err := marshalJSONRPCRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("marshal mcp %s request: %w", method, err)
	}
	if err := c.post(ctx, encoded); err != nil {
		return err
	}
	body, err := readSSEData(c.reader)
	if err != nil {
		return fmt.Errorf("read mcp %s sse response: %w", method, err)
	}
	return decodeJSONRPCResult(method, body, id, result)
}

func (c *SSEClient) notifyLocked(ctx context.Context, method string, params any) error {
	encoded, err := marshalJSONRPCNotification(method, params)
	if err != nil {
		return fmt.Errorf("marshal mcp %s notification: %w", method, err)
	}
	return c.post(ctx, encoded)
}

func (c *SSEClient) post(ctx context.Context, encoded []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("build sse mcp message request: %w", err)
	}
	applyHTTPHeaders(req, c.headers)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("MCP-Protocol-Version", protocolVersion)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("send sse mcp message request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("send sse mcp message request: status %d", resp.StatusCode)
	}
	return nil
}

type toolsListResult struct {
	ToolsData []struct {
		Name             string          `json:"name"`
		Description      string          `json:"description"`
		InputSchema      json.RawMessage `json:"inputSchema"`
		InputSchemaSnake json.RawMessage `json:"input_schema"`
	} `json:"tools"`
}

func (r toolsListResult) Tools() []Tool {
	tools := make([]Tool, 0, len(r.ToolsData))
	for _, item := range r.ToolsData {
		schema := item.InputSchema
		if len(schema) == 0 {
			schema = item.InputSchemaSnake
		}
		tools = append(tools, Tool{Name: item.Name, Description: item.Description, InputSchema: schema})
	}
	return tools
}

func initializeParams() map[string]any {
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]string{"name": "g0router", "version": "dev"},
	}
}

func responseBody(resp *http.Response) ([]byte, error) {
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return readSSEData(bufio.NewReader(resp.Body))
	}
	return io.ReadAll(resp.Body)
}

func decodeJSONRPCResult(method string, body []byte, id int64, result any) error {
	var resp jsonrpcResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode mcp %s response: %w", method, err)
	}
	if resp.ID != id {
		return fmt.Errorf("decode mcp %s response: mismatched id %d", method, resp.ID)
	}
	if resp.Error != nil {
		return resp.Error
	}
	if result == nil || len(resp.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Result, result); err != nil {
		return fmt.Errorf("decode mcp %s result: %w", method, err)
	}
	return nil
}

func readSSEEndpoint(reader *bufio.Reader) (string, error) {
	for {
		event, data, err := readSSEEvent(reader)
		if err != nil {
			return "", fmt.Errorf("read sse endpoint: %w", err)
		}
		if event == "endpoint" && strings.TrimSpace(string(data)) != "" {
			return strings.TrimSpace(string(data)), nil
		}
	}
}

func readSSEData(reader *bufio.Reader) ([]byte, error) {
	for {
		event, data, err := readSSEEvent(reader)
		if err != nil {
			return nil, err
		}
		if event == "endpoint" {
			continue
		}
		if len(bytes.TrimSpace(data)) > 0 {
			return bytes.TrimSpace(data), nil
		}
	}
}

func readSSEEvent(reader *bufio.Reader) (string, []byte, error) {
	var event string
	var data []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return event, []byte(strings.Join(data, "\n")), nil
		}
		if strings.HasPrefix(line, "event:") {
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
}

func resolveEndpoint(baseURL, endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.IsAbs() {
		return endpoint
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return endpoint
	}
	rel, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	return base.ResolveReference(rel).String()
}
