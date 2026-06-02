package store

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
)

func TestMCPInstanceCreateAllowsSameServerKeyWithDifferentNames(t *testing.T) {
	s := openTestStore(t)
	first := &MCPInstance{
		Name:       "linear-work",
		ServerKey:  "linear",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        strPtr("https://mcp.linear.app/mcp"),
		IsActive:   true,
	}
	second := &MCPInstance{
		Name:       "linear-personal",
		ServerKey:  "linear",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        strPtr("https://mcp.linear.app/mcp"),
		IsActive:   true,
	}

	if err := s.CreateMCPInstance(first); err != nil {
		t.Fatalf("CreateMCPInstance first: %v", err)
	}
	if err := s.CreateMCPInstance(second); err != nil {
		t.Fatalf("CreateMCPInstance second: %v", err)
	}

	instances, err := s.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("len = %d, want 2", len(instances))
	}
}

func TestMCPInstanceRejectsDuplicateName(t *testing.T) {
	s := openTestStore(t)
	instance := &MCPInstance{Name: "linear", ServerKey: "linear", LaunchType: mcp.LaunchHTTP, Transport: mcp.TransportStreamableHTTP, URL: strPtr("https://mcp.linear.app/mcp")}

	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance first: %v", err)
	}
	err := s.CreateMCPInstance(&MCPInstance{Name: "linear", ServerKey: "linear-copy", LaunchType: mcp.LaunchHTTP, Transport: mcp.TransportStreamableHTTP, URL: strPtr("https://mcp.example/mcp")})
	if err == nil {
		t.Fatal("duplicate name should fail")
	}
}

func TestMCPInstanceManifestBelongsToOneInstance(t *testing.T) {
	s := openTestStore(t)
	first := &MCPInstance{Name: "docs-a", ServerKey: "docs", LaunchType: mcp.LaunchCommand, Transport: mcp.TransportStdio, Command: strPtr("mcp-docs")}
	second := &MCPInstance{Name: "docs-b", ServerKey: "docs", LaunchType: mcp.LaunchCommand, Transport: mcp.TransportStdio, Command: strPtr("mcp-docs")}
	if err := s.CreateMCPInstance(first); err != nil {
		t.Fatalf("CreateMCPInstance first: %v", err)
	}
	if err := s.CreateMCPInstance(second); err != nil {
		t.Fatalf("CreateMCPInstance second: %v", err)
	}

	manifest := mcp.Manifest{
		ClientID: first.ID,
		Tools: []mcp.Tool{{
			ClientID:    first.ID,
			Name:        "search",
			Description: "Search docs",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		}},
	}
	if err := s.UpdateMCPInstanceManifest(first.ID, manifest); err != nil {
		t.Fatalf("UpdateMCPInstanceManifest: %v", err)
	}

	gotFirst, err := s.GetMCPInstance(first.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance first: %v", err)
	}
	gotSecond, err := s.GetMCPInstance(second.ID)
	if err != nil {
		t.Fatalf("GetMCPInstance second: %v", err)
	}
	if gotFirst.ToolManifest == nil || len(gotFirst.ToolManifest.Tools) != 1 {
		t.Fatalf("first manifest = %+v, want one tool", gotFirst.ToolManifest)
	}
	if gotSecond.ToolManifest != nil {
		t.Fatalf("second manifest = %+v, want nil", gotSecond.ToolManifest)
	}
}

func TestMCPInstanceListRedactsEnvAndHeaders(t *testing.T) {
	s := openTestStore(t)
	instance := &MCPInstance{
		Name:       "secure",
		ServerKey:  "secure",
		LaunchType: mcp.LaunchHTTP,
		Transport:  mcp.TransportStreamableHTTP,
		URL:        strPtr("https://mcp.example/mcp"),
		Env:        map[string]string{"TOKEN": "secret", "MODE": "readonly"},
		Headers:    map[string]string{"Authorization": "Bearer token", "X-Mode": "readonly"},
	}
	if err := s.CreateMCPInstance(instance); err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	instances, err := s.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if instances[0].Env["TOKEN"] != mcp.RedactedValue {
		t.Fatalf("env TOKEN = %q, want redacted", instances[0].Env["TOKEN"])
	}
	if instances[0].Env["MODE"] != "readonly" {
		t.Fatalf("env MODE = %q, want readonly", instances[0].Env["MODE"])
	}
	if instances[0].Headers["Authorization"] != mcp.RedactedValue {
		t.Fatalf("Authorization = %q, want redacted", instances[0].Headers["Authorization"])
	}
}

func TestMCPInstanceRejectsInvalidLaunchTypesAndTransports(t *testing.T) {
	s := openTestStore(t)

	err := s.CreateMCPInstance(&MCPInstance{Name: "bad-launch", ServerKey: "bad", LaunchType: "shell", Transport: mcp.TransportStdio})
	if err == nil {
		t.Fatal("invalid launch type should fail")
	}
	err = s.CreateMCPInstance(&MCPInstance{Name: "bad-transport", ServerKey: "bad", LaunchType: mcp.LaunchHTTP, Transport: "websocket"})
	if err == nil {
		t.Fatal("invalid transport should fail")
	}
}

func TestMCPInstanceGetNotFound(t *testing.T) {
	s := openTestStore(t)

	_, err := s.GetMCPInstance("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
