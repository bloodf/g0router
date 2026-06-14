package mcp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// INTEGRATION-ONLY — not unit-tested.
//
// This file is the ONLY place os/exec appears. It implements the real
// ProcessRunner/Process by spawning an MCP stdio child via os/exec
// (StdinPipe/StdoutPipe/StderrPipe + cmd.Wait + goroutine scanners that feed
// splitFrames / OnStderr / OnExit). Spawning arbitrary MCP servers in CI is the
// exact RCE risk 9router cites (PAR-MCP-042), so it is excluded from
// `go test ./...` determinism: every unit test injects a fake runner via
// Launcher.SetRunner and never reaches this code. Mirrors the integration-only
// real-spawn precedent in internal/platform/tunnel/cloudflared.go.

// osProcessRunner is the real ProcessRunner. INTEGRATION-ONLY.
type osProcessRunner struct{}

// newOSProcessRunner constructs the real runner used by NewLauncher in
// production. INTEGRATION-ONLY.
func newOSProcessRunner() *osProcessRunner { return &osProcessRunner{} }

// Start spawns the child process described by spec and wires its stdout (via
// splitFrames), stderr, and exit back to spec's callbacks. INTEGRATION-ONLY.
func (r *osProcessRunner) Start(spec ProcessSpec) (Process, error) {
	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Env = mergeEnv(spec.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", spec.Command, err)
	}

	p := &osProcess{cmd: cmd, stdin: stdin}

	// stdout goroutine: accumulate + split newline-delimited JSON-RPC frames.
	go func() {
		buf := make([]byte, 0, 4096)
		chunk := make([]byte, 4096)
		for {
			n, rerr := stdout.Read(chunk)
			if n > 0 {
				buf = append(buf, chunk[:n]...)
				frames, rest := splitFrames(buf)
				buf = rest
				if spec.OnFrame != nil {
					for _, f := range frames {
						spec.OnFrame(f)
					}
				}
			}
			if rerr != nil {
				return
			}
		}
	}()

	// stderr goroutine: forward each line (PAR-MCP-052).
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			if spec.OnStderr != nil {
				spec.OnStderr(sc.Text())
			}
		}
	}()

	// wait goroutine: fire OnExit once with the exit code (PAR-MCP-053).
	go func() {
		werr := cmd.Wait()
		p.markExited()
		if spec.OnExit != nil {
			spec.OnExit(exitCode(werr))
		}
	}()

	return p, nil
}

// osProcess is one real running child. INTEGRATION-ONLY.
type osProcess struct {
	cmd   *exec.Cmd
	stdin io.WriteCloser

	mu     sync.Mutex
	exited bool
}

// Write sends a newline-delimited JSON-RPC frame to the child's stdin.
func (p *osProcess) Write(frame []byte) error {
	if _, err := p.stdin.Write(append(frame, '\n')); err != nil {
		return fmt.Errorf("write to mcp stdin: %w", err)
	}
	return nil
}

// IsRunning reports liveness via the process state (PAR-MCP-051).
func (p *osProcess) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited {
		return false
	}
	return p.cmd.ProcessState == nil || !p.cmd.ProcessState.Exited()
}

// Stop kills the child. Idempotent.
func (p *osProcess) Stop() error {
	if p.cmd.Process == nil {
		return nil
	}
	if err := p.cmd.Process.Kill(); err != nil && !p.alreadyExited() {
		return fmt.Errorf("kill mcp process: %w", err)
	}
	return nil
}

func (p *osProcess) markExited() {
	p.mu.Lock()
	p.exited = true
	p.mu.Unlock()
}

func (p *osProcess) alreadyExited() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exited
}

// mergeEnv combines the process environment with the spec's overrides.
func mergeEnv(extra map[string]string) []string {
	env := os.Environ()
	for k, v := range extra {
		env = append(env, k+"="+v)
	}
	return env
}

// exitCode extracts the integer exit code from a cmd.Wait error (0 on success).
func exitCode(werr error) int {
	if werr == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(werr, &ee) {
		return ee.ExitCode()
	}
	return 1
}
