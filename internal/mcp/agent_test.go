package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/providers"
)

type fakeAgentProvider struct {
	responses []*providers.ChatResponse
	requests  []*providers.ChatRequest
	err       error
}

func (f *fakeAgentProvider) Name() providers.ModelProvider {
	return providers.ProviderOpenAI
}

func (f *fakeAgentProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.responses) == 0 {
		return &providers.ChatResponse{}, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

func (f *fakeAgentProvider) ChatCompletionStream(ctx context.Context, key providers.Key, req *providers.ChatRequest) (<-chan providers.StreamChunk, error) {
	return nil, errors.New("stream not used")
}

func (f *fakeAgentProvider) ListModels(ctx context.Context, key providers.Key) ([]providers.Model, error) {
	return nil, nil
}

func TestAgentRunsToolCallLoop(t *testing.T) {
	toolCallID := "call-1"
	provider := &fakeAgentProvider{responses: []*providers.ChatResponse{
		{
			Choices: []providers.Choice{
				{
					Message: providers.Message{
						Role:    "assistant",
						Content: nil,
						ToolCalls: []providers.ToolCall{
							{
								ID:   toolCallID,
								Type: "function",
								Function: providers.ToolCallFunc{
									Name:      "docs__search",
									Arguments: `{"query":"mcp"}`,
								},
							},
						},
					},
				},
			},
		},
		{
			Choices: []providers.Choice{
				{Message: providers.Message{Role: "assistant", Content: "found it"}},
			},
		},
	}}
	client := &fakeClient{callResult: CallResult{Content: "doc result"}}
	tools := NewToolManager()
	if err := tools.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	tools.RegisterClient("docs", client)

	agent := NewAgent(provider, providers.Key{Provider: providers.ProviderOpenAI}, tools)
	resp, err := agent.Run(context.Background(), &providers.ChatRequest{
		Model:    "gpt-test",
		Messages: []providers.Message{{Role: "user", Content: "find mcp docs"}},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if resp.Choices[0].Message.Content != "found it" {
		t.Fatalf("final response = %#v", resp)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider requests = %d, want 2", len(provider.requests))
	}
	if len(provider.requests[0].Tools) != 1 || provider.requests[0].Tools[0].Function.Name != "docs__search" {
		t.Fatalf("first request tools = %#v", provider.requests[0].Tools)
	}
	secondMessages := provider.requests[1].Messages
	if len(secondMessages) != 3 {
		t.Fatalf("second request messages = %#v", secondMessages)
	}
	if len(secondMessages[1].ToolCalls) != 1 || secondMessages[1].ToolCalls[0].ID != toolCallID {
		t.Fatalf("assistant tool-call message = %#v", secondMessages[1])
	}
	if secondMessages[2].Role != "tool" || secondMessages[2].ToolCallID == nil || *secondMessages[2].ToolCallID != toolCallID {
		t.Fatalf("tool result message = %#v", secondMessages[2])
	}
	if secondMessages[2].Content != "doc result" {
		t.Fatalf("tool result content = %#v", secondMessages[2].Content)
	}
	if len(client.calls) != 1 || client.calls[0].Name != "search" {
		t.Fatalf("client calls = %#v", client.calls)
	}
}

func TestAgentWrapsProviderAndToolErrors(t *testing.T) {
	providerErr := errors.New("provider down")
	agent := NewAgent(&fakeAgentProvider{err: providerErr}, providers.Key{}, NewToolManager())

	_, err := agent.Run(context.Background(), &providers.ChatRequest{Model: "gpt-test"})
	if !errors.Is(err, providerErr) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}

	tools := NewToolManager()
	if err := tools.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	agent = NewAgent(&fakeAgentProvider{responses: []*providers.ChatResponse{
		{
			Choices: []providers.Choice{
				{
					Message: providers.Message{
						Role: "assistant",
						ToolCalls: []providers.ToolCall{
							{ID: "call-1", Type: "function", Function: providers.ToolCallFunc{Name: "docs__search", Arguments: `{}`}},
						},
					},
				},
			},
		},
	}}, providers.Key{}, tools)

	_, err = agent.Run(context.Background(), &providers.ChatRequest{Model: "gpt-test"})
	if !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("expected wrapped tool error, got %v", err)
	}
}

func TestAgentCopiesRequestMessages(t *testing.T) {
	provider := &fakeAgentProvider{responses: []*providers.ChatResponse{
		{Choices: []providers.Choice{{Message: providers.Message{Role: "assistant", Content: "done"}}}},
	}}
	agent := NewAgent(provider, providers.Key{}, NewToolManager())
	req := &providers.ChatRequest{
		Model:    "gpt-test",
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	}

	if _, err := agent.Run(context.Background(), req); err != nil {
		t.Fatalf("Run: %v", err)
	}

	provider.requests[0].Messages = append(provider.requests[0].Messages, providers.Message{Role: "user", Content: "mutated"})
	if len(req.Messages) != 1 {
		t.Fatalf("original request messages were mutated: %#v", req.Messages)
	}
}

func TestAgentPassesRawToolArguments(t *testing.T) {
	provider := &fakeAgentProvider{responses: []*providers.ChatResponse{
		{
			Choices: []providers.Choice{
				{
					Message: providers.Message{
						Role: "assistant",
						ToolCalls: []providers.ToolCall{
							{
								ID:   "call-1",
								Type: "function",
								Function: providers.ToolCallFunc{
									Name:      "docs__search",
									Arguments: `{"query":"mcp"}`,
								},
							},
						},
					},
				},
			},
		},
		{Choices: []providers.Choice{{Message: providers.Message{Role: "assistant", Content: "done"}}}},
	}}
	client := &fakeClient{callResult: CallResult{Content: "ok"}}
	tools := NewToolManager()
	if err := tools.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	tools.RegisterClient("docs", client)

	agent := NewAgent(provider, providers.Key{}, tools)
	if _, err := agent.Run(context.Background(), &providers.ChatRequest{Model: "gpt-test"}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !json.Valid(client.calls[0].Arguments) || string(client.calls[0].Arguments) != `{"query":"mcp"}` {
		t.Fatalf("arguments = %s", client.calls[0].Arguments)
	}
}

func TestAgentFiltersVisibleToolsFromRequestContext(t *testing.T) {
	provider := &fakeAgentProvider{responses: []*providers.ChatResponse{
		{Choices: []providers.Choice{{Message: providers.Message{Role: "assistant", Content: "done"}}}},
	}}
	tools := NewToolManager()
	if err := tools.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools: []Tool{
			{Name: "search", Description: "Search docs"},
			{Name: "read", Description: "Read docs"},
		},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	agent := NewAgent(provider, providers.Key{}, tools)
	_, err := agent.Run(WithAllowedTools(context.Background(), "docs__search"), &providers.ChatRequest{Model: "gpt-test"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(provider.requests) != 1 {
		t.Fatalf("provider requests = %d, want 1", len(provider.requests))
	}
	if len(provider.requests[0].Tools) != 1 || provider.requests[0].Tools[0].Function.Name != "docs__search" {
		t.Fatalf("request tools = %#v, want only docs__search", provider.requests[0].Tools)
	}
}
