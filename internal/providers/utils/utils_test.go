package utils

import (
	"io"
	"strings"
	"testing"
)

func TestSSEScanner(t *testing.T) {
	input := "data: {\"id\":\"1\"}\n\ndata: {\"id\":\"2\"}\n\ndata: [DONE]\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

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

	line, err = scanner.Scan()
	if err != nil {
		t.Fatalf("third scan: %v", err)
	}
	if line != "[DONE]" {
		t.Errorf("third line = %q, want [DONE]", line)
	}

	_, err = scanner.Scan()
	if err != io.EOF {
		t.Errorf("final scan err = %v, want EOF", err)
	}
}

func TestSSEScannerMalformed(t *testing.T) {
	input := "event: message\n\ndata: ok\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	line, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if line != "ok" {
		t.Errorf("line = %q, want ok", line)
	}
}

func TestPtrHelpers(t *testing.T) {
	f := PtrFloat64(1.5)
	if *f != 1.5 {
		t.Errorf("PtrFloat64 = %v, want 1.5", *f)
	}
	i := PtrInt(42)
	if *i != 42 {
		t.Errorf("PtrInt = %v, want 42", *i)
	}
	s := PtrString("hello")
	if *s != "hello" {
		t.Errorf("PtrString = %v, want hello", *s)
	}
}

func TestSetJSONBody(t *testing.T) {
	pool := NewClientPool()
	req := pool.AcquireRequest()
	defer pool.ReleaseRequest(req)

	v := map[string]string{"key": "value"}
	if err := SetJSONBody(req, v); err != nil {
		t.Fatalf("SetJSONBody: %v", err)
	}
	if string(req.Body()) != `{"key":"value"}` {
		t.Errorf("body = %s, want {\"key\":\"value\"}", req.Body())
	}
	ct := string(req.Header.ContentType())
	if ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
}

func TestReadJSONBody(t *testing.T) {
	pool := NewClientPool()
	resp := pool.AcquireResponse()
	defer pool.ReleaseResponse(resp)

	resp.SetBodyRaw([]byte(`{"result":"ok"}`))
	var v map[string]string
	if err := ReadJSONBody(resp, &v); err != nil {
		t.Fatalf("ReadJSONBody: %v", err)
	}
	if v["result"] != "ok" {
		t.Errorf("result = %q, want ok", v["result"])
	}
}

func TestSetAuthHeader(t *testing.T) {
	pool := NewClientPool()
	req := pool.AcquireRequest()
	defer pool.ReleaseRequest(req)

	SetAuthHeader(req, "secret-token")
	auth := string(req.Header.Peek("Authorization"))
	if auth != "Bearer secret-token" {
		t.Errorf("auth = %q, want Bearer secret-token", auth)
	}
}
