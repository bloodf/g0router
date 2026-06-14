package admin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bloodf/g0router/internal/logging"
	"github.com/valyala/fasthttp"
)

// ConsoleStreamHandler serves GET /api/console-logs/stream as a Server-Sent
// Events stream of captured server log lines. It mirrors UsageStreamHandler:
// SetBodyStreamWriter + a select loop over the client-done channel and a
// keepalive ticker, replaying Recent() then streaming Subscribe() frames.
type ConsoleStreamHandler struct {
	Handlers  *Handlers
	Keepalive time.Duration
}

// ConsoleLogStream writes an SSE stream of console log lines. When no console
// log is wired it reports 501 (mirrors the nil-safe Shutdown precedent).
func (h *ConsoleStreamHandler) ConsoleLogStream(ctx *fasthttp.RequestCtx) {
	if h.Handlers.console == nil {
		writeError(ctx, fasthttp.StatusNotImplemented, "console log stream not available")
		return
	}
	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		h.serve(w, ctx.Done(), nil)
	})
}

// serve runs the SSE loop against the provided writer. If done is non-nil it is
// closed when the loop exits, which lets tests observe goroutine termination.
func (h *ConsoleStreamHandler) serve(w *bufio.Writer, clientDone <-chan struct{}, done chan<- struct{}) {
	if done != nil {
		defer close(done)
	}

	console := h.Handlers.console
	if console == nil {
		return
	}

	interval := h.Keepalive
	if interval <= 0 {
		interval = productionKeepalive
	}

	state := &consoleStreamState{writer: w}

	// Replay recent lines first so a fresh client sees context.
	for _, line := range console.Recent() {
		if err := state.writeLine(line); err != nil {
			return
		}
	}

	frames, unsubscribe := console.Subscribe()
	defer unsubscribe()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-clientDone:
			return
		case line, ok := <-frames:
			if !ok {
				return
			}
			if err := state.writeLine(line); err != nil {
				return
			}
		case <-ticker.C:
			if err := state.writePing(); err != nil {
				return
			}
		}
	}
}

// consoleStreamState serializes writes to a single SSE stream.
type consoleStreamState struct {
	mu     sync.Mutex
	closed bool
	writer *bufio.Writer
}

func (s *consoleStreamState) writeLine(line logging.ConsoleLine) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	b, err := json.Marshal(line)
	if err != nil {
		s.closed = true
		return fmt.Errorf("marshal console frame: %w", err)
	}
	if _, err := s.writer.WriteString("data: "); err != nil {
		s.closed = true
		return err
	}
	if _, err := s.writer.Write(b); err != nil {
		s.closed = true
		return err
	}
	if _, err := s.writer.WriteString("\n\n"); err != nil {
		s.closed = true
		return err
	}
	if err := s.writer.Flush(); err != nil {
		s.closed = true
		return err
	}
	return nil
}

func (s *consoleStreamState) writePing() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	if _, err := s.writer.WriteString(": ping\n\n"); err != nil {
		s.closed = true
		return err
	}
	return s.writer.Flush()
}
