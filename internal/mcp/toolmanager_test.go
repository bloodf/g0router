package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
