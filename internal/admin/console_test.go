package admin

import (
	"bufio"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/logging"
	"github.com/valyala/fasthttp"
)

func TestConsoleLogStreamEmitsFrame(t *testing.T) {
	env := newTestEnv(t)
	console := logging.NewConsoleLog(16)
	env.handlers.SetConsoleLog(console)

	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	done := make(chan struct{})

	h := &ConsoleStreamHandler{Handlers: env.handlers, Keepalive: time.Hour}
	go h.serve(w, nil, done)

	// Give the subscription a moment, then push a line.
	time.Sleep(20 * time.Millisecond)
	console.Append("warn", "boom happened")

	var frames []streamFrame
	waitFor(t, time.Second, func() bool {
		frames = parseFrames(t, rec.Bytes())
		for _, f := range frames {
			if f.comment {
				continue
			}
			return true
		}
		return false
	})

	var got map[string]any
	for _, f := range frames {
		if f.comment {
			continue
		}
		got = frameData(t, f)
		break
	}
	if got == nil {
		t.Fatalf("no data frame emitted; frames = %v", frames)
	}
	if got["level"] != "warn" {
		t.Fatalf("frame level = %v, want warn", got["level"])
	}
	if got["message"] != "boom happened" {
		t.Fatalf("frame message = %v, want %q", got["message"], "boom happened")
	}
	if _, ok := got["timestamp"].(string); !ok || got["timestamp"] == "" {
		t.Fatalf("frame timestamp missing: %v", got["timestamp"])
	}

	rec.Close()
	console.Append("info", "trigger close")
	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestConsoleLogStreamReplaysRecent(t *testing.T) {
	env := newTestEnv(t)
	console := logging.NewConsoleLog(16)
	console.Append("error", "earlier line")
	env.handlers.SetConsoleLog(console)

	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	done := make(chan struct{})

	h := &ConsoleStreamHandler{Handlers: env.handlers, Keepalive: time.Hour}
	go h.serve(w, nil, done)

	var got map[string]any
	waitFor(t, time.Second, func() bool {
		for _, f := range parseFrames(t, rec.Bytes()) {
			if f.comment {
				continue
			}
			got = frameData(t, f)
			return true
		}
		return false
	})
	if got == nil || got["message"] != "earlier line" {
		t.Fatalf("replay frame = %v, want message %q", got, "earlier line")
	}

	rec.Close()
	console.Append("info", "trigger close")
	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})
}

func TestConsoleLogStreamNilConsole501(t *testing.T) {
	env := newTestEnv(t)
	// No SetConsoleLog: console is nil → 501.
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/console-logs/stream")
	h := &ConsoleStreamHandler{Handlers: env.handlers}
	h.ConsoleLogStream(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusNotImplemented {
		t.Fatalf("nil console status = %d, want 501", ctx.Response.StatusCode())
	}
}
