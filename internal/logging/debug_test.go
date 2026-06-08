package logging

import (
	"bytes"
	"strings"
	"testing"
)

func TestDebugLoggerDisabled(t *testing.T) {
	var buf bytes.Buffer
	l := NewDebugLoggerWithOutput(false, false, &buf)
	l.Debug("should not appear")
	l.Trace("should not appear")
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestDebugLoggerDebugEnabled(t *testing.T) {
	var buf bytes.Buffer
	l := NewDebugLoggerWithOutput(true, false, &buf)
	l.Debug("hello", "k1", "v1", "k2", 2)
	out := buf.String()
	if !strings.Contains(out, "[DEBUG]") {
		t.Fatalf("expected [DEBUG] in output, got %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("expected message in output, got %q", out)
	}
	if !strings.Contains(out, `k1="v1"`) {
		t.Fatalf("expected k1 attr, got %q", out)
	}
	l.Trace("should not appear")
	if strings.Contains(buf.String(), "[TRACE]") {
		t.Fatal("trace should not appear when only debug is enabled")
	}
}

func TestDebugLoggerTraceEnabled(t *testing.T) {
	var buf bytes.Buffer
	l := NewDebugLoggerWithOutput(false, true, &buf)
	l.Debug("hello")
	l.Trace("deep", "path", "/api/test")
	out := buf.String()
	if !strings.Contains(out, "[DEBUG]") {
		t.Fatalf("expected [DEBUG] in output, got %q", out)
	}
	if !strings.Contains(out, "[TRACE]") {
		t.Fatalf("expected [TRACE] in output, got %q", out)
	}
}

func TestDebugLoggerIsMethods(t *testing.T) {
	l := NewDebugLogger(false, false)
	if l.IsDebug() || l.IsTrace() {
		t.Fatal("expected both flags false")
	}
	l2 := NewDebugLogger(true, true)
	if !l2.IsDebug() || !l2.IsTrace() {
		t.Fatal("expected both flags true")
	}
}
