package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// defaultAgentMaxTurns bounds an agent loop when the caller supplies a
// non-positive maxTurns. It mirrors a conservative agent depth (PAR-MCP-040).
const defaultAgentMaxTurns = 8

// toolCallTimeout bounds a single bridge tool-call round-trip so a silent child
// can never wedge the loop.
const toolCallTimeout = 30 * time.Second

// ErrMaxTurnsExceeded is returned by Agent.Run when the model keeps requesting
// tool calls past the hard maxTurns bound (no runaway loop — PAR-MCP-040).
var ErrMaxTurnsExceeded = errors.New("mcp: agent max turns exceeded")

// ToolExecutor runs one tool call and returns its (filtered) result text. The
// real impl drives a Bridge.Send / sseClient.postMessage (bridgeToolExecutor);
// the test impl returns canned results — so the loop is fully unit-testable with
// NO real process or network.
type ToolExecutor interface {
	Execute(ctx context.Context, name string, args map[string]any) (string, error)
}

// AgentMessage is one entry in the running conversation the model step sees. Role
// is one of "user" | "assistant" | "tool".
type AgentMessage struct {
	Role    string
	Content string
}

// AgentRequest is the initial input to an agent run.
type AgentRequest struct {
	Prompt string
}

// ModelDecision is one model turn: either a tool call to execute (ToolName set)
// or a final answer (Final set). A non-empty ToolName takes precedence.
type ModelDecision struct {
	ToolName string
	ToolArgs map[string]any
	Final    string
}

// ModelStep is the injected model round-trip: given the running history it returns
// the next decision. The real production wiring supplies an LLM-backed step
// (INTEGRATION-ONLY); the unit tests supply a deterministic fake.
type ModelStep func(ctx context.Context, history []AgentMessage) (ModelDecision, error)

// AgentResult is the outcome of an agent run.
type AgentResult struct {
	Answer string
	Turns  int
}

// Agent runs a bounded multi-turn tool-execution loop: it asks the model for the
// next step, executes any requested tool via the ToolExecutor, appends the result
// to the history, and repeats until the model returns a final answer OR maxTurns
// is hit (the bound). PAR-MCP-040.
type Agent struct {
	exec     ToolExecutor
	maxTurns int
}

// NewAgent constructs an Agent. A non-positive maxTurns falls back to
// defaultAgentMaxTurns so the loop is always bounded.
func NewAgent(exec ToolExecutor, maxTurns int) *Agent {
	if maxTurns <= 0 {
		maxTurns = defaultAgentMaxTurns
	}
	return &Agent{exec: exec, maxTurns: maxTurns}
}

// Run drives the loop given an initial request and the injected modelStep. It
// returns the final answer, or ErrMaxTurnsExceeded if the model never finishes,
// or the surfaced tool error if a tool execution fails (not swallowed).
func (a *Agent) Run(ctx context.Context, req AgentRequest, modelStep ModelStep) (AgentResult, error) {
	history := []AgentMessage{{Role: "user", Content: req.Prompt}}
	turns := 0
	for turns < a.maxTurns {
		turns++
		decision, err := modelStep(ctx, history)
		if err != nil {
			return AgentResult{Turns: turns}, fmt.Errorf("agent model step: %w", err)
		}
		if decision.ToolName == "" {
			return AgentResult{Answer: decision.Final, Turns: turns}, nil
		}
		result, err := a.exec.Execute(ctx, decision.ToolName, decision.ToolArgs)
		if err != nil {
			return AgentResult{Turns: turns}, fmt.Errorf("agent tool %q: %w", decision.ToolName, err)
		}
		history = append(history,
			AgentMessage{Role: "assistant", Content: "tool_call:" + decision.ToolName},
			AgentMessage{Role: "tool", Content: result},
		)
	}
	return AgentResult{Turns: turns}, ErrMaxTurnsExceeded
}

// bridgeToolExecutor is the real ToolExecutor the ExecuteTool handler also uses.
// It drives a stdio plugin via Bridge.Send + a capturing SessionSink (matching
// the JSON-RPC response id), then applies smartFilterText to the result text. It
// is exercised by the admin ExecuteTool test through the FAKE process (canned
// result frame) — NO real spawn in any unit test.
type bridgeToolExecutor struct {
	bridge *Bridge
	mu     sync.Mutex
	nextID int
}

// newBridgeToolExecutor builds a real executor over a live Bridge.
func newBridgeToolExecutor(b *Bridge) *bridgeToolExecutor {
	return &bridgeToolExecutor{bridge: b, nextID: 100}
}

// jsonRPCResult is the minimal JSON-RPC frame shape the executor matches on: the
// id correlates the request, and result.content[].text carries the tool output.
type jsonRPCResult struct {
	ID     int `json:"id"`
	Result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Execute sends a tools/call frame to the plugin and waits for the matching
// result frame, returning its filtered text. It registers a temporary session
// sink keyed by the request id and removes it on return.
func (e *bridgeToolExecutor) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	if e.bridge == nil || !e.bridge.IsRunning() {
		return "", errors.New("mcp: plugin bridge not running")
	}

	e.mu.Lock()
	e.nextID++
	id := e.nextID
	e.mu.Unlock()

	if args == nil {
		args = map[string]any{}
	}
	frame, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	})
	if err != nil {
		return "", fmt.Errorf("marshal tools/call: %w", err)
	}

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	sid := fmt.Sprintf("exec-%d", id)
	e.bridge.AddSession(sid, func(raw []byte) error {
		var r jsonRPCResult
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil // not our frame; keep the session alive
		}
		if r.ID != id {
			return nil
		}
		if r.Error != nil {
			select {
			case errCh <- fmt.Errorf("mcp tool error: %s", r.Error.Message):
			default:
			}
			return nil
		}
		var text string
		for _, c := range r.Result.Content {
			text += c.Text
		}
		select {
		case resultCh <- text:
		default:
		}
		return nil
	})
	defer e.bridge.RemoveSession(sid)

	if err := e.bridge.Send(frame); err != nil {
		return "", fmt.Errorf("send tools/call: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, toolCallTimeout)
	defer cancel()
	select {
	case text := <-resultCh:
		return smartFilterText(text), nil
	case err := <-errCh:
		return "", err
	case <-callCtx.Done():
		return "", fmt.Errorf("mcp tools/call %q: %w", name, callCtx.Err())
	}
}
