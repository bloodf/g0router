package mcp

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestParseSSEFrameEndpointEvent(t *testing.T) {
	// The 9router server emits "event: endpoint\ndata: <messageURL>\n\n"
	// (sse/route.js:22). The client reads the event name + data payload.
	block := []byte("event: endpoint\ndata: /api/mcp/exa/message?sessionId=abc\n\n")
	event, data := parseSSEFrame(block)
	if event != "endpoint" {
		t.Fatalf("event = %q, want endpoint", event)
	}
	if data != "/api/mcp/exa/message?sessionId=abc" {
		t.Fatalf("data = %q", data)
	}
}

func TestParseSSEFrameNoEvent(t *testing.T) {
	// A frame with only a data line (default "message" event).
	block := []byte("data: {\"jsonrpc\":\"2.0\"}\n\n")
	event, data := parseSSEFrame(block)
	if event != "" {
		t.Fatalf("event = %q, want empty", event)
	}
	if data != `{"jsonrpc":"2.0"}` {
		t.Fatalf("data = %q", data)
	}
}

func TestParseSSEDataFramesMultiple(t *testing.T) {
	// Mirrors cowork-mcp-tools/route.js:60-69: split on newlines, keep lines
	// beginning "data:", strip the prefix (and one optional leading space).
	body := "event: message\n" +
		"data: {\"id\":1}\n" +
		"\n" +
		"data: {\"id\":2}\n" +
		"\n"
	got := parseSSEDataFrames(body)
	want := []string{`{"id":1}`, `{"id":2}`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSSEDataFrames = %#v, want %#v", got, want)
	}
}

func TestParseSSEDataFramesIgnoresBlankAndComments(t *testing.T) {
	body := "\n: keep-alive comment\ndata: hello\n\ndata:\n"
	got := parseSSEDataFrames(body)
	want := []string{"hello", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSSEDataFrames = %#v, want %#v", got, want)
	}
}

func TestPostMessageSends202(t *testing.T) {
	ft := &fakeTransport{responses: []fakeResp{{status: http.StatusAccepted}}}
	c := newSSEClient(fakeClient(ft))

	payload := []byte(`{"jsonrpc":"2.0","method":"ping","id":9}`)
	if err := c.postMessage(context.Background(), "https://srv.example/message?sessionId=xyz", payload); err != nil {
		t.Fatalf("postMessage: %v", err)
	}

	if len(ft.captured) != 1 {
		t.Fatalf("captured %d requests, want 1", len(ft.captured))
	}
	req := ft.captured[0]
	if req.Method != http.MethodPost {
		t.Fatalf("method = %s, want POST", req.Method)
	}
	if req.URL.String() != "https://srv.example/message?sessionId=xyz" {
		t.Fatalf("url = %s", req.URL.String())
	}
	if got := ft.bodyAt(0); got != string(payload) {
		t.Fatalf("body = %q, want %q", got, payload)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestPostMessageNon202Errors(t *testing.T) {
	ft := &fakeTransport{responses: []fakeResp{{status: http.StatusInternalServerError, body: "boom"}}}
	c := newSSEClient(fakeClient(ft))
	if err := c.postMessage(context.Background(), "https://srv.example/message", []byte(`{}`)); err == nil {
		t.Fatalf("expected error on non-202 status")
	}
}
