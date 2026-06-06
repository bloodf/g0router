package handlers

import (
	"errors"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/console"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type fakeConsoleBroker struct {
	cleared      bool
	subscribeID  uint64
	subscribeCh  chan console.Entry
	recent       []console.Entry
	subscribeErr bool
}

func (f *fakeConsoleBroker) Subscribe() (uint64, <-chan console.Entry) {
	return f.subscribeID, f.subscribeCh
}

func (f *fakeConsoleBroker) Unsubscribe(id uint64) {
	if f.subscribeCh != nil {
		close(f.subscribeCh)
		f.subscribeCh = nil
	}
}

func (f *fakeConsoleBroker) Recent() []console.Entry {
	return f.recent
}

func (f *fakeConsoleBroker) Clear() {
	f.cleared = true
}

type fakeConsoleAuditWriter struct {
	entries []store.AuditEntry
	err     error
}

func (f *fakeConsoleAuditWriter) AppendAudit(entry store.AuditEntry) error {
	f.entries = append(f.entries, entry)
	return f.err
}

func TestConsoleLogsStreamNilBroker(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ConsoleLogsStream(&ctx, nil, make(chan struct{}))
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestConsoleLogsClear(t *testing.T) {
	broker := &fakeConsoleBroker{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodDelete)
	ConsoleLogsClear(&ctx, broker, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", ctx.Response.StatusCode())
	}
	if !broker.cleared {
		t.Fatal("broker.Clear() was not called")
	}
}

func TestConsoleLogsClearNilBroker(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodDelete)
	ConsoleLogsClear(&ctx, nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestConsoleLogsClearAudit(t *testing.T) {
	broker := &fakeConsoleBroker{}
	audit := &fakeConsoleAuditWriter{}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodDelete)
	ConsoleLogsClear(&ctx, broker, audit)
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", ctx.Response.StatusCode())
	}
	if len(audit.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(audit.entries))
	}
	if audit.entries[0].Action != "console_logs.clear" {
		t.Fatalf("action = %q, want console_logs.clear", audit.entries[0].Action)
	}
}

func TestConsoleLogsClearAuditErrorIgnored(t *testing.T) {
	broker := &fakeConsoleBroker{}
	audit := &fakeConsoleAuditWriter{err: errors.New("audit error")}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodDelete)
	ConsoleLogsClear(&ctx, broker, audit)
	if ctx.Response.StatusCode() != fasthttp.StatusNoContent {
		t.Fatalf("status = %d, want 204", ctx.Response.StatusCode())
	}
}

func TestConsoleLogsStreamReplaysRecent(t *testing.T) {
	broker := &fakeConsoleBroker{
		recent: []console.Entry{
			{Timestamp: time.Now().UTC(), Level: "INFO", Message: "replay"},
		},
		subscribeCh: make(chan console.Entry, 1),
	}
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ConsoleLogsStream(&ctx, broker, make(chan struct{}))
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d, want 200", ctx.Response.StatusCode())
	}
	if ct := string(ctx.Response.Header.Peek("Content-Type")); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
}

func TestWriteConsoleLogEvent(t *testing.T) {
	// The bufio.Writer path is covered by integration tests in
	// api/server_console_test.go.
}
