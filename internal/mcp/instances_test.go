package mcp

import "testing"

func TestInstanceConfigValidatesLaunchAndTransport(t *testing.T) {
	cfg := InstanceConfig{
		Name:       "docs",
		ServerKey:  "filesystem",
		LaunchType: LaunchCommand,
		Transport:  TransportStdio,
		Command:    "mcp-filesystem",
		Env:        map[string]string{"TOKEN": "secret"},
		Headers:    map[string]string{"Authorization": "Bearer token"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if err := (InstanceConfig{Name: "bad", ServerKey: "bad", LaunchType: "shell", Transport: TransportStdio}).Validate(); err == nil {
		t.Fatal("invalid launch type should fail")
	}
	if err := (InstanceConfig{Name: "bad", ServerKey: "bad", LaunchType: LaunchHTTP, Transport: TransportStdio, URL: "https://mcp.example"}).Validate(); err == nil {
		t.Fatal("http launch with stdio transport should fail")
	}
}

func TestInstanceConfigRedactsSecrets(t *testing.T) {
	cfg := InstanceConfig{
		Name:      "docs",
		ServerKey: "filesystem",
		Env: map[string]string{
			"TOKEN": "secret",
			"MODE":  "read-only",
		},
		Headers: map[string]string{
			"Authorization": "Bearer token",
			"X-Mode":        "read-only",
		},
	}

	redacted := cfg.Redacted()
	if redacted.Env["TOKEN"] != RedactedValue {
		t.Fatalf("env TOKEN = %q, want redacted", redacted.Env["TOKEN"])
	}
	if redacted.Env["MODE"] != "read-only" {
		t.Fatalf("env MODE = %q, want preserved", redacted.Env["MODE"])
	}
	if redacted.Headers["Authorization"] != RedactedValue {
		t.Fatalf("Authorization = %q, want redacted", redacted.Headers["Authorization"])
	}
	if redacted.Headers["X-Mode"] != "read-only" {
		t.Fatalf("X-Mode = %q, want preserved", redacted.Headers["X-Mode"])
	}
}
