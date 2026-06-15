package mcp

import (
	"context"
	"encoding/json"
)

// mcpServerName is the serverInfo.name advertised by g0router's MCP server mode
// (PAR-BF-MCP-002/003).
const mcpServerName = "g0router"

// JSON-RPC 2.0 error codes (the base spec the matrix rows cite — PAR-BF-MCP-075
// is HAVE-by-variant over g0router's own bridge, no mark3labs/mcp-go dep).
const (
	rpcParseError     = -32700
	rpcMethodNotFound = -32601
	rpcInternalError  = -32603
)

// ServerTool is one tool the server-mode catalog advertises over tools/list. It
// is the shared shape the admin catalog assembler fills from the existing
// CLIENT-mode aggregation (D3 — one source of truth).
type ServerTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

// CatalogSource yields the global un-scoped tool surface the server re-exposes
// (D3). The admin layer wires this to the SAME aggregation ListTools serves so
// the /api/mcp/tools DTO surface and the /mcp server-mode surface never diverge.
type CatalogSource interface {
	ListServerTools() []ServerTool
}

// ToolDispatcher executes one tool call and returns its (filtered) result text.
// It is the existing ToolExecutor seam — NewBridgeToolExecutor satisfies it — so
// the server-mode tools/call DELEGATES to the shipped bridge dispatch path
// rather than re-implementing tool execution.
type ToolDispatcher = ToolExecutor

// NewBridgeDispatcher adapts a live Bridge to the server-mode ToolDispatcher by
// reusing the shipped NewBridgeToolExecutor (which drives Bridge.Send + a
// capturing SessionSink + smartFilterText). No duplicated framing.
func NewBridgeDispatcher(b *Bridge) ToolDispatcher {
	return NewBridgeToolExecutor(b)
}

// Server is g0router's MCP server-mode JSON-RPC dispatcher. It routes the three
// matrix-evidenced methods (initialize / tools/list / tools/call) over the
// SHIPPED CLIENT-mode framing (splitFrames) and dispatch (ToolDispatcher), with
// no mark3labs/mcp-go dependency (D1). The dispatch core is pure (no I/O) so it
// is fully unit-testable with canned JSON-RPC requests + a fake dispatcher.
type Server struct {
	catalog CatalogSource
	exec    ToolDispatcher
}

// NewServer constructs the server-mode dispatcher over an injected catalog +
// tool dispatcher (so tests feed canned tools + a fake executor).
func NewServer(catalog CatalogSource, exec ToolDispatcher) *Server {
	return &Server{catalog: catalog, exec: exec}
}

// rpcRequest is the inbound JSON-RPC 2.0 request shape.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Dispatch routes one inbound JSON-RPC 2.0 frame and returns the marshalled
// JSON-RPC response (raw — never the {data,error} admin envelope). A malformed
// body returns a -32700 parse error; an unknown method returns -32601.
func (s *Server) Dispatch(ctx context.Context, body []byte) []byte {
	// Reuse the shipped newline-delimited frame splitter so a body carrying a
	// trailing newline (or a single framed line) is normalized exactly like the
	// CLIENT-mode stdin path. A single un-framed object falls through unchanged.
	if frames, _ := splitFrames(body); len(frames) == 1 {
		body = frames[0]
	}

	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return marshalRPCError(nil, rpcParseError, "parse error")
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return marshalRPCError(req.ID, rpcMethodNotFound, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req rpcRequest) []byte {
	return marshalRPCResult(req.ID, map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      map[string]any{"name": mcpServerName, "version": "1"},
	})
}

func (s *Server) handleToolsList(req rpcRequest) []byte {
	tools := []ServerTool{}
	if s.catalog != nil {
		tools = s.catalog.ListServerTools()
		if tools == nil {
			tools = []ServerTool{}
		}
	}
	return marshalRPCResult(req.ID, map[string]any{"tools": tools})
}

// toolCallParams is the tools/call params shape.
type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (s *Server) handleToolsCall(ctx context.Context, req rpcRequest) []byte {
	var params toolCallParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return marshalRPCError(req.ID, rpcParseError, "parse error")
		}
	}
	if s.exec == nil {
		return marshalRPCError(req.ID, rpcInternalError, "no tool dispatcher")
	}
	text, err := s.exec.Execute(ctx, params.Name, params.Arguments)
	if err != nil {
		return marshalRPCError(req.ID, rpcInternalError, err.Error())
	}
	// smartFilterText is idempotent (the bridge dispatcher already applies it);
	// applying it here keeps the server-mode result filtered even for a
	// dispatcher that does not (e.g. a future non-bridge source).
	return marshalRPCResult(req.ID, map[string]any{
		"content": []map[string]any{{"type": "text", "text": smartFilterText(text)}},
	})
}

// marshalRPCResult builds a JSON-RPC 2.0 success frame. A nil id marshals as
// JSON null (notification-less error path).
func marshalRPCResult(id json.RawMessage, result any) []byte {
	out := map[string]any{"jsonrpc": "2.0", "id": rawID(id), "result": result}
	b, err := json.Marshal(out)
	if err != nil {
		return marshalRPCError(id, rpcInternalError, "encode result")
	}
	return b
}

// marshalRPCError builds a JSON-RPC 2.0 error frame.
func marshalRPCError(id json.RawMessage, code int, message string) []byte {
	out := map[string]any{
		"jsonrpc": "2.0",
		"id":      rawID(id),
		"error":   map[string]any{"code": code, "message": message},
	}
	b, _ := json.Marshal(out)
	return b
}

// rawID returns the request id as a json.RawMessage, defaulting to null.
func rawID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}
