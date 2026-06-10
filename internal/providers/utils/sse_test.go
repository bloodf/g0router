package utils

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

func TestSSEScannerNDJSONMode(t *testing.T) {
	input := `{"id":"1"}
prose line to skip
{"id":"2"}
`
	scanner := NewNDJSONScanner(strings.NewReader(input))

	line, err := scanner.Scan()
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if line != `{"id":"1"}` {
		t.Errorf("first line = %q, want {\"id\":\"1\"}", line)
	}

	line, err = scanner.Scan()
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if line != `{"id":"2"}` {
		t.Errorf("second line = %q, want {\"id\":\"2\"}", line)
	}

	_, err = scanner.Scan()
	if err != io.EOF {
		t.Errorf("final scan err = %v, want EOF", err)
	}
}

func TestSSEScannerEventCapture(t *testing.T) {
	input := "event: message_start\n\ndata: payload1\n\ndata: payload2\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	line, err := scanner.Scan()
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if line != "payload1" {
		t.Errorf("first line = %q, want payload1", line)
	}
	if scanner.Event() != "message_start" {
		t.Errorf("Event() after first payload = %q, want message_start", scanner.Event())
	}

	line, err = scanner.Scan()
	if err != nil {
		t.Fatalf("second scan: %v", err)
	}
	if line != "payload2" {
		t.Errorf("second line = %q, want payload2", line)
	}
	if scanner.Event() != "" {
		t.Errorf("Event() after second payload = %q, want empty string", scanner.Event())
	}
}

func TestSSEScannerMalformedLineWarnsAndSkips(t *testing.T) {
	var buf bytes.Buffer
	prevLog := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prevLog)

	input := "data: {invalid json}\n\ndata: ok\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	line, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if line != "ok" {
		t.Errorf("line = %q, want ok", line)
	}

	output := buf.String()
	if !strings.Contains(output, "[WARN]") {
		t.Errorf("expected [WARN] in log output, got: %q", output)
	}
	if !strings.Contains(output, "failed to parse SSE line") {
		t.Errorf("expected 'failed to parse SSE line' in log output, got: %q", output)
	}
}

func TestSSEScannerEOFBadFinalLineReturnsEOF(t *testing.T) {
	var buf bytes.Buffer
	prevLog := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prevLog)

	// No trailing newline; final line is malformed JSON.
	input := "data: ok\n\ndata: {bad json"
	scanner := NewSSEScanner(strings.NewReader(input))

	line, err := scanner.Scan()
	if err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if line != "ok" {
		t.Errorf("first line = %q, want ok", line)
	}

	line, err = scanner.Scan()
	if err != io.EOF {
		t.Errorf("second scan err = %v, want EOF", err)
	}
	if line != "" {
		t.Errorf("second line = %q, want empty string", line)
	}
}

func TestSSEScannerEOFValidUnterminatedLineReturnsPayload(t *testing.T) {
	// No trailing newline; final line is a valid payload.
	input := "data: valid-unterminated"
	scanner := NewSSEScanner(strings.NewReader(input))

	line, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if line != "valid-unterminated" {
		t.Errorf("line = %q, want valid-unterminated", line)
	}

	_, err = scanner.Scan()
	if err != io.EOF {
		t.Errorf("final scan err = %v, want EOF", err)
	}
}
