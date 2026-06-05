package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

func TestToolManagerRegistersAndReturnsCompactTools(t *testing.T) {
	manager := NewToolManager()
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)

	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools: []Tool{
			{Name: "search", Description: "Search docs", InputSchema: schema},
			{Name: "read", Description: "Read doc", InputSchema: []byte(`{"type":"object"}`)},
		},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	tools := manager.CompactTools()
	if len(tools) != 2 {
		t.Fatalf("CompactTools len = %d, want 2", len(tools))
	}
	if tools[0].Type != "function" || tools[0].Function.Name != "docs__search" {
		t.Fatalf("first compact tool = %#v", tools[0])
	}
	if tools[0].Function.Description != "Search docs" {
		t.Fatalf("description = %q, want Search docs", tools[0].Function.Description)
	}
	if len(tools[0].Function.Parameters) != 0 {
		t.Fatalf("compact tool included schema: %s", tools[0].Function.Parameters)
	}
}

func TestToolManagerLookupReturnsFullTool(t *testing.T) {
	manager := NewToolManager()
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs", InputSchema: schema}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	tool, err := manager.Lookup("docs__search")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if tool.ClientID != "docs" || tool.Name != "search" {
		t.Fatalf("tool = %#v", tool)
	}
	if string(tool.InputSchema) != string(schema) {
		t.Fatalf("schema = %s, want %s", tool.InputSchema, schema)
	}
}

func TestToolManagerRejectsDuplicateRegisteredToolName(t *testing.T) {
	manager := NewToolManager()
	manifest := Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}

	if err := manager.RegisterManifest(manifest); err != nil {
		t.Fatalf("RegisterManifest first: %v", err)
	}
	err := manager.RegisterManifest(manifest)
	if !errors.Is(err, ErrToolAlreadyRegistered) {
		t.Fatalf("expected ErrToolAlreadyRegistered, got %v", err)
	}
}

func TestToolManagerRejectsInvalidManifest(t *testing.T) {
	manager := NewToolManager()

	err := manager.RegisterManifest(Manifest{Tools: []Tool{{Name: "search"}}})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected ErrInvalidManifest for missing client ID, got %v", err)
	}

	err = manager.RegisterManifest(Manifest{ClientID: "docs", Tools: []Tool{{Description: "missing name"}}})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected ErrInvalidManifest for missing tool name, got %v", err)
	}
}

func TestToolManagerLookupUnknownTool(t *testing.T) {
	manager := NewToolManager()

	_, err := manager.Lookup("docs__missing")
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("expected ErrToolNotFound, got %v", err)
	}
}

func TestToolManagerCallRoutesToClient(t *testing.T) {
	client := &fakeClient{callResult: CallResult{Content: "found"}}
	manager := NewToolManager()
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	manager.RegisterClient("docs", client)

	result, err := manager.Call(context.Background(), "docs__search", json.RawMessage(`{"query":"mcp"}`))
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if result.Content != "found" {
		t.Fatalf("result = %#v", result)
	}
	if len(client.calls) != 1 || client.calls[0].Name != "search" {
		t.Fatalf("client calls = %#v", client.calls)
	}
}

func TestToolManagerUnregisterClientRemovesOnlyThatClientsTools(t *testing.T) {
	manager := NewToolManager()
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools: []Tool{
			{Name: "search", Description: "Search docs"},
			{Name: "read", Description: "Read docs"},
		},
	}); err != nil {
		t.Fatalf("RegisterManifest docs: %v", err)
	}
	if err := manager.RegisterManifest(Manifest{
		ClientID: "repo",
		Tools:    []Tool{{Name: "search", Description: "Search repos"}},
	}); err != nil {
		t.Fatalf("RegisterManifest repo: %v", err)
	}
	manager.RegisterClient("docs", &fakeClient{callResult: CallResult{Content: "docs"}})
	manager.RegisterClient("repo", &fakeClient{callResult: CallResult{Content: "repo"}})

	manager.UnregisterClient("docs")

	tools := manager.CompactTools()
	if len(tools) != 1 || tools[0].Function.Name != "repo__search" {
		t.Fatalf("tools = %#v, want only repo__search", tools)
	}
	if _, err := manager.Call(context.Background(), "docs__search", json.RawMessage(`{}`)); !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("docs tool error = %v, want ErrToolNotFound", err)
	}
	if _, err := manager.Call(context.Background(), "repo__search", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("repo tool should remain callable: %v", err)
	}
}

func TestToolManagerValidatesArgumentsBeforeDispatch(t *testing.T) {
	client := &fakeClient{callResult: CallResult{Content: "found"}}
	manager := NewToolManager()
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools: []Tool{{
			Name:        "search",
			Description: "Search docs",
			InputSchema: json.RawMessage(`{
				"type":"object",
				"required":["query"],
				"properties":{"query":{"type":"string"},"limit":{"type":"integer"}},
				"additionalProperties":false
			}`),
		}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	manager.RegisterClient("docs", client)

	_, err := manager.Call(context.Background(), "docs__search", json.RawMessage(`{"query":7}`))
	if !errors.Is(err, ErrInvalidToolArguments) {
		t.Fatalf("expected ErrInvalidToolArguments, got %v", err)
	}
	if len(client.calls) != 0 {
		t.Fatalf("client was called with invalid arguments: %#v", client.calls)
	}

	result, err := manager.Call(context.Background(), "docs__search", json.RawMessage(`{"query":"mcp","limit":2}`))
	if err != nil {
		t.Fatalf("Call valid arguments: %v", err)
	}
	if result.Content != "found" {
		t.Fatalf("result = %#v", result)
	}
	if len(client.calls) != 1 || string(client.calls[0].Arguments) != `{"query":"mcp","limit":2}` {
		t.Fatalf("client calls = %#v", client.calls)
	}
}

func TestToolManagerRequestContextFiltersVisibleAndCallableTools(t *testing.T) {
	client := &fakeClient{callResult: CallResult{Content: "found"}}
	manager := NewToolManager()
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools: []Tool{
			{Name: "search", Description: "Search docs"},
			{Name: "read", Description: "Read docs"},
		},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}
	manager.RegisterClient("docs", client)
	ctx := WithAllowedTools(context.Background(), "docs__search")

	tools := manager.CompactToolsForRequest(ctx)
	if len(tools) != 1 || tools[0].Function.Name != "docs__search" {
		t.Fatalf("filtered tools = %#v, want only docs__search", tools)
	}

	_, err := manager.Call(ctx, "docs__read", json.RawMessage(`{}`))
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("expected ErrToolNotFound for filtered tool, got %v", err)
	}
	if len(client.calls) != 0 {
		t.Fatalf("client was called for filtered tool: %#v", client.calls)
	}

	if _, err := manager.Call(ctx, "docs__search", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("Call allowed tool: %v", err)
	}
	if len(client.calls) != 1 || client.calls[0].Name != "search" {
		t.Fatalf("client calls = %#v", client.calls)
	}
}

func TestToolManagerConcurrentRegistrationListAndCalls(t *testing.T) {
	manager := NewToolManager()
	errs := make(chan error, 100)
	var wg sync.WaitGroup

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			clientID := fmt.Sprintf("docs-%02d", i)
			if err := manager.RegisterManifest(Manifest{
				ClientID: clientID,
				Tools:    []Tool{{Name: "search", Description: "Search docs"}},
			}); err != nil {
				errs <- fmt.Errorf("RegisterManifest %s: %w", clientID, err)
				return
			}
			manager.RegisterClient(clientID, &concurrentCallClient{})
			if _, err := manager.Call(context.Background(), toolFullName(clientID, "search"), json.RawMessage(`{}`)); err != nil {
				errs <- fmt.Errorf("Call %s: %w", clientID, err)
			}
		}(i)
	}
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.CompactTools()
			_, _ = manager.Lookup("missing__tool")
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	tools := manager.CompactTools()
	if len(tools) != 25 {
		t.Fatalf("tools len = %d, want 25", len(tools))
	}
}

func TestToolManagerCallUnknownClient(t *testing.T) {
	manager := NewToolManager()
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	_, err := manager.Call(context.Background(), "docs__search", nil)
	if !errors.Is(err, ErrClientNotFound) {
		t.Fatalf("expected ErrClientNotFound, got %v", err)
	}
}

func TestToolManagerOpenAIToolNameEscapesSeparator(t *testing.T) {
	manager := NewToolManager()
	if err := manager.RegisterManifest(Manifest{
		ClientID: "docs_api",
		Tools:    []Tool{{Name: "deep_search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	tool, err := manager.Lookup("docs_api__deep_search")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if tool.Name != "deep_search" {
		t.Fatalf("tool name = %q, want deep_search", tool.Name)
	}
}

func TestToolManagerCompactToolsUseProviderTypes(t *testing.T) {
	var _ []providers.Tool = NewToolManager().CompactTools()
}

// closeTrackingClient panics if CallTool runs after Close, surfacing any
// use-after-close under the race detector.
type closeTrackingClient struct {
	mu     sync.Mutex
	closed bool
}

func (c *closeTrackingClient) ListTools(ctx context.Context) ([]Tool, error) { return nil, nil }

func (c *closeTrackingClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		panic("CallTool invoked after Close")
	}
	time.Sleep(time.Millisecond)
	return CallResult{Content: "ok"}, nil
}

func (c *closeTrackingClient) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return nil
}

func TestToolManagerCallVsUnregisterRace(t *testing.T) {
	for iter := 0; iter < 50; iter++ {
		manager := NewToolManager()
		if err := manager.RegisterManifest(Manifest{
			ClientID: "docs",
			Tools:    []Tool{{Name: "search", Description: "Search docs"}},
		}); err != nil {
			t.Fatalf("RegisterManifest: %v", err)
		}
		client := &closeTrackingClient{}
		manager.RegisterClient("docs", client)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = manager.Call(context.Background(), "docs__search", json.RawMessage(`{}`))
		}()
		go func() {
			defer wg.Done()
			manager.UnregisterClient("docs")
			// Safe to close only after UnregisterClient drained in-flight calls.
			_ = client.Close()
		}()
		wg.Wait()
	}
}

type concurrentCallClient struct{}

func (c *concurrentCallClient) ListTools(ctx context.Context) ([]Tool, error) {
	return nil, nil
}

func (c *concurrentCallClient) CallTool(ctx context.Context, req CallRequest) (CallResult, error) {
	return CallResult{Content: "ok"}, nil
}

func (c *concurrentCallClient) Close() error {
	return nil
}
