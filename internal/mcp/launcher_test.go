package mcp

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// fakeRunner is a deterministic ProcessRunner used to unit-test the launcher
// WITHOUT spawning any real process. It records Start calls and hands back a
// controllable fakeProcess whose callbacks the launcher wires up.
type fakeRunner struct {
	mu         sync.Mutex
	startCount int
	lastSpec   ProcessSpec
	startErr   error
	procs      []*fakeProcess
}

func (f *fakeRunner) Start(spec ProcessSpec) (Process, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCount++
	f.lastSpec = spec
	if f.startErr != nil {
		return nil, f.startErr
	}
	p := &fakeProcess{running: true}
	f.procs = append(f.procs, p)
	return p, nil
}

func (f *fakeRunner) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.startCount
}

func newLauncherTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// TestLauncherRejectsNonAllowlistedBeforeSpawn: StartStdio rejects a command that
// is not on the allowlist BEFORE invoking the runner (PAR-MCP-005). The fake
// runner's Start must NEVER be called.
func TestLauncherRejectsNonAllowlistedBeforeSpawn(t *testing.T) {
	l := NewLauncher(newLauncherTestStore(t))
	fake := &fakeRunner{}
	l.SetRunner(fake)

	_, err := l.StartStdio("evil", "rm", []string{"-rf", "/"}, nil)
	if err == nil {
		t.Fatalf("StartStdio(rm) = nil error, want rejection")
	}
	if !errors.Is(err, ErrCommandNotAllowed) {
		t.Fatalf("err = %v, want ErrCommandNotAllowed", err)
	}
	if fake.calls() != 0 {
		t.Fatalf("runner.Start called %d times for a rejected command, want 0", fake.calls())
	}
}

// TestLauncherStartStdioAllowlistedSpawns: an allowlisted command starts a bridge
// and the runner is invoked exactly once with the spec.
func TestLauncherStartStdioAllowlistedSpawns(t *testing.T) {
	l := NewLauncher(newLauncherTestStore(t))
	fake := &fakeRunner{}
	l.SetRunner(fake)

	b, err := l.StartStdio("browsermcp", "npx", []string{"-y", "@browsermcp/mcp@latest"}, nil)
	if err != nil {
		t.Fatalf("StartStdio: %v", err)
	}
	if b == nil {
		t.Fatalf("StartStdio returned nil bridge")
	}
	if fake.calls() != 1 {
		t.Fatalf("runner.Start calls = %d, want 1", fake.calls())
	}
	if fake.lastSpec.Command != "npx" || len(fake.lastSpec.Args) != 2 {
		t.Fatalf("spec not forwarded: %+v", fake.lastSpec)
	}
	if !l.IsRunning("browsermcp") {
		t.Fatalf("IsRunning(browsermcp) = false after start")
	}
}

// TestLauncherOneBridgePerPlugin: starting the same plugin while its process is
// running reuses the existing bridge and does NOT re-spawn (PAR-MCP-003 +
// re-spawn-iff-not-running, ESC-RESPAWN).
func TestLauncherOneBridgePerPlugin(t *testing.T) {
	l := NewLauncher(newLauncherTestStore(t))
	fake := &fakeRunner{}
	l.SetRunner(fake)

	b1, err := l.StartStdio("p", "npx", nil, nil)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	b2, err := l.StartStdio("p", "npx", nil, nil)
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if b1 != b2 {
		t.Fatalf("expected the same bridge reused while running")
	}
	if fake.calls() != 1 {
		t.Fatalf("re-spawn while running: calls = %d, want 1", fake.calls())
	}
}

// TestLauncherRespawnsAfterExit: once a plugin's process has exited, a new
// StartStdio re-spawns it (re-spawn-iff-not-running).
func TestLauncherRespawnsAfterExit(t *testing.T) {
	l := NewLauncher(newLauncherTestStore(t))
	fake := &fakeRunner{}
	l.SetRunner(fake)

	if _, err := l.StartStdio("p", "npx", nil, nil); err != nil {
		t.Fatalf("first start: %v", err)
	}
	// Simulate the child exiting → registry-delete (PAR-MCP-053).
	fake.procs[0].Stop()
	fake.lastSpec.OnExit(0)

	if l.IsRunning("p") {
		t.Fatalf("IsRunning(p) = true after exit, want false")
	}
	if _, err := l.StartStdio("p", "npx", nil, nil); err != nil {
		t.Fatalf("re-start: %v", err)
	}
	if fake.calls() != 2 {
		t.Fatalf("re-spawn after exit: calls = %d, want 2", fake.calls())
	}
}

// TestLauncherExitRemovesBridge: the OnExit callback removes the bridge from the
// registry (PAR-MCP-053).
func TestLauncherExitRemovesBridge(t *testing.T) {
	l := NewLauncher(newLauncherTestStore(t))
	fake := &fakeRunner{}
	l.SetRunner(fake)

	if _, err := l.StartStdio("p", "npx", nil, nil); err != nil {
		t.Fatalf("start: %v", err)
	}
	if l.bridgeCount() != 1 {
		t.Fatalf("bridgeCount = %d, want 1", l.bridgeCount())
	}
	fake.lastSpec.OnExit(0)
	if l.bridgeCount() != 0 {
		t.Fatalf("bridge not removed on exit: bridgeCount = %d, want 0", l.bridgeCount())
	}
}

// TestLauncherStartHTTPSSEPersistMode: StartHTTP/StartSSE record the transport
// mode + URL without any live dial (no runner spawn).
func TestLauncherStartHTTPSSEModes(t *testing.T) {
	l := NewLauncher(newLauncherTestStore(t))
	fake := &fakeRunner{}
	l.SetRunner(fake)

	if err := l.StartHTTP("exa", "https://exa.example.com/mcp"); err != nil {
		t.Fatalf("StartHTTP: %v", err)
	}
	if err := l.StartSSE("tavily", "https://tavily.example.com/sse"); err != nil {
		t.Fatalf("StartSSE: %v", err)
	}
	// No process spawned for HTTP/SSE modes.
	if fake.calls() != 0 {
		t.Fatalf("HTTP/SSE modes spawned a process: calls = %d, want 0", fake.calls())
	}
}
