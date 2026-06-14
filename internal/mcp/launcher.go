package mcp

import (
	"errors"
	"fmt"
	"sync"

	"github.com/bloodf/g0router/internal/store"
)

// ErrCommandNotAllowed is returned by StartStdio when the requested command is
// not on the allowlist (the security boundary — PAR-MCP-005). It is returned
// BEFORE any process is spawned.
var ErrCommandNotAllowed = errors.New("mcp: command not allowed")

// Launcher spawns and tracks MCP plugin processes. It owns a per-plugin bridge
// registry behind a mutex (one Bridge per plugin — PAR-MCP-003) and an injectable
// ProcessRunner: the REAL osProcessRunner does the real subprocess spawn
// (process.go), while tests inject a fake via SetRunner — mirroring
// tunnel.Service's Runner/SetRunner injection (service.go:32,44).
type Launcher struct {
	st     *store.Store
	mu     sync.RWMutex
	runner ProcessRunner
	bridges map[string]*Bridge // plugin name -> bridge
}

// NewLauncher constructs a Launcher over a store with the REAL osProcessRunner.
// The real runner is the ONLY place a real subprocess is spawned (process.go);
// no unit test exercises it (tests inject a fake via SetRunner).
func NewLauncher(st *store.Store) *Launcher {
	return &Launcher{
		st:      st,
		runner:  newOSProcessRunner(),
		bridges: make(map[string]*Bridge),
	}
}

// SetRunner overrides the process runner (tests inject a deterministic fake).
// Mirrors tunnel.Service.SetRunner (service.go:44).
func (l *Launcher) SetRunner(r ProcessRunner) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.runner = r
}

// StartStdio starts (or reuses) the stdio MCP plugin named name. It FIRST
// validates command against the allowlist (rejecting BEFORE any spawn —
// PAR-MCP-005), then, if no live bridge exists for name, spawns the process via
// the runner and registers a Bridge. If a live bridge already exists it is reused
// (one bridge per plugin; re-spawn only if the prior process exited —
// ESC-RESPAWN). Returns the active Bridge.
func (l *Launcher) StartStdio(name, command string, args []string, env map[string]string) (*Bridge, error) {
	if !isAllowedCommand(command) {
		return nil, fmt.Errorf("%w: %q", ErrCommandNotAllowed, command)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Reuse an existing bridge whose process is still running.
	if b, ok := l.bridges[name]; ok && b.IsRunning() {
		return b, nil
	}
	// A prior bridge whose process exited is replaced.
	delete(l.bridges, name)

	bridge := &Bridge{sessions: make(map[string]SessionSink)}
	bridge.SetOnExit(func(int) { l.removeBridge(name) })

	proc, err := l.runner.Start(ProcessSpec{
		Command:  command,
		Args:     args,
		Env:      env,
		OnFrame:  bridge.onFrame,
		OnStderr: bridge.onStderr,
		OnExit:   bridge.onExit,
	})
	if err != nil {
		return nil, fmt.Errorf("start mcp process %s: %w", name, err)
	}
	bridge.proc = proc
	l.bridges[name] = bridge
	return bridge, nil
}

// StartHTTP records an HTTP-transport MCP instance for name at url. It performs
// NO live dial (the real HTTP/SSE client is w7-mcp-2); it only models the mode +
// exposes the URL the mcp-2 client will use.
func (l *Launcher) StartHTTP(name, url string) error {
	return l.recordInstance(name, "http", url)
}

// StartSSE records an SSE-transport MCP instance for name at url. NO live dial
// (w7-mcp-2).
func (l *Launcher) StartSSE(name, url string) error {
	return l.recordInstance(name, "sse", url)
}

// recordInstance is the no-dial bookkeeping shared by StartHTTP/StartSSE. The
// instance persistence belongs to the store; here it is a placeholder seam the
// mcp-2 transport will extend.
func (l *Launcher) recordInstance(name, transport, url string) error {
	_ = name
	_ = transport
	_ = url
	return nil
}

// IsRunning reports whether the named plugin's process is alive (PAR-MCP-051).
func (l *Launcher) IsRunning(name string) bool {
	l.mu.RLock()
	b, ok := l.bridges[name]
	l.mu.RUnlock()
	return ok && b.IsRunning()
}

// Bridge returns the active bridge for name, if any.
func (l *Launcher) Bridge(name string) (*Bridge, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	b, ok := l.bridges[name]
	return b, ok
}

// Stop kills the named plugin's process and removes its bridge. Idempotent.
func (l *Launcher) Stop(name string) error {
	l.mu.Lock()
	b, ok := l.bridges[name]
	delete(l.bridges, name)
	l.mu.Unlock()
	if !ok {
		return nil
	}
	return b.Stop()
}

// removeBridge deletes the named bridge from the registry (the child-exit
// handler — PAR-MCP-053).
func (l *Launcher) removeBridge(name string) {
	l.mu.Lock()
	delete(l.bridges, name)
	l.mu.Unlock()
}

// bridgeCount returns the number of registered bridges (test/introspection).
func (l *Launcher) bridgeCount() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.bridges)
}
