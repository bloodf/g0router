package mcp

import (
	"context"
	"errors"
	"testing"
)

// fakeToolExecutor is a deterministic ToolExecutor for the agent-loop unit tests.
// It records the calls it receives and returns canned results/errors — NO real
// process, NO real network.
type fakeToolExecutor struct {
	calls   []string
	result  string
	err     error
}

func (f *fakeToolExecutor) Execute(_ context.Context, name string, _ map[string]any) (string, error) {
	f.calls = append(f.calls, name)
	if f.err != nil {
		return "", f.err
	}
	return f.result, nil
}

// TestAgentTerminatesOnFinalAnswer: when the model returns a final answer with no
// tool call, the loop stops immediately and surfaces that answer.
func TestAgentTerminatesOnFinalAnswer(t *testing.T) {
	exec := &fakeToolExecutor{}
	a := NewAgent(exec, 8)

	step := func(_ context.Context, _ []AgentMessage) (ModelDecision, error) {
		return ModelDecision{Final: "all done"}, nil
	}

	res, err := a.Run(context.Background(), AgentRequest{Prompt: "hi"}, step)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Answer != "all done" {
		t.Fatalf("res.Answer = %q, want %q", res.Answer, "all done")
	}
	if len(exec.calls) != 0 {
		t.Fatalf("tool executions = %d, want 0", len(exec.calls))
	}
	if res.Turns != 1 {
		t.Fatalf("res.Turns = %d, want 1", res.Turns)
	}
}

// TestAgentExecutesToolThenFeedsBack: the model first requests a tool call, the
// agent executes it via the ToolExecutor and re-invokes the model with the result
// appended, then the model returns a final answer.
func TestAgentExecutesToolThenFeedsBack(t *testing.T) {
	exec := &fakeToolExecutor{result: "file contents"}
	a := NewAgent(exec, 8)

	var sawResult string
	calls := 0
	step := func(_ context.Context, history []AgentMessage) (ModelDecision, error) {
		calls++
		if calls == 1 {
			return ModelDecision{ToolName: "read_file", ToolArgs: map[string]any{"path": "/x"}}, nil
		}
		// Second invocation must see the tool result appended to history.
		for _, m := range history {
			if m.Role == "tool" {
				sawResult = m.Content
			}
		}
		return ModelDecision{Final: "answered"}, nil
	}

	res, err := a.Run(context.Background(), AgentRequest{Prompt: "read it"}, step)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(exec.calls) != 1 || exec.calls[0] != "read_file" {
		t.Fatalf("tool calls = %v, want [read_file]", exec.calls)
	}
	if sawResult != "file contents" {
		t.Fatalf("model did not see tool result fed back, saw %q", sawResult)
	}
	if res.Answer != "answered" {
		t.Fatalf("res.Answer = %q", res.Answer)
	}
}

// TestAgentStopsAtMaxTurns: a model that always requests a tool call must be
// stopped at the hard maxTurns cap — no runaway loop (PAR-MCP-040 bound).
func TestAgentStopsAtMaxTurns(t *testing.T) {
	exec := &fakeToolExecutor{result: "again"}
	a := NewAgent(exec, 3)

	step := func(_ context.Context, _ []AgentMessage) (ModelDecision, error) {
		return ModelDecision{ToolName: "loop_tool", ToolArgs: nil}, nil
	}

	res, err := a.Run(context.Background(), AgentRequest{Prompt: "go"}, step)
	if !errors.Is(err, ErrMaxTurnsExceeded) {
		t.Fatalf("Run err = %v, want ErrMaxTurnsExceeded", err)
	}
	if len(exec.calls) > 3 {
		t.Fatalf("tool executions = %d, want <= maxTurns (3)", len(exec.calls))
	}
	if res.Turns > 3 {
		t.Fatalf("res.Turns = %d, want <= 3", res.Turns)
	}
}

// TestAgentSurfacesToolError: a tool execution error is surfaced (not swallowed
// into an infinite retry).
func TestAgentSurfacesToolError(t *testing.T) {
	wantErr := errors.New("tool boom")
	exec := &fakeToolExecutor{err: wantErr}
	a := NewAgent(exec, 8)

	step := func(_ context.Context, _ []AgentMessage) (ModelDecision, error) {
		return ModelDecision{ToolName: "broken", ToolArgs: nil}, nil
	}

	_, err := a.Run(context.Background(), AgentRequest{Prompt: "go"}, step)
	if err == nil {
		t.Fatalf("Run err = nil, want a surfaced tool error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run err = %v, want wrapped %v", err, wantErr)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("tool executions = %d, want exactly 1 (no retry loop)", len(exec.calls))
	}
}

// TestNewAgentDefaultMaxTurns: a non-positive maxTurns falls back to the default
// bound so the loop is always bounded.
func TestNewAgentDefaultMaxTurns(t *testing.T) {
	a := NewAgent(&fakeToolExecutor{}, 0)
	if a.maxTurns != defaultAgentMaxTurns {
		t.Fatalf("maxTurns = %d, want default %d", a.maxTurns, defaultAgentMaxTurns)
	}
}
