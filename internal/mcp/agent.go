package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bloodf/g0router/internal/providers"
)

const defaultAgentMaxTurns = 8

type Agent struct {
	provider providers.Provider
	key      providers.Key
	tools    *ToolManager
	maxTurns int
}

func NewAgent(provider providers.Provider, key providers.Key, tools *ToolManager) *Agent {
	return &Agent{
		provider: provider,
		key:      key,
		tools:    tools,
		maxTurns: defaultAgentMaxTurns,
	}
}

func (a *Agent) Run(ctx context.Context, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	nextReq := cloneChatRequest(req)
	if a.tools != nil && len(nextReq.Tools) == 0 {
		nextReq.Tools = a.tools.CompactToolsForRequest(ctx)
	}

	for turn := 0; turn < a.maxTurns; turn++ {
		resp, err := a.provider.ChatCompletion(ctx, a.key, nextReq)
		if err != nil {
			return nil, fmt.Errorf("agent chat completion: %w", err)
		}

		toolCalls := firstChoiceToolCalls(resp)
		if len(toolCalls) == 0 {
			return resp, nil
		}

		nextReq.Messages = append(nextReq.Messages, resp.Choices[0].Message)
		for _, toolCall := range toolCalls {
			result, err := a.tools.Call(ctx, toolCall.Function.Name, json.RawMessage(toolCall.Function.Arguments))
			if err != nil {
				return nil, fmt.Errorf("agent tool call %q: %w", toolCall.Function.Name, err)
			}
			nextReq.Messages = append(nextReq.Messages, providers.Message{
				Role:       "tool",
				Content:    toolResultContent(result.Content),
				ToolCallID: &toolCall.ID,
			})
		}
	}

	return nil, fmt.Errorf("agent exceeded max turns %d", a.maxTurns)
}

func cloneChatRequest(req *providers.ChatRequest) *providers.ChatRequest {
	cloned := *req
	cloned.Messages = append([]providers.Message(nil), req.Messages...)
	cloned.Tools = append([]providers.Tool(nil), req.Tools...)
	return &cloned
}

func firstChoiceToolCalls(resp *providers.ChatResponse) []providers.ToolCall {
	if resp == nil || len(resp.Choices) == 0 {
		return nil
	}
	return resp.Choices[0].Message.ToolCalls
}

func toolResultContent(content any) any {
	switch value := content.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return value
	}
}
