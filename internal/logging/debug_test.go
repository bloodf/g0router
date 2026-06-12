package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestDebugLogProductionGate(t *testing.T) {
	t.Run("production gate blocks output", func(t *testing.T) {
		getenv := func(key string) string {
			if key == "NODE_ENV" {
				return "production"
			}
			return ""
		}
		var buf bytes.Buffer
		d := NewDebug(getenv, &buf)
		d.Logf("test", "hello %s", "world")
		if buf.Len() != 0 {
			t.Errorf("production wrote %q, want nothing", buf.String())
		}
	})

	t.Run("non-production writes tagged line", func(t *testing.T) {
		getenv := func(key string) string {
			if key == "NODE_ENV" {
				return "development"
			}
			return ""
		}
		var buf bytes.Buffer
		d := NewDebug(getenv, &buf)
		d.Logf("auth", "token %s", "abc")
		got := buf.String()
		if got == "" {
			t.Fatal("expected output, got nothing")
		}
		if !strings.Contains(got, "[DBG:auth]") {
			t.Errorf("output %q missing [DBG:auth]", got)
		}
		if !strings.Contains(got, "token abc") {
			t.Errorf("output %q missing formatted message", got)
		}
	})

	t.Run("empty NODE_ENV writes output", func(t *testing.T) {
		getenv := func(string) string { return "" }
		var buf bytes.Buffer
		d := NewDebug(getenv, &buf)
		d.Logf("x", "msg")
		if buf.Len() == 0 {
			t.Fatal("expected output, got nothing")
		}
	})
}
