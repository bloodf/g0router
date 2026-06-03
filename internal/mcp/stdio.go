package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type StdioClient struct {
	process     Process
	reader      *bufio.Reader
	writer      *bufio.Writer
	mu          sync.Mutex
	nextID      int64
	initialized bool
}

func NewStdioClient(process Process) *StdioClient {
	return &StdioClient{
		process: process,
		reader:  bufio.NewReader(process.Stdout()),
		writer:  bufio.NewWriter(process.Stdin()),
	}
}

func (c *StdioClient) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return nil, err
	}

	var result struct {
		Tools []struct {
			Name             string          `json:"name"`
			Description      string          `json:"description"`
			InputSchema      json.RawMessage `json:"inputSchema"`
			InputSchemaSnake json.RawMessage `json:"input_schema"`
		} `json:"tools"`
	}
	if err := c.callLocked(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}

	tools := make([]Tool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		schema := tool.InputSchema
		if len(schema) == 0 {
			schema = tool.InputSchemaSnake
		}
		tools = append(tools, Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}
	return tools, nil
}

func (c *StdioClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return CallResult{}, err
	}

	var result struct {
		Content any `json:"content"`
	}
	params := map[string]any{
		"name":      req.Name,
		"arguments": rawArguments(req.Arguments),
	}
	if err := c.callLocked(ctx, "tools/call", params, &result); err != nil {
		return CallResult{}, err
	}
	return CallResult{Content: result.Content}, nil
}

func (c *StdioClient) Close() error {
	if c.process == nil {
		return nil
	}
	return c.process.Close()
}

func (c *StdioClient) ensureInitialized(ctx context.Context) error {
	if c.initialized {
		return nil
	}
	params := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "g0router",
			"version": "dev",
		},
	}
	var result map[string]any
	if err := c.callLocked(ctx, "initialize", params, &result); err != nil {
		return err
	}
	if err := c.notifyLocked("notifications/initialized", map[string]any{}); err != nil {
		return err
	}
	c.initialized = true
	return nil
}

func (c *StdioClient) callLocked(ctx context.Context, method string, params any, result any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.nextID++
	id := c.nextID
	encoded, err := marshalJSONRPCRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("marshal mcp %s request: %w", method, err)
	}
	if err := c.writeLineLocked(encoded); err != nil {
		return fmt.Errorf("write mcp %s request: %w", method, err)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("read mcp %s response: %w", method, err)
		}
		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			return fmt.Errorf("decode mcp %s response: %w", method, err)
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return resp.Error
		}
		if result == nil {
			return nil
		}
		if len(resp.Result) == 0 {
			return nil
		}
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("decode mcp %s result: %w", method, err)
		}
		return nil
	}
}

func (c *StdioClient) notifyLocked(method string, params any) error {
	encoded, err := marshalJSONRPCNotification(method, params)
	if err != nil {
		return fmt.Errorf("marshal mcp %s notification: %w", method, err)
	}
	if err := c.writeLineLocked(encoded); err != nil {
		return fmt.Errorf("write mcp %s notification: %w", method, err)
	}
	return nil
}

func (c *StdioClient) writeLineLocked(encoded []byte) error {
	if _, err := c.writer.Write(encoded); err != nil {
		return err
	}
	if err := c.writer.WriteByte('\n'); err != nil {
		return err
	}
	return c.writer.Flush()
}

func rawArguments(raw json.RawMessage) any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{}
	}
	return decoded
}
