package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestBuildCompactManifestOmitsInputSchemas(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)

	compact, err := BuildCompactManifest(Manifest{
		ClientID: "docs",
		Tools: []Tool{
			{Name: "search", Description: "Search docs", InputSchema: schema},
			{Name: "read", Description: "Read docs", InputSchema: []byte(`{"type":"object"}`)},
		},
	})
	if err != nil {
		t.Fatalf("BuildCompactManifest: %v", err)
	}

	if compact.ClientID != "docs" {
		t.Fatalf("ClientID = %q, want docs", compact.ClientID)
	}
	if len(compact.Tools) != 2 {
		t.Fatalf("Tools len = %d, want 2", len(compact.Tools))
	}
	if compact.Tools[0].Type != "function" {
		t.Fatalf("tool type = %q, want function", compact.Tools[0].Type)
	}
	if compact.Tools[0].Function.Name != "docs__search" {
		t.Fatalf("tool name = %q, want docs__search", compact.Tools[0].Function.Name)
	}
	if compact.Tools[0].Function.Description != "Search docs" {
		t.Fatalf("description = %q, want Search docs", compact.Tools[0].Function.Description)
	}
	if len(compact.Tools[0].Function.Parameters) != 0 {
		t.Fatalf("compact manifest included schema: %s", compact.Tools[0].Function.Parameters)
	}
}

func TestDiscoveryRegistersClientAndReturnsCompactManifest(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`)
	client := &fakeClient{tools: []Tool{
		{Name: "search", Description: "Search docs", InputSchema: schema},
	}}
	clientManager := NewClientManager(fakeConnector{client: client})
	toolManager := NewToolManager()
	discovery := NewDiscovery(clientManager, toolManager)

	compact, err := discovery.Discover(context.Background(), ClientConfig{
		ID:        "docs",
		Name:      "Docs",
		Transport: TransportStdio,
		Command:   "mcp-docs",
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if len(compact.Tools) != 1 {
		t.Fatalf("compact tools len = %d, want 1", len(compact.Tools))
	}
	if compact.Tools[0].Function.Name != "docs__search" {
		t.Fatalf("compact tool name = %q, want docs__search", compact.Tools[0].Function.Name)
	}
	if len(compact.Tools[0].Function.Parameters) != 0 {
		t.Fatalf("compact tool included schema: %s", compact.Tools[0].Function.Parameters)
	}

	full, err := toolManager.Lookup("docs__search")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if string(full.InputSchema) != string(schema) {
		t.Fatalf("full schema = %s, want %s", full.InputSchema, schema)
	}

	if _, err := toolManager.Call(context.Background(), "docs__search", json.RawMessage(`{"query":"mcp"}`)); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if len(client.calls) != 1 || client.calls[0].Name != "search" {
		t.Fatalf("client calls = %#v", client.calls)
	}
}

func TestDiscoveryWrapsClientRegistrationErrors(t *testing.T) {
	connectErr := errors.New("spawn failed")
	discovery := NewDiscovery(NewClientManager(fakeConnector{err: connectErr}), NewToolManager())

	_, err := discovery.Discover(context.Background(), ClientConfig{
		ID:        "docs",
		Name:      "Docs",
		Transport: TransportStdio,
		Command:   "mcp-docs",
	})
	if !errors.Is(err, connectErr) {
		t.Fatalf("expected wrapped connect error, got %v", err)
	}
}

func TestDiscoveryClosesClientWhenToolRegistrationFails(t *testing.T) {
	registeredClient := &fakeClient{tools: []Tool{{Name: "search", Description: "Search docs"}}}
	toolManager := NewToolManager()
	if err := toolManager.RegisterManifest(Manifest{
		ClientID: "docs",
		Tools:    []Tool{{Name: "search", Description: "Search docs"}},
	}); err != nil {
		t.Fatalf("RegisterManifest: %v", err)
	}

	discovery := NewDiscovery(NewClientManager(fakeConnector{client: registeredClient}), toolManager)
	_, err := discovery.Discover(context.Background(), ClientConfig{
		ID:        "docs",
		Name:      "Docs",
		Transport: TransportStdio,
		Command:   "mcp-docs",
	})
	if !errors.Is(err, ErrToolAlreadyRegistered) {
		t.Fatalf("expected ErrToolAlreadyRegistered, got %v", err)
	}
	if !registeredClient.closed {
		t.Fatal("client was not closed after tool registration failure")
	}
}

func TestBuildCompactManifestRejectsInvalidManifest(t *testing.T) {
	_, err := BuildCompactManifest(Manifest{Tools: []Tool{{Name: "search"}}})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected ErrInvalidManifest for missing client ID, got %v", err)
	}

	_, err = BuildCompactManifest(Manifest{ClientID: "docs", Tools: []Tool{{Description: "missing name"}}})
	if !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("expected ErrInvalidManifest for missing tool name, got %v", err)
	}
}
