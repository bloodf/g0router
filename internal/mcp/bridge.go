package mcp

import (
	"bytes"
	"sync"
)

// SessionSink delivers one JSON-RPC frame to one SSE session. A real sink writes
// to the http.ResponseWriter/flusher (w7-mcp-2/3); the test sink is an in-memory
// recorder. A sink that returns an error is treated as a broken pipe and dropped
// (PAR-MCP-054).
type SessionSink func(frame []byte) error

// Bridge wraps one MCP child Process and fans its newline-delimited stdout
// JSON-RPC frames out to a set of SSE session sinks. One Bridge exists per plugin
// (PAR-MCP-003). The framing/broadcast/lifecycle wiring is fully unit-testable
// against a fake Process — no real subprocess required.
type Bridge struct {
	mu       sync.RWMutex
	proc     Process
	sessions map[string]SessionSink
	buffer   []byte // accumulated partial stdout, fed through splitFrames

	onExitFn   func(code int)
	onStderrFn func(line string)
}

// newBridge constructs a Bridge over a Process (real or fake).
func newBridge(proc Process) *Bridge {
	return &Bridge{
		proc:     proc,
		sessions: make(map[string]SessionSink),
	}
}

// AddSession registers an SSE consumer under sid.
func (b *Bridge) AddSession(sid string, sink SessionSink) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessions[sid] = sink
}

// RemoveSession unregisters the SSE consumer sid.
func (b *Bridge) RemoveSession(sid string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.sessions, sid)
}

// sessionCount returns the number of registered sessions (test/introspection).
func (b *Bridge) sessionCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.sessions)
}

// SetOnExit registers the callback invoked once when the child process exits
// (the launcher wires this to its registry-delete — PAR-MCP-053).
func (b *Bridge) SetOnExit(fn func(code int)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onExitFn = fn
}

// SetOnStderr registers the callback invoked for each child stderr line
// (PAR-MCP-052).
func (b *Bridge) SetOnStderr(fn func(line string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onStderrFn = fn
}

// onFrame accumulates a stdout chunk, splits out every complete newline-delimited
// JSON-RPC frame, and broadcasts each to all session sinks (PAR-MCP-007). A sink
// that errors is dropped and never aborts the broadcast (PAR-MCP-054). Wired as
// the Process's stdout callback.
func (b *Bridge) onFrame(chunk []byte) {
	b.mu.Lock()
	b.buffer = append(b.buffer, chunk...)
	frames, rest := splitFrames(b.buffer)
	b.buffer = rest
	b.mu.Unlock()

	for _, frame := range frames {
		b.broadcast(frame)
	}
}

// broadcast sends one frame to every session sink, dropping any sink that errors
// (broken pipe) without aborting the loop.
func (b *Bridge) broadcast(frame []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for sid, sink := range b.sessions {
		if err := sink(frame); err != nil {
			delete(b.sessions, sid)
		}
	}
}

// onStderr forwards a child stderr line to the registered callback (PAR-MCP-052).
func (b *Bridge) onStderr(line string) {
	b.mu.RLock()
	fn := b.onStderrFn
	b.mu.RUnlock()
	if fn != nil {
		fn(line)
	}
}

// onExit forwards the child exit code to the registered callback (PAR-MCP-053).
func (b *Bridge) onExit(code int) {
	b.mu.RLock()
	fn := b.onExitFn
	b.mu.RUnlock()
	if fn != nil {
		fn(code)
	}
}

// Send writes a JSON-RPC frame to the child process stdin.
func (b *Bridge) Send(frame []byte) error {
	return b.proc.Write(frame)
}

// IsRunning reports whether the underlying child process is alive (PAR-MCP-051).
func (b *Bridge) IsRunning() bool {
	return b.proc.IsRunning()
}

// Stop kills the underlying child process. Idempotent.
func (b *Bridge) Stop() error {
	return b.proc.Stop()
}

// splitFrames consumes complete newline-delimited JSON frames from buf, returning
// the complete frames and the remaining partial tail (the bytes after the last
// newline, held until its own newline arrives in a later chunk). Blank lines are
// skipped. PURE — no I/O. Mirrors 9router's newline-split of proc.stdout
// (stdioSseBridge.js:151).
func splitFrames(buf []byte) (frames [][]byte, rest []byte) {
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			break
		}
		line := bytes.TrimRight(buf[:i], "\r")
		if len(line) > 0 {
			// Copy so callers may reuse/append to the original buffer safely.
			frame := make([]byte, len(line))
			copy(frame, line)
			frames = append(frames, frame)
		}
		buf = buf[i+1:]
	}
	return frames, buf
}
