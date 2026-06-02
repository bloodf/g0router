package cli

import (
	"bytes"
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
	for _, want := range []string{"user service", ".config/systemd/user/g0router.service", ".g0router"} {
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
	for _, want := range []string{"system service", "/etc/systemd/system/g0router.service", "/var/lib/g0router"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want %q", output, want)
		}
	}
}
