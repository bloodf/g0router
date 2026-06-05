package mcp

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
)

func TestMatchesJSONType(t *testing.T) {
	cases := []struct {
		value any
		typ   string
		want  bool
	}{
		{map[string]any{}, "object", true},
		{"x", "object", false},
		{[]any{}, "array", true},
		{"x", "array", false},
		{"hello", "string", true},
		{1.0, "string", false},
		{true, "boolean", true},
		{"x", "boolean", false},
		{3.14, "number", true},
		{"x", "number", false},
		{float64(5), "integer", true},
		{5.5, "integer", false},
		{"x", "integer", false},
		{"anything", "unknown-type", true},
	}
	for _, tc := range cases {
		if got := matchesJSONType(tc.value, tc.typ); got != tc.want {
			t.Errorf("matchesJSONType(%v, %q) = %v, want %v", tc.value, tc.typ, got, tc.want)
		}
	}
}

func TestValidateToolArguments(t *testing.T) {
	schema := json.RawMessage(`{
		"type":"object",
		"required":["query"],
		"properties":{"query":{"type":"string"},"limit":{"type":"integer"}},
		"additionalProperties":false
	}`)

	// Valid.
	if err := validateToolArguments(schema, json.RawMessage(`{"query":"hi","limit":5}`)); err != nil {
		t.Fatalf("valid args: %v", err)
	}
	// Missing required.
	if err := validateToolArguments(schema, json.RawMessage(`{"limit":5}`)); err == nil {
		t.Fatal("missing required: want error")
	}
	// Wrong type.
	if err := validateToolArguments(schema, json.RawMessage(`{"query":123}`)); err == nil {
		t.Fatal("wrong type: want error")
	}
	// Additional property not allowed.
	if err := validateToolArguments(schema, json.RawMessage(`{"query":"hi","extra":1}`)); err == nil {
		t.Fatal("additional property: want error")
	}
	// Top-level type mismatch.
	if err := validateToolArguments(schema, json.RawMessage(`"notanobject"`)); err == nil {
		t.Fatal("type mismatch: want error")
	}
	// Invalid schema.
	if err := validateToolArguments(json.RawMessage(`notjson`), json.RawMessage(`{}`)); err == nil {
		t.Fatal("invalid schema: want error")
	}
	// Invalid arguments JSON.
	if err := validateToolArguments(json.RawMessage(`{"type":"object"}`), json.RawMessage(`notjson`)); err == nil {
		t.Fatal("invalid arguments: want error")
	}
}

func TestNormalizedToolArguments(t *testing.T) {
	if got := string(normalizedToolArguments(nil)); got != `{}` {
		t.Fatalf("nil args = %q, want {}", got)
	}
	if got := string(normalizedToolArguments(json.RawMessage(`  `))); got != `{}` {
		t.Fatalf("blank args = %q, want {}", got)
	}
	if got := string(normalizedToolArguments(json.RawMessage(`{"a":1}`))); got != `{"a":1}` {
		t.Fatalf("args = %q", got)
	}
}

func TestInstanceConfigValidate(t *testing.T) {
	valid := InstanceConfig{Name: "n", ServerKey: "k", LaunchType: LaunchCommand, Transport: TransportStdio, Command: "x"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	httpValid := InstanceConfig{Name: "n", ServerKey: "k", LaunchType: LaunchHTTP, Transport: TransportStreamableHTTP, URL: "http://x"}
	if err := httpValid.Validate(); err != nil {
		t.Fatalf("valid http config: %v", err)
	}

	bad := []InstanceConfig{
		{ServerKey: "k", LaunchType: LaunchCommand, Transport: TransportStdio},                        // missing name
		{Name: "n", LaunchType: LaunchCommand, Transport: TransportStdio},                             // missing server key
		{Name: "n", ServerKey: "k", LaunchType: "invalid", Transport: TransportStdio},                 // bad launch type
		{Name: "n", ServerKey: "k", LaunchType: LaunchCommand, Transport: "invalid"},                  // bad transport
		{Name: "n", ServerKey: "k", LaunchType: LaunchCommand, Transport: TransportStreamableHTTP},    // command needs stdio
		{Name: "n", ServerKey: "k", LaunchType: LaunchHTTP, Transport: TransportStdio},                // http needs http transport
		{Name: "n", ServerKey: "k", LaunchType: LaunchHTTP, Transport: TransportStreamableHTTP},       // http needs url
	}
	for i, cfg := range bad {
		if err := cfg.Validate(); !errors.Is(err, ErrInvalidInstanceConfig) {
			t.Errorf("bad config %d: err = %v, want ErrInvalidInstanceConfig", i, err)
		}
	}
}

func TestValidTransportAndLaunchType(t *testing.T) {
	for _, tr := range []Transport{TransportStdio, TransportSSE, TransportStreamableHTTP} {
		if !validTransport(tr) {
			t.Errorf("validTransport(%q) = false", tr)
		}
	}
	if validTransport("nope") {
		t.Error("validTransport(nope) = true")
	}
	for _, lt := range []LaunchType{LaunchCommand, LaunchNPX, LaunchDocker, LaunchHTTP} {
		if !validLaunchType(lt) {
			t.Errorf("validLaunchType(%q) = false", lt)
		}
	}
	if validLaunchType("nope") {
		t.Error("validLaunchType(nope) = true")
	}
}

func TestJSONRPCErrorString(t *testing.T) {
	var nilErr *jsonrpcError
	if nilErr.Error() != "" {
		t.Fatal("nil jsonrpcError should be empty string")
	}
	err := &jsonrpcError{Code: -32600, Message: "Invalid Request"}
	if err.Error() != "mcp json-rpc error -32600: Invalid Request" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRealHealthTicker(t *testing.T) {
	ticker := newRealHealthTicker(time.Millisecond)
	if ticker.C() == nil {
		t.Fatal("ticker channel is nil")
	}
	ticker.Stop()
}

func TestFirstChoiceToolCalls(t *testing.T) {
	if firstChoiceToolCalls(nil) != nil {
		t.Fatal("nil resp should yield nil")
	}
	if firstChoiceToolCalls(&providers.ChatResponse{}) != nil {
		t.Fatal("no choices should yield nil")
	}
	resp := &providers.ChatResponse{
		Choices: []providers.Choice{{Message: providers.Message{ToolCalls: []providers.ToolCall{{ID: "1"}}}}},
	}
	if calls := firstChoiceToolCalls(resp); len(calls) != 1 || calls[0].ID != "1" {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestToolResultContent(t *testing.T) {
	if toolResultContent(nil) != "" {
		t.Fatal("nil content should be empty string")
	}
	if toolResultContent("hi") != "hi" {
		t.Fatal("string content passthrough")
	}
	blocks := []any{"a", "b"}
	got := toolResultContent(blocks)
	if gotSlice, ok := got.([]any); !ok || len(gotSlice) != 2 {
		t.Fatalf("default passthrough = %v", got)
	}
}
