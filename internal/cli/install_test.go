package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCommandPrintsUserPlan(t *testing.T) {
	cmd := NewInstallCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--user"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"user service", ".config/systemd/user/g0router.service", ".g0router", "deploy/g0router.service"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}

func TestInstallCommandPrintsSystemPlan(t *testing.T) {
	cmd := NewInstallCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"system service", "/etc/systemd/system/g0router.service", "/etc/default/g0router", "/var/lib/g0router", "deploy/g0router.default"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}

func TestUninstallCommandPrintsRemovalPlan(t *testing.T) {
	cmd := NewRootCommand("test")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"uninstall"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	for _, want := range []string{"Remove systemd service", "systemctl disable --now g0router", "keeps data"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}

func TestDeployTemplatesExist(t *testing.T) {
	service, err := os.ReadFile(filepath.Join("..", "..", "deploy", "g0router.service"))
	if err != nil {
		t.Fatalf("read service: %v", err)
	}
	for _, want := range []string{"[Unit]", "ExecStart=/usr/local/bin/g0router serve", "ReadWritePaths=/var/lib/g0router"} {
		if !strings.Contains(string(service), want) {
			t.Fatalf("service = %q, want %q", string(service), want)
		}
	}

	defaults, err := os.ReadFile(filepath.Join("..", "..", "deploy", "g0router.default"))
	if err != nil {
		t.Fatalf("read defaults: %v", err)
	}
	for _, want := range []string{"PORT=20128", "DATA_DIR=/var/lib/g0router", "REQUIRE_API_KEY=true"} {
		if !strings.Contains(string(defaults), want) {
			t.Fatalf("defaults = %q, want %q", string(defaults), want)
		}
	}
}
