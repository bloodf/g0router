package store

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
)

func TestMCPClientCreateAndGetRoundTripsConfig(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{
		Name:      "docs",
		Transport: mcp.TransportStdio,
		Command:   strPtr("mcp-docs"),
		Args:      []string{"--stdio"},
		Env:       map[string]string{"TOKEN": "secret"},
		IsActive:  true,
	}

	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	if client.ID == "" {
		t.Fatal("ID should be set after create")
	}
	if client.CreatedAt == "" {
		t.Fatal("CreatedAt should be set after create")
	}

	got, err := s.GetMCPClient(client.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	if got.Name != "docs" || got.Transport != mcp.TransportStdio {
		t.Fatalf("got = %+v, want docs/stdio", got)
	}
	if got.Command == nil || *got.Command != "mcp-docs" {
		t.Fatalf("command = %v, want mcp-docs", got.Command)
	}
	if len(got.Args) != 1 || got.Args[0] != "--stdio" {
		t.Fatalf("args = %+v, want --stdio", got.Args)
	}
	if got.Env["TOKEN"] != "secret" {
		t.Fatalf("env = %+v, want TOKEN", got.Env)
	}
	if !got.IsActive {
		t.Fatal("client should be active")
	}
}

func TestMCPClientListOrdersByCreation(t *testing.T) {
	s := openTestStore(t)
	for _, name := range []string{"docs", "files"} {
		if err := s.CreateMCPClient(&MCPClient{Name: name, Transport: mcp.TransportSSE, URL: strPtr("http://example.test"), IsActive: true}); err != nil {
			t.Fatalf("CreateMCPClient %s: %v", name, err)
		}
	}

	clients, err := s.ListMCPClients()
	if err != nil {
		t.Fatalf("ListMCPClients: %v", err)
	}
	if len(clients) != 2 {
		t.Fatalf("len = %d, want 2", len(clients))
	}
	if clients[0].Name != "docs" || clients[1].Name != "files" {
		t.Fatalf("unexpected order: %+v", clients)
	}
}

func TestMCPClientUpdateManifestAndHealth(t *testing.T) {
	s := openTestStore(t)
	client := &MCPClient{Name: "docs", Transport: mcp.TransportStdio, Command: strPtr("mcp-docs"), IsActive: true}
	if err := s.CreateMCPClient(client); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	manifest := mcp.Manifest{
		ClientID: client.ID,
		Tools: []mcp.Tool{{
			ClientID:    client.ID,
			Name:        "search",
			Description: "Search docs",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		}},
	}

	if err := s.UpdateMCPClientManifest(client.ID, manifest); err != nil {
		t.Fatalf("UpdateMCPClientManifest: %v", err)
	}
	if err := s.UpdateMCPClientHealth(client.ID, "healthy"); err != nil {
		t.Fatalf("UpdateMCPClientHealth: %v", err)
	}

	got, err := s.GetMCPClient(client.ID)
	if err != nil {
		t.Fatalf("GetMCPClient: %v", err)
	}
	if got.ToolManifest == nil || len(got.ToolManifest.Tools) != 1 {
		t.Fatalf("manifest = %+v, want one tool", got.ToolManifest)
	}
	if got.ToolManifest.Tools[0].InputSchema == nil {
		t.Fatal("tool schema should be cached")
	}
	if got.ManifestUpdatedAt == nil || *got.ManifestUpdatedAt == "" {
		t.Fatalf("manifest_updated_at = %v, want timestamp", got.ManifestUpdatedAt)
	}
	if got.HealthStatus != "healthy" {
		t.Fatalf("health = %q, want healthy", got.HealthStatus)
	}
	if got.LastHealthCheck == nil || *got.LastHealthCheck == "" {
		t.Fatalf("last_health_check = %v, want timestamp", got.LastHealthCheck)
	}
}

func TestMCPClientDeleteNotFound(t *testing.T) {
	s := openTestStore(t)

	err := s.DeleteMCPClient("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
